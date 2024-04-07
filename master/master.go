package master

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Nrich-sunny/crawler/cmd/worker"
	proto "github.com/Nrich-sunny/crawler/proto/crawler"
	"github.com/bwmarrin/snowflake"
	"github.com/golang/protobuf/ptypes/empty"
	"go-micro.dev/v4/registry"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"net"
	"reflect"
	"sort"
	"strings"
	"sync/atomic"
	"time"
)

const (
	RESOURCEPATH = "/resources"
)

type Command int

const (
	MSGADD Command = iota
	MSGDELETE
)

// ResourceSpec 资源的规格
type ResourceSpec struct {
	ID           string
	Name         string // 资源名称（任务名称）
	AssignedNode string // 资源分配到的 Worker 节点: "{NodeID}|{NodeAddress}"
	CreationTime int64  // 资源创建时间
}

// WorkerNodeSpec 描述 Worker 节点的状态
type WorkerNodeSpec struct {
	Node    *registry.Node // Worker 节点的信息
	Payload int            // 节点的负载(当前有多少资源分配到了该节点)
}

// Message 用于 Master 和 Worker 之间通信的消息结构
type Message struct {
	Cmd   Command
	Specs []*ResourceSpec
}

// Master
// ID: 包含 Master 的序号、Master 的 IP 地址和监听的 GRPC 地址
type Master struct {
	ID        string                     // Master 的 ID
	ready     int32                      // 用于标记当前 Master 是否为 Leader
	leaderID  string                     // 记录当前集群中的 Leader
	workNodes map[string]*WorkerNodeSpec // 记录当前集群中所有的 Worker 节点的信息
	resources map[string]*ResourceSpec   // 记录当前集群中所有的资源(爬虫任务可以视为一种资源)
	IDGen     *snowflake.Node            // 生成全局唯一 ID
	etcdCli   *clientv3.Client           // etcd 客户端
	options
}

func New(id string, opts ...Option) (*Master, error) {
	m := &Master{}
	// 初始化 Master 的配置
	options := defaultOptions
	for _, opt := range opts {
		opt(&options)
	}
	m.options = options

	// 生成 Master 的 ID
	ipv4, err := getLocalIP()
	if err != nil {
		return nil, err
	}
	m.ID = genMasterID(id, ipv4, m.GRPCAddress)
	m.logger.Sugar().Debugln("master_id:", m.ID)

	// 初始化资源和 Worker 节点
	m.resources = make(map[string]*ResourceSpec)

	// 创建一个全局唯一 ID 生成器
	node, err := snowflake.NewNode(1)
	if err != nil {
		return nil, err
	}
	m.IDGen = node

	// 创建 etcd clientv3 客户端
	endpoints := []string{m.registryURL}
	cli, err := clientv3.New(clientv3.Config{Endpoints: endpoints})
	if err != nil {
		return nil, err
	}
	m.etcdCli = cli

	// 更新 Worker 节点信息
	m.updateWorkNodes()

	// 将 Seeds 中的任务写入 etcd
	m.AddSeed()

	// 启动 Master 的选主逻辑
	go m.Campaign()

	go m.HandleMsg()

	return m, nil
}

func (m *Master) HandleMsg() {
	msgCh := make(chan *Message)

	select {
	case msg := <-msgCh:
		switch msg.Cmd {
		case MSGADD:
			m.AddResources(msg.Specs)
		}
	}
}

// AddSeed 将 Seeds 中的任务写入 etcd
func (m *Master) AddSeed() {
	rs := make([]*ResourceSpec, 0, len(m.Seeds))
	for _, seed := range m.Seeds {
		resp, err := m.etcdCli.Get(context.Background(), getResourcePath(seed.Name), clientv3.WithPrefix(), clientv3.WithSerializable())
		if err != nil {
			m.logger.Error("get resource failed", zap.Error(err))
			continue
		}
		if len(resp.Kvs) == 0 {
			r := &ResourceSpec{
				Name: seed.Name,
			}
			rs = append(rs, r)
		}
	}
	// 将没有写入 etcd 的任务存储到 etcd
	m.AddResources(rs)
}

