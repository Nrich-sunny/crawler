package collect

type (
	TaskModule struct {
		Property
		Root  string       `json:"root_script"`
		Rules []RuleModule `json:"rule"`
	}

	RuleModule struct {
		Name      string `json:"name"`
		ParseFunc string `json:"parse_script"`
	}
)
