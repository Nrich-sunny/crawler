package collect

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"time"
)

// Task 整个任务实例，所有请求共享的参数
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
	UniqueStr string
	Reload    bool   // 网站是否可以重复请求
	Url       string // 这里存的是单个请求对应的 url
	Method    string
	Depth     int                                // 该请求对应的深度
	Priority  int                                // 请求的优先级, 值越大优先级越高（目前只有两个优先级：0 和 大于0）
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

// 请求的唯一标识码
func (r *Request) Unique() string {
	block := md5.Sum([]byte(r.Url + r.Method))
	return hex.EncodeToString(block[:])
}
