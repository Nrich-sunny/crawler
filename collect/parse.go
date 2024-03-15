package collect

// 采集规则树
type RuleTree struct {
	Root  func() []*Request
	Trunk map[string]*Rule
}

// 采集规则节点
type Rule struct {
	ParseFunc func([]byte, *Request) ParseResult
}
