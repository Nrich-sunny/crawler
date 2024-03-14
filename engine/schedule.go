package engine

import (
	"github.com/Nrich-sunny/crawler/collect"
	"go.uber.org/zap"
	"sync"
)

type Crawler struct {
	OutCh       chan collect.ParseResult // 负责处理爬取后的数据
	Visited     map[string]bool
	VisitedLock sync.Mutex
	Options
}

type Scheduler interface {
	Schedule()                // 负责启动调度器
	Push(...*collect.Request) // 将请求放入到调度器中
	Pull() *collect.Request   // 从调度器中获取请求
}

type ScheduleEngine struct {
	RequestCh chan *collect.Request
	WorkerCh  chan *collect.Request
	ReqQueue  []*collect.Request
	Logger    *zap.Logger
}

func NewEngine(opts ...Option) *Crawler {
	options := defaultOptions
	for _, opt := range opts {
		opt(&options)
	}
	crawler := &Crawler{}
	crawler.Visited = make(map[string]bool, 100)
	crawler.OutCh = make(chan collect.ParseResult)
	crawler.Options = options
	return crawler
}

func NewSchedule() *ScheduleEngine {
	s := &ScheduleEngine{}
	requestCh := make(chan *collect.Request) // 负责接收请求
	workCh := make(chan *collect.Request)    // 负责分配任务
	s.RequestCh = requestCh
	s.WorkerCh = workCh
	return s
}

// Schedule
/**
 * 调度的核心逻辑
 * 监听 RequestCh，新的请求塞进 ReqQueue 中;
 * 遍历 ReqQueue 中 Request，塞进 WorkerCh 中。
 */
func (s *ScheduleEngine) Schedule() {
	for {
		var req *collect.Request
		var ch chan *collect.Request

		if len(s.ReqQueue) > 0 {
			req = s.ReqQueue[0]
			s.ReqQueue = s.ReqQueue[1:]
			ch = s.WorkerCh
		}
		select {
		case r := <-s.RequestCh:
			s.ReqQueue = append(s.ReqQueue, r)
		case ch <- req:
		}
	}
}

func (s *ScheduleEngine) Push(reqs ...*collect.Request) {
	for _, req := range reqs {
		s.RequestCh <- req
	}
}

func (s *ScheduleEngine) Pull() *collect.Request {
	r := <-s.WorkerCh
	return r
}

//func (s *ScheduleEngine) Output() *collect.Request {
//	r := <-s.WorkerCh
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
		seed.RootReq.Task = seed
		seed.RootReq.Url = seed.Url
		reqs = append(reqs, seed.RootReq)
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
			continue
		}

		result := r.ParseFunc(body, r)
		// FIXME: 为啥要在创建请求任务的时候处理结果呢。。
		if len(result.Requests) > 0 {
			go crawler.Scheduler.Push(result.Requests...)
		}
		crawler.OutCh <- result
	}
}

func (crawler *Crawler) HandleResult() {
	for {
		select {
		case result := <-crawler.OutCh:
			//for _, req := range result.Requests {
			//	crawler.RequestCh <- req // 进一步要爬取的Requests列表
			//}
			for _, item := range result.Items {
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