func (m *Master) AddResources(rs []*ResourceSpec) {
	for _, r := range rs {
		m.addResource(r)
	}
}

// AddResource 将资源写入 etcd，同时为其分配 Worker 节点
func (m *Master) addResource(r *ResourceSpec) (*WorkerNodeSpec, error) {
	// 资源各个字段填充
	r.ID = m.IDGen.Generate().String()
	// 为资源分配 Worker 节点
	nodeAssigned, err := m.Assign(r)
	if err != nil {
		m.logger.Error("assign worker failed", zap.Error(err))
		return nil, err
	}
	if nodeAssigned.Node == nil {
		m.logger.Error("no node to assign")
		return nil, errors.New("no node to assign")
	}
	r.AssignedNode = nodeAssigned.Node.Id + "|" + nodeAssigned.Node.Address

	r.CreationTime = time.Now().UnixNano()
	m.logger.Debug("add resource", zap.Any("specs", r))

	// 资源写入 etcd（存储到 etcd 中的 Value 需要是 string 类型，用 json 的序列化）
	_, err = m.etcdCli.Put(context.Background(), getResourcePath(r.Name), encode(r))
	if err != nil {
		m.logger.Error("put etcd failed", zap.Error(err))
		return nil, err
	}
	// 记录资源
	m.resources[r.Name] = r
	// 更新 Worker 节点的负载
	nodeAssigned.Payload++
	return nodeAssigned, nil
}

// AddResource 允许客户端添加资源
func (m *Master) AddResource(ctx context.Context, req *proto.ResourceSpec, resp *proto.WorkerNodeSpec) error {
	r := &ResourceSpec{
		Name: req.Name,
	}
	nodeSpec, err := m.addResource(r)
	if err != nil {
		return err
	}
	if nodeSpec != nil {
		resp.Id = nodeSpec.Node.Id
		resp.Address = nodeSpec.Node.Address
	}
	return nil
}

// DeleteResource 允许客户端删除资源
func (m *Master) DeleteResource(ctx context.Context, req *proto.ResourceSpec, empty *empty.Empty) error {
	r, ok := m.resources[req.Name]
	if ok {
		// 删除 etcd 中的资源
		if _, err := m.etcdCli.Delete(ctx, getResourcePath(r.Name)); err != nil {
			return err
		}
	}

	// 更新 Worker 节点的负载
	if r.AssignedNode != "" {
		nodeId, err := getNodeID(r.AssignedNode)
		if err != nil {
			return err
		}

		if nodeSpec, ok := m.workNodes[nodeId]; ok {
			nodeSpec.Payload -= 1
		}
	}
	return nil
}

// Assign 为资源分配 Worker 节点, 计算当前的资源应该被分配到哪个节点
// 最小负载法
func (m *Master) Assign(r *ResourceSpec) (*WorkerNodeSpec, error) {
	// 遍历所有节点，找到合适的 Worker 节点
	candidates := make([]*WorkerNodeSpec, 0, len(m.workNodes))
	for _, node := range m.workNodes {
		candidates = append(candidates, node)
	}

	// 根据负载对 Worker 队列进行排序
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Payload < candidates[j].Payload
	})
	// 选择负载最小的节点作为目标 Worker 节点
	if len(candidates) > 0 {
		return candidates[0], nil
	}

	return nil, errors.New("no worker nodes")
}

func (m *Master) reAssign() {
	// 需要重新分配 worker 节点的资源
	rs := make([]*ResourceSpec, 0, len(m.resources))

	for _, r := range m.resources {
		// 没有分配节点的资源, 记录
		if r.AssignedNode == "" {
			rs = append(rs, r)
			continue
		}

		nodeID, err := getNodeID(r.AssignedNode)
		if err != nil {
			m.logger.Error("get nodeId failed", zap.Error(err))
		}
		// 分配了节点的资源，但节点已经不存在了，记录
		if _, ok := m.workNodes[nodeID]; !ok {
			rs = append(rs, r)
		}
	}
	// 重新分配资源
	m.AddResources(rs)
}

func (m *Master) IsLeader() bool {
	return atomic.LoadInt32(&m.ready) != 0
}

