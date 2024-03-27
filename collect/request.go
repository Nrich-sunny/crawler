package collect

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"regexp"
	"time"
)

type Property struct {
	Name     string        `json:"name"` // 任务名称，应保证唯一性
	Url      string        `json:"url"`
	Cookie   string        `json:"cookie"`
	WaitTime time.Duration `json:"wait_time"`
	MaxDepth int           `json:"max_depth"`
}

// Task 整个任务实例，所有请求共享的参数
type Task struct {
	Property
	Rule RuleTree // 任务中的规则
}

type Context struct {
	Body []byte
	Req  *Request
}

// ParseJsReq 动态解析JS中的正则表达式
func (c *Context) ParseJsReq(name string, reg string) ParseResult {
	re := regexp.MustCompile(reg)

	matches := re.FindAllSubmatch(c.Body, -1)
	result := ParseResult{}

	for _, m := range matches {
		u := string(m[1])
		result.Requests = append(result.Requests, &Request{
			Method:   "GET",
			Task:     c.Req.Task,
			Url:      u,
			Depth:    c.Req.Depth + 1,
			RuleName: name,
		})
	}
	return result
}

// OutputJs 解析内容并输出结果
func (c *Context) OutputJs(reg string) ParseResult {
	re := regexp.MustCompile(reg)
	ok := re.Match(c.Body)
	if !ok {
		return ParseResult{
			Items: []interface{}{},
		}
	}
	result := ParseResult{
		Items: []interface{}{c.Req.Url},
	}
	return result
}

// Request 单个请求
type Request struct {
	Task      *Task
	UniqueStr string
	Reload    bool   // 网站是否可以重复请求
	Url       string // 这里存的是单个请求对应的 url
	Method    string
	Depth     int    // 该请求对应的深度
	Priority  int    // 请求的优先级, 值越大优先级越高（目前只有两个优先级：0 和 大于0）
	RuleName  string // 该请求对应的规则名
}

type ParseResult struct {
	Requests []*Request    // 用于进一步获取数据。进一步要爬取的 Requests 列表
	Items    []interface{} // 获取到的数据(类型：任意元素类型的切片)
}

func (r *Request) Check() error {
	if r.Depth > r.Task.Property.MaxDepth {
		return errors.New("max depth limit reached")
	}
	return nil
}

// 请求的唯一标识码
func (r *Request) Unique() string {
	block := md5.Sum([]byte(r.Url + r.Method))
	return hex.EncodeToString(block[:])
}
