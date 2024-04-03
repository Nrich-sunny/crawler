package collect

import (
	"github.com/Nrich-sunny/crawler/storage"
)

type Property struct {
	Name     string `json:"name"` // 任务名称，应保证唯一性
	Url      string `json:"url"`
	Cookie   string `json:"cookie"`
	WaitTime int64  `json:"wait_time"` // 随机休眠时间，秒
	MaxDepth int    `json:"max_depth"`
}

// Task 整个任务实例，所有请求共享的参数
type Task struct {
	Fetcher Fetcher
	Storage storage.Storage
	Rule    RuleTree // 任务中的规则
	Options
}

type TaskConfig struct {
	Name     string
	Cookie   string
	WaitTime int64
	Reload   bool
	MaxDepth int
	Fetcher  string
	Limits   []LimitConfig
}

type LimitConfig struct {
	EventCount int
	EventDur   int // 秒
	Bucket     int // 桶大小
}

func NewTask(opts ...Option) *Task {
	options := defaultOptions
	for _, opt := range opts {
		opt(&options)
	}

	t := &Task{}
	t.Options = options

	return t
}
