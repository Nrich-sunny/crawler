package collect

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"github.com/Nrich-sunny/crawler/storage"
	"math/rand"
	"regexp"
	"time"
)

type Context struct {
	Body []byte
	Req  *Request
}

func (c *Context) GetRule(ruleName string) *Rule {
	return c.Req.Task.Rule.Trunk[ruleName]
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

func (c *Context) Output(data interface{}) *storage.DataCell {
	res := &storage.DataCell{}
	res.Data = make(map[string]interface{})
	res.Data["Task"] = c.Req.Task.Name                          // 当前的任务名
	res.Data["Rule"] = c.Req.RuleName                           // 当前的规则名
	res.Data["Data"] = data                                     // 当前书籍的详细信息
	res.Data["Url"] = c.Req.Url                                 // 当前的网址
	res.Data["Time"] = time.Now().Format("2024-01-01 15:39:00") // 当前的时间
	return res
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
	TempData  *Temp  // 缓存临时数据供下一个阶段读取
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

func (r *Request) Fetch() ([]byte, error) {
	if err := r.Task.Limit.Wait(context.Background()); err != nil {
		return nil, err
	}
	// 随机休眠，模拟人类行为
	sleeptime := rand.Int63n(r.Task.WaitTime * 1000)
	time.Sleep(time.Duration(sleeptime) * time.Millisecond)
	return r.Task.Fetcher.Get(r)
}