// Campaign 分布式选主的核心逻辑
func (m *Master) Campaign() {
	session, err := concurrency.NewSession(m.etcdCli, concurrency.WithTTL(5))
	if err != nil {
		m.logger.Error("NewSession", zap.Error(err))
	}
	defer session.Close()

	// 创建一个新的 etcd 选举对象
	// 抢占到 "/resources/election" Key 的 Master 将变为 Leader
	election := concurrency.NewElection(session, "/crawler/election")
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
				m.logger.Info("master start change to leader")
				m.leaderID = m.ID
				if !m.IsLeader() {
					if err := m.BecomeLeader(); err != nil {
						m.logger.Error("BecomeLeader failed", zap.Error(err))
					}
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
			// 全量更新 Worker 节点信息
			m.updateWorkNodes()
			// 全量更新资源的状态
			if err := m.loadResource(); err != nil {
				m.logger.Error("loadResource failed", zap.Error(err))
			}
			// 重新分配资源
			m.reAssign()

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

func (m *Master) BecomeLeader() error {
	// 当 Master 成为新的 Leader 后，全量更新当前 Worker 的节点状态 和 资源的状态
	// 全量加载当前的 Worker 节点
	m.updateWorkNodes()
	// 全量获取 etcd 中当前最新的资源信息
	if err := m.loadResource(); err != nil {
		return fmt.Errorf("loadResource failed:%w", err)
	}

	// 重新分配资源
	m.reAssign()

	// 标记当前 Master 成为 Leader
	atomic.StoreInt32(&m.ready, 1)
	return nil
}

// loadResource 全量地获取一次 etcd 中当前最新的资源信息, 并把它保存到内存中
func (m *Master) loadResource() error {
	resp, err := m.etcdCli.Get(context.Background(), RESOURCEPATH, clientv3.WithPrefix(), clientv3.WithSerializable())
	if err != nil {
		return fmt.Errorf("etcd get failed")
	}

	// 保存资源信息
	resources := make(map[string]*ResourceSpec)
	for _, kv := range resp.Kvs {
		r, err := decode(kv.Value)
		if err == nil && r != nil {
			resources[r.Name] = r
		}
	}

	m.logger.Info("leader init load resource", zap.Int("lenth", len(m.resources)))
	m.resources = resources

	// 更新 Worker 节点的负载
	for _, r := range m.resources {
		if r.AssignedNode != "" {
			id, err := getNodeID(r.AssignedNode)
			if err != nil {
				m.logger.Error("getNodeID failed", zap.Error(err))
			}
			if node, ok := m.workNodes[id]; ok {
				node.Payload++
			}
		}
	}
	return nil
}

// updateWorkNodes 更新当前集群中的 Worker 节点信息
func (m *Master) updateWorkNodes() {
	services, err := m.registry.GetService(worker.ServiceName)
	if err != nil {
		m.logger.Error("get service ", zap.Error(err))
	}

	nodes := make(map[string]*WorkerNodeSpec)
	if len(services) > 0 {
		for _, spec := range services[0].Nodes {
			nodes[spec.Id] = &WorkerNodeSpec{
				Node: spec,
			}
		}
	}

	added, deleted, changed := workNodeDiff(m.workNodes, nodes)
	m.logger.Sugar().Info("worker joined: ", added, ", leaved: ", deleted, ", changed: ", changed)

	m.workNodes = nodes
}

func workNodeDiff(old map[string]*WorkerNodeSpec, new map[string]*WorkerNodeSpec) ([]string, []string, []string) {
	added := make([]string, 0)
	deleted := make([]string, 0)
	changed := make([]string, 0)
	for k, v := range new {
		if ov, ok := old[k]; ok {
			if !reflect.DeepEqual(v.Node, ov.Node) {
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

func getResourcePath(name string) string {
	return fmt.Sprintf("%s/%s", RESOURCEPATH, name)
}

func encode(s *ResourceSpec) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func decode(ds []byte) (*ResourceSpec, error) {
	var s *ResourceSpec
	err := json.Unmarshal(ds, &s)
	return s, err
}

func getNodeID(assigned string) (string, error) {
	node := strings.Split(assigned, "|")
	if len(node) < 2 {
		return "", errors.New("")
	}
	id := node[0]
	return id, nil
}
