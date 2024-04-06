package master

import (
	"errors"
	"github.com/Nrich-sunny/crawler/cmd/worker"
	"go-micro.dev/v4/registry"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"net"
	"reflect"
	"sync/atomic"
	"time"
)

// Master
// ID: 包含 Master 的序号、Master 的 IP 地址和监听的 GRPC 地址
type Master struct {
	ID        string                    // Master 的 ID
	ready     int32                     // 用于标记当前 Master 是否为 Leader
	leaderID  string                    // 记录当前集群中的 Leader
	workNodes map[string]*registry.Node // 记录当前集群中所有的 Worker 节点
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

func (m *Master) IsLeader() bool {
	return atomic.LoadInt32(&m.ready) != 0
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

	workerNodeChange := m.WatchWorker()

	for {
		select {
		// leaderCh 负责监听当前 Master 是否当上了 Leader
		case err := <-leaderCh:
			if err != nil {
				m.logger.Error("leader elect error", zap.Error(err))
				go m.elect(election, leaderCh)
				return
			} else {
				// 当选 Leader
				m.logger.Info("master change to leader")
				m.leaderID = m.ID
				if !m.IsLeader() {
					m.BecomeLeader()
				}
			}
		// leaderChange 负责监听当前集群中 Leader 是否发生了变化
		case resp := <-leaderChangeCh:
			if len(resp.Kvs) > 0 {
				m.logger.Info("watch leader change", zap.String("leader", string(resp.Kvs[0].Value)))
			}
		// workerNodeChange 负责监听当前集群中 Worker 节点的变化
		case resp := <-workerNodeChange:
			m.logger.Info("watch worker change", zap.Any("worker:", resp))
			m.updateNodes()
		case <-time.After(20 * time.Second):
			resp, err := election.Leader(context.Background())
			if err != nil {
				m.logger.Info("get Leader failed", zap.Error(err))
				if errors.Is(err, concurrency.ErrElectionNoLeader) {
					go m.elect(election, leaderCh)
				}
			}
			if resp != nil && len(resp.Kvs) > 0 {
				m.logger.Debug("get Leader", zap.String("value", string(resp.Kvs[0].Value)))
				if m.IsLeader() && m.ID != string(resp.Kvs[0].Value) {
					//当前已不再是leader
					atomic.StoreInt32(&m.ready, 0)
				}
			}
		}

	}
}

func (m *Master) elect(election *concurrency.Election, ch chan error) {
	// 阻塞直到选主成功
	err := election.Campaign(context.Background(), m.ID)
	ch <- err
}

// WatchWorker 监听 Worker 节点的信息，感知到 Worker 节点的注册与销毁
func (m *Master) WatchWorker() chan *registry.Result {
	// 监听 Worker 节点的变化
	watcher, err := m.registry.Watch(registry.WatchService(worker.ServiceName))
	if err != nil {
		panic(err)
	}
	ch := make(chan *registry.Result)
	go func() {
		for {
			res, err := watcher.Next() // 堵塞等待节点的下一个事件
			if err != nil {
				m.logger.Error("watch worker service failed", zap.Error(err))
				continue
			}
			// Master 收到节点变化事件，将事件发送到 workerNodeChange 通道
			ch <- res
		}
	}()

	return ch
}

func (m *Master) BecomeLeader() {
	atomic.StoreInt32(&m.ready, 1)
}

// updateNodes 更新当前集群中的 Worker 节点信息
func (m *Master) updateNodes() {
	services, err := m.registry.GetService(worker.ServiceName)
	if err != nil {
		m.logger.Error("get service ", zap.Error(err))
	}

	nodes := make(map[string]*registry.Node)
	if len(services) > 0 {
		for _, spec := range services[0].Nodes {
			nodes[spec.Id] = spec
		}
	}

	added, deleted, changed := workNodeDiff(m.workNodes, nodes)
	m.logger.Sugar().Info("worker joined: ", added, ", leaved: ", deleted, ", changed: ", changed)

	m.workNodes = nodes
}

func workNodeDiff(old map[string]*registry.Node, new map[string]*registry.Node) ([]string, []string, []string) {
	added := make([]string, 0)
	deleted := make([]string, 0)
	changed := make([]string, 0)
	for k, v := range new {
		if ov, ok := old[k]; ok {
			if !reflect.DeepEqual(v, ov) {
				changed = append(changed, k)
			}
		} else {
			added = append(added, k)
		}
	}
	for k := range old {
		if _, ok := new[k]; !ok {
			deleted = append(deleted, k)
		}
	}
	return added, deleted, changed
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
