package engine

import (
	"github.com/Nrich-sunny/crawler/collect"
	"go.uber.org/zap"
)

type ScheduleEngine struct {
	requestCh chan *collect.Request
	workerCh  chan *collect.Request
	out       chan collect.ParseResult
	options
}

type Config struct {
	WorkCount int
	Fetcher   collect.Fetcher
	Logger    *zap.Logger
	Seeds     []*collect.Request
}

func NewSchedule(opts ...Option) *ScheduleEngine {
	options := defaultOptions
	for _, opt := range opts {
		opt(&options)
	}
	s := &ScheduleEngine{}
	s.options = options
	return s
}

func (s *ScheduleEngine) Run() {
	requestCh := make(chan *collect.Request) // 负责接收请求
	workerCh := make(chan *collect.Request)  // 负责分配任务
	out := make(chan collect.ParseResult)    // 负责处理爬取后的数据
	s.requestCh = requestCh
	s.workerCh = workerCh
	s.out = out
	go s.Schedule()
	for i := 0; i < s.WorkCount; i++ {
		go s.CreateWork()
	}
	s.HandleResult()
}

// Schedule
/**
 * 调度的核心逻辑
 * 监听 requestCh，新的请求塞进 reqQueue 中;
 * 遍历 reqQueue 中 Request，塞进 workerCh 中。
 */
func (s *ScheduleEngine) Schedule() {
	reqQueue := s.Seeds
	go func() {
		for {
			var req *collect.Request
			var ch chan *collect.Request

			if len(reqQueue) > 0 {
				req = reqQueue[0]
				reqQueue = reqQueue[1:]
				ch = s.workerCh
			}
			select {
			case r := <-s.requestCh:
				reqQueue = append(reqQueue, r)
			case ch <- req:
			}
		}
	}()
}

func (s *ScheduleEngine) CreateWork() {
	for {
		r := <-s.workerCh
		if err := r.Check(); err != nil {
			s.Logger.Error("check failed")
			continue
		}
		body, err := s.Fetcher.Get(r)
		if err != nil {
			s.Logger.Error("can't fetch ", zap.Error(err))
			continue
		}
		result := r.ParseFunc(body, r)
		s.out <- result
	}
}

func (s *ScheduleEngine) HandleResult() {
	for {
		select {
		case result := <-s.out:
			for _, req := range result.Requests {
				s.requestCh <- req // 进一步要爬取的Requests列表
			}
			for _, item := range result.Items {
				s.Logger.Sugar().Info("get result: ", item)
			}
		}
	}
}
