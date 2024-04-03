package doubangroup

import (
	"github.com/Nrich-sunny/crawler/collect"
	"log"
	"regexp"
)

const urlListRe = `(https://www.douban.com/group/topic/[0-9a-z]+/)"[^>]*>([^<]+)</a>`

var DoubangroupTask = &collect.Task{
	//Options: collect.Options{
	//	Name: "find_douban_sun_room",
	//},
	Rule: collect.RuleTree{
		Root: func() ([]*collect.Request, error) {
			var roots []*collect.Request
			for i := 0; i < 25; i += 25 {
				str := "https://www.douban.com/group/szsh/discussion?start=%d" + string(i)
				roots = append(roots, &collect.Request{
					Priority: 1,
					Url:      str,
					Method:   "GET",
					RuleName: "解析网站URL",
				})
			}
			return roots, nil
		},
		Trunk: map[string]*collect.Rule{
			"解析网站URL": {ParseFunc: ParseURL},
			"解析阳台房":   {ParseFunc: GetSunRoom},
		},
	},
}

func ParseURL(ctx *collect.Context) (collect.ParseResult, error) {
	re := regexp.MustCompile(urlListRe)

	matches := re.FindAllSubmatch(ctx.Body, -1) // 首页中所有符合规则的链接
	log.Println("matches: ")
	log.Println(len(matches))
	result := collect.ParseResult{} // 记录解析所得的结果

	for _, m := range matches {
		u := string(m[1])
		result.Requests = append(
			result.Requests, &collect.Request{
				Task:     ctx.Req.Task,
				Method:   "GET",
				Reload:   true,
				Url:      u,
				Depth:    ctx.Req.Depth + 1,
				RuleName: "解析阳台房",
			},
		)
	}

	return result, nil
}

const ContentRe = `<div class="topic-content">[\s\S]*?阳台[\s\S]*?<div`

func GetSunRoom(ctx *collect.Context) (collect.ParseResult, error) {
	re := regexp.MustCompile(ContentRe)

	ok := re.Match(ctx.Body)
	if !ok {
		return collect.ParseResult{
			Items: []interface{}{},
		}, nil
	}

	result := collect.ParseResult{
		Items: []interface{}{ctx.Req.Url},
	}

	return result, nil
}
