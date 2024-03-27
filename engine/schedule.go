package engine

import (
	"github.com/Nrich-sunny/crawler/collect"
	"github.com/Nrich-sunny/crawler/collector"
	"github.com/Nrich-sunny/crawler/parse/doubanbook"
	"github.com/Nrich-sunny/crawler/parse/doubangroup"
	"github.com/robertkrimen/otto"
	"go.uber.org/zap"
	"sync"
)

// Store 全局爬虫种类实例
var Store = &CrawlerStore{
	list: []*collect.Task{},          // 全局任务队列
	Hash: map[string]*collect.Task{}, // 全局任务哈希表
}

func init() {
	Store.Add(doubangroup.DoubangroupTask)
	Store.Add(doubanbook.DoubanBookTask)
	//Storage.Add(doubangroup.DoubangroupJSTask)
}

func GetFields(taskName string, ruleName string) []string {
	return Store.Hash[taskName].Rule.Trunk[ruleName].ItemFields
}

type CrawlerStore struct {
	list []*collect.Task          // 任务队列
	Hash map[string]*collect.Task // 任务哈希表， 任务名 -> 任务
}

func (c *CrawlerStore) Add(task *collect.Task) {
	c.Hash[task.Name] = task
	c.list = append(c.list, task)
}

// AddJsReqs 用于动态规则添加请求。
func AddJsReqs(jreqs []map[string]interface{}) []*collect.Request {
	reqs := make([]*collect.Request, 0)

	for _, jreq := range jreqs {
		req := &collect.Request{}
		u, ok := jreq["Url"].(string)
		if !ok {
			return nil
		}
		req.Url = u
		req.RuleName, _ = jreq["RuleName"].(string)
		req.Method, _ = jreq["Method"].(string)
		req.Priority, _ = jreq["Priority"].(int)
		reqs = append(reqs, req)
	}
	return reqs
}

// AddJsReq 用于动态规则添加请求。
func AddJsReq(jreq map[string]interface{}) []*collect.Request {
	reqs := make([]*collect.Request, 0)
	req := &collect.Request{}
	u, ok := jreq["Url"].(string)
	if !ok {
		return nil
	}
	req.Url = u
	req.RuleName, _ = jreq["RuleName"].(string)
	req.Method, _ = jreq["Method"].(string)
	req.Priority, _ = jreq["Priority"].(int)
	reqs = append(reqs, req)
	return reqs
}

// AddJsTask 初始化任务与规则
func (c *CrawlerStore) AddJsTask(m *collect.TaskModule) {
	task := &collect.Task{
		Property: m.Property,
	}

	task.Rule.Root = func() ([]*collect.Request, error) {
		vm := otto.New()
		vm.Set("AddJsReq", AddJsReqs)
		v, err := vm.Eval(m.Root)
		if err != nil {
			return nil, err
		}
		e, err := v.Export()
		if err != nil {
			return nil, err
		}
		return e.([]*collect.Request), nil
	}

	for _, r := range m.Rules {
		parseFunc := func(parse string) func(ctx *collect.Context) (collect.ParseResult, error) {
			return func(ctx *collect.Context) (collect.ParseResult, error) {
				vm := otto.New()
				vm.Set("ctx", ctx)
				v, err := vm.Eval(parse)
				if err != nil {
					return collect.ParseResult{}, err
				}
				e, err := v.Export()
				if err != nil {
					return collect.ParseResult{}, err
				}
				if e == nil {
					return collect.ParseResult{}, err
				}
				return e.(collect.ParseResult), err
			}
		}(r.ParseFunc)
		if task.Rule.Trunk == nil {
			task.Rule.Trunk = make(map[string]*collect.Rule, 0)
		}
		task.Rule.Trunk[r.Name] = &collect.Rule{
			ParseFunc: parseFunc,
		}
	}

	c.Hash[task.Name] = task
	c.list = append(c.list, task)
}

type Crawler struct {
	outCh       chan collect.ParseResult // 负责处理爬取后的数据
	Visited     map[string]bool
	VisitedLock sync.Mutex

	failures     map[string]*collect.Request // 失败请求id -> 失败请求
	failuresLock sync.Mutex

	options
}

type Scheduler interface {
	Schedule()                // 负责启动调度器
	Push(...*collect.Request) // 将请求放入到调度器中
	Pull() *collect.Request   // 从调度器中获取请求
}

type ScheduleEngine struct {
	requestCh   chan *collect.Request
	workerCh    chan *collect.Request
	priReqQueue []*collect.Request
	reqQueue    []*collect.Request
	Logger      *zap.Logger
}

func NewEngine(opts ...Option) *Crawler {
	options := defaultOptions
	for _, opt := range opts {
		opt(&options)
	}
	crawler := &Crawler{}
	crawler.Visited = make(map[string]bool, 100)
	crawler.outCh = make(chan collect.ParseResult)
	crawler.failures = make(map[string]*collect.Request)
	crawler.options = options
	return crawler
}

func NewSchedule() *ScheduleEngine {
	s := &ScheduleEngine{}
	requestCh := make(chan *collect.Request) // 负责接收请求
	workCh := make(chan *collect.Request)    // 负责分配任务
	s.requestCh = requestCh
	s.workerCh = workCh
	return s
}

