package collect

type Request struct {
	Url       string
	Cookie    string
	ParseFunc func([]byte) ParseResult // 解析从网站获取到的网站信息的函数
}

type ParseResult struct {
	Requests []*Request    // 用于进一步获取数据。进一步要爬取的 Requests 列表
	Items    []interface{} // 获取到的数据(类型：任意元素类型的切片)
}
