package doubangroup

import (
	"github.com/Nrich-sunny/crawler/collect"
	"log"
	"regexp"
)

const urlListRe = `(https://www.douban.com/group/topic/[0-9a-z]+/)"[^>]*>([^<]+)</a>`

var DoubangroupTask = &collect.Task{
	Property: collect.Property{
		Name:     "find_douban_sun_room",
		WaitTime: 2,
		Cookie:   "ll=\"118227\"; bid=ZYEsSfhim-4; viewed=\"1017143\"; gr_user_id=ab178b0e-82de-47c4-99fe-5221b995da91; __utmc=30149280; frodotk_db=\"ad691528e65057fd083b3f7ab2f107f7\"; __utmv=30149280.24935; dbcl2=\"249356040:nrPxmf/90qY\"; ck=WuR-; _pk_ref.100001.8cb4=%5B%22%22%2C%22%22%2C1689773168%2C%22https%3A%2F%2Faccounts.douban.com%2F%22%5D; _pk_id.100001.8cb4=56f26c8ed8708b6e.1669633904.; push_noty_num=0; push_doumail_num=0; __utma=30149280.1232258161.1670930004.1670930004.1689773169.2; __utmz=30149280.1689773169.2.1.utmcsr=accounts.douban.com|utmccn=(referral)|utmcmd=referral|utmcct=/; __yadk_uid=h928WDA73zU4Fnb7ttuaHTF5hnvU8yCl; douban-fav-remind=1; __gads=ID=b974d4f0d099827e-220633badad8009d:T=1670930004:RT=1689773653:S=ALNI_MZxUNCxXhloIXubkbdvrrwt2gWOUQ; __gpi=UID=00000b8f534707d0:T=1670930004:RT=1689773653:S=ALNI_MZ52YkdO2Wc_9jiLidVNsCOIplaWg",
		MaxDepth: 5,
	},
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
