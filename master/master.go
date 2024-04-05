package master

import (
	"errors"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"net"
	"time"
)

// Master
// ID: 包含 Master 的序号、Master 的 IP 地址和监听的 GRPC 地址
type Master struct {
	ID string
	options
}

func New(id string, opts ...Option) (*Master, error) {
	m := &Master{}
	options := defaultOptions
	for _, opt := range opts {
		opt(&options)
	}
	m.options = options

	ipv4, err := getLocalIP()
	if err != nil {
		return nil, err
	}
	m.ID = genMasterID(id, ipv4, m.GRPCAddress)
	m.logger.Sugar().Debugln("master_id:", m.ID)
	go m.Campaign()

	return &Master{}, nil

}

// Campaign 分布式选主的核心逻辑
func (m *Master) Campaign() {
	// 创建一个 etcd clientv3 的客户端
	endpoints := []string{m.registryURL}
	cli, err := clientv3.New(clientv3.Config{Endpoints: endpoints})
	if err != nil {
		panic(err)
	}

	session, err := concurrency.NewSession(cli, concurrency.WithTTL(5))
	if err != nil {
		m.logger.Error("NewSession", zap.Error(err))
	}
	defer session.Close()

	// 创建一个新的 etcd 选举对象
	// 抢占到 "/resources/election" Key 的 Master 将变为 Leader
	election := concurrency.NewElection(session, "/resources/election")
	leaderCh := make(chan error)

	// 当前的 Master 进行 Leader 的选举，成功选举后取消阻塞
	go m.elect(election, leaderCh)

	// 监听 Leader 的变化
	leaderChangeCh := election.Observe(context.Background())
	select {
	case resp := <-leaderChangeCh:
		m.logger.Info("watch leader change", zap.String("leader", string(resp.Kvs[0].Value)))
	}

	for {
		select {
		// leaderCh 负责监听当前 Master 是否当上了 Leader
		case err := <-leaderCh:
			if err != nil {
				m.logger.Error("leader elect error", zap.Error(err))
				go m.elect(election, leaderCh)
				return
			} else {
				m.logger.Info("master change to leader")
			}
		// leaderChange 负责监听当前集群中 Leader 是否发生了变化
		case resp := <-leaderChangeCh:
			if len(resp.Kvs) > 0 {
				m.logger.Info("watch leader change", zap.String("leader", string(resp.Kvs[0].Value)))
			}
		case <-time.After(10 * time.Second):
			resp, err := election.Leader(context.Background())
			if err != nil {
				m.logger.Info("get Leader failed", zap.Error(err))
			}
			if resp != nil && len(resp.Kvs) > 0 {
				m.logger.Debug("get Leader", zap.String("value", string(resp.Kvs[0].Value)))
			}
		}

	}
}

func (m *Master) elect(election *concurrency.Election, ch chan error) {
	// 阻塞直到选主成功
	err := election.Campaign(context.Background(), m.ID)
	ch <- err
}

func genMasterID(id string, ipv4 string, GRPCAddress string) string {
	return "master" + id + "-" + ipv4 + GRPCAddress
}

// getLocalIP 获取本地网卡 IPv4 地址
func getLocalIP() (string, error) {
	var (
		addrs []net.Addr
		err   error
	)
	// 获取所有网卡地址
	if addrs, err = net.InterfaceAddrs(); err != nil {
		return "", err
	}
	// 取第一个非lo的网卡IP
	for _, addr := range addrs {
		if ipNet, isIpNet := addr.(*net.IPNet); isIpNet && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String(), nil
			}
		}
	}

	return "", errors.New("no local ip")
}
