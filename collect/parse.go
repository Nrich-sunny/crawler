package collect

// RuleTree 采集规则树
type RuleTree struct {
	Root  func() ([]*Request, error) // 根节点(执行入口)，用于生成爬虫的种子网站
	Trunk map[string]*Rule           // 规则哈希表，用于存储当前任务所有的规则，规则名 -> 具体规则
}

// Rule 采集规则节点
type Rule struct {
	ItemFields []string                            // 当前输出数据的字段名
	ParseFunc  func(*Context) (ParseResult, error) // 内容解析函数
}
