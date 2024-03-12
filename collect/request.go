package collect

import (
	"errors"
	"time"
)

// 整个任务示例，所有请求共享的参数
type Task struct {
	Url      string // 这里存的是一个 seed 对应的 url
	Cookie   string
	WaitTime time.Duration
	MaxDepth int      // 任务的最大深度
	RootReq  *Request // 任务中的第一个请求
}

// Request 单个请求
type Request struct {
	Task      *Task
	Url       string                             // 这里存的是单个请求对应的 url
	Depth     int                                // 该请求对应的深度
	ParseFunc func([]byte, *Request) ParseResult // 解析从网站获取到的网站信息的函数
}

type ParseResult struct {
	Requests []*Request    // 用于进一步获取数据。进一步要爬取的 Requests 列表
	Items    []interface{} // 获取到的数据(类型：任意元素类型的切片)
}

func (r *Request) Check() error {
	if r.Depth > r.Task.MaxDepth {
		return errors.New("max depth limit reached")
	}
	return nil
}
