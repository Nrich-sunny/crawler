package engine

import (
	"github.com/Nrich-sunny/crawler/collect"
	"go.uber.org/zap"
)

type Crawler struct {
	out chan collect.ParseResult // 负责处理爬取后的数据
	options
}

type Scheduler interface {
	Schedule()                // 负责启动调度器
	Push(...*collect.Request) // 将请求放入到调度器中
	Pull() *collect.Request   // 从调度器中获取请求
}

type ScheduleEngine struct {
	requestCh chan *collect.Request
	workerCh  chan *collect.Request
	reqQueue  []*collect.Request
	Logger    *zap.Logger
}

func NewEngine(opts ...Option) *Crawler {
	options := defaultOptions
	for _, opt := range opts {
		opt(&options)
	}
	crawler := &Crawler{}
	out := make(chan collect.ParseResult)
	crawler.out = out
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
	for {
		var req *collect.Request
		var ch chan *collect.Request

		if len(s.reqQueue) > 0 {
			req = s.reqQueue[0]
			s.reqQueue = s.reqQueue[1:]
			ch = s.workerCh
		}
		select {
		case r := <-s.requestCh:
			s.reqQueue = append(s.reqQueue, r)
		case ch <- req:
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
		if err := r.Check(); err != nil { // 检查当前 request 是否已经达到最大深度限制
			crawler.Logger.Error("check failed")
			continue
		}
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
		crawler.out <- result
	}
}

func (crawler *Crawler) HandleResult() {
	for {
		select {
		case result := <-crawler.out:
			//for _, req := range result.Requests {
			//	crawler.requestCh <- req // 进一步要爬取的Requests列表
			//}
			for _, item := range result.Items {
				crawler.Logger.Sugar().Info("get result: ", item)
			}
		}
	}
}
