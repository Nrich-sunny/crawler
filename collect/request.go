package collect

import "errors"

type Request struct {
	Url       string
	Cookie    string
	Depth     int                                // 任务的当前深度
	MaxDepth  int                                // 任务的最大深度
	ParseFunc func([]byte, *Request) ParseResult // 解析从网站获取到的网站信息的函数
}

type ParseResult struct {
	Requests []*Request    // 用于进一步获取数据。进一步要爬取的 Requests 列表
	Items    []interface{} // 获取到的数据(类型：任意元素类型的切片)
}

func (r *Request) Check() error {
	if r.Depth > r.MaxDepth {
		return errors.New("max depth limit reached")
	}
	return nil
}