// Schedule
/**
 * 调度的核心逻辑
 * 监听 requestCh，新的请求塞进 reqQueue 中;
 * 遍历 reqQueue 中 Request，塞进 workerCh 中。
 */
func (s *ScheduleEngine) Schedule() {
	var req *collect.Request
	var ch chan *collect.Request
	for {
		// 更高优先级的队列
		if req == nil && len(s.priReqQueue) > 0 {
			req = s.priReqQueue[0]
			s.priReqQueue = s.priReqQueue[1:]
			ch = s.workerCh
		}

		// 更低优先级的队列
		if req == nil && len(s.reqQueue) > 0 {
			req = s.reqQueue[0]
			s.reqQueue = s.reqQueue[1:]
			ch = s.workerCh
		}
		select {
		case r := <-s.requestCh:
			if r.Priority > 0 {
				s.priReqQueue = append(s.priReqQueue, r)
			} else {
				s.reqQueue = append(s.reqQueue, r)
			}
		case ch <- req:
			req = nil
			ch = nil
		}
	}
}

func (s *ScheduleEngine) Push(reqs ...*collect.Request) {
	for _, req := range reqs {
		s.requestCh <- req
	}
}

func (s *ScheduleEngine) Pull() *collect.Request {
	r := <-s.workerCh
	return r
}

//func (s *ScheduleEngine) Output() *collect.Request {
//	r := <-s.workerCh
//	return r
//}

func (crawler *Crawler) Run() {
	go crawler.Schedule()
	for i := 0; i < crawler.WorkCount; i++ {
		go crawler.CreateWork()
	}
	crawler.HandleResult()
}

func (crawler *Crawler) Schedule() {
	var reqs []*collect.Request
	for _, seed := range crawler.Seeds {
		task, ok := Store.Hash[seed.Name]
		if !ok {
			crawler.Logger.Debug("task not found", zap.String("task name", seed.Name))
		}
		task.Storage = seed.Storage
		// 获取初始化任务
		rootReqs, err := task.Rule.Root()
		if err != nil {
			crawler.Logger.Error("get root failed",
				zap.Error(err),
			)
			continue
		}
		reqs = append(reqs, rootReqs...)
	}
	go crawler.Scheduler.Schedule()
	go crawler.Scheduler.Push(reqs...)
}

func (crawler *Crawler) CreateWork() {
	for {
		r := crawler.Scheduler.Pull()
		// 检查当前 request 是否已经达到最大深度限制
		if err := r.Check(); err != nil {
			crawler.Logger.Error("check failed")
			continue
		}
		// 判断当前是否已经访问
		if crawler.HasVisited(r) {
			crawler.Logger.Debug("request has Visited", zap.String("url:", r.Url))
			continue
		}
		// 设置当前请求已被访问
		crawler.StoreVisited(r)

		body, err := crawler.Fetcher.Get(r)
		if err != nil {
			crawler.Logger.Error("can't fetch ", zap.Error(err))
			crawler.SetFailure(r)
			continue
		}

		// 获取当前任务对应的规则
		rule, ok := r.Task.Rule.Trunk[r.RuleName]
		if !ok {
			crawler.Logger.Error("rule not found", zap.String("rule name", r.RuleName))
			continue
		}
		// 内容解析
		result, err := rule.ParseFunc(&collect.Context{
			Body: body,
			Req:  r,
		})

		if err != nil {
			crawler.Logger.Error("ParseFunc failed ",
				zap.Error(err),
				zap.String("url", r.Url),
			)
			continue
		}
		// FIXME: 为啥要在创建请求任务的时候处理结果呢。。
		// 新的任务加入队列中
		if len(result.Requests) > 0 {
			go crawler.Scheduler.Push(result.Requests...)
		}
		crawler.outCh <- result
	}
}

func (crawler *Crawler) HandleResult() {
	for {
		select {
		case result := <-crawler.outCh:
			for _, item := range result.Items {
				switch d := item.(type) {
				case *collector.DataCell:
					name := d.GetTaskName()
					task := Store.Hash[name]
					task.Storage.Save(d)
				}
				crawler.Logger.Sugar().Info("get result: ", item)
			}
		}
	}
}

func (crawler *Crawler) HasVisited(r *collect.Request) bool {
	crawler.VisitedLock.Lock()
	defer crawler.VisitedLock.Unlock()
	unique := r.Unique()
	return crawler.Visited[unique]
}

func (crawler *Crawler) StoreVisited(reqs ...*collect.Request) {
	crawler.VisitedLock.Lock()
	defer crawler.VisitedLock.Unlock()

	for _, r := range reqs {
		unique := r.Unique()
		crawler.Visited[unique] = true
	}
}

func (crawler *Crawler) SetFailure(r *collect.Request) {
	if r.Reload && crawler.HasVisited(r) {
		// 首次失败时，再重新执行一次
		unique := r.Unique()
		delete(crawler.Visited, unique)
		r.Reload = false // 下次进来就非首次失败了，不是可重复的请求了
		crawler.Scheduler.Push(r)
		return
	}

	// 失败2次及以上，加载到失败队列中
	crawler.failuresLock.Lock()
	defer crawler.failuresLock.Unlock()
	crawler.failures[r.Unique()] = r // 加入失败队列
}
