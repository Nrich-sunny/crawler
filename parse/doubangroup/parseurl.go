package doubangroup

import (
	"github.com/Nrich-sunny/crawler/collect"
	"log"
	"regexp"
)

const urlListRe = `(https://www.douban.com/group/topic/[0-9a-z]+/)"[^>]*>([^<]+)</a>`

// origin = `(<https://www.douban.com/group/topic/[0-9a-z]+/>)"[^>]*>([^<]+)</a>`
// origin_useful = `(https://www.douban.com/group/topic/[0-9a-z]+/)"[^>]*>([^<]+)</a>`
// my = `(https://www.douban.com/group/topic/[0-9a-z]+/)(\?_i=[0-9a-zA-Z]+)` // unuseful

func ParseURL(content []byte) collect.ParseResult {
	re := regexp.MustCompile(urlListRe)

	matches := re.FindAllSubmatch(content, -1) // 首页中所有符合规则的链接
	log.Println("matches: ")
	log.Println(len(matches))
	result := collect.ParseResult{} // 记录解析所得的结果

	for _, m := range matches {
		u := string(m[1])
		result.Requesrts = append(
			result.Requesrts, &collect.Request{
				Url: u,
				ParseFunc: func(c []byte) collect.ParseResult {
					return GetContent(c, u)
				},
			},
		)
	}

	return result
}

const ContentRe = `<div class="topic-content">[\s\S]*?阳台[\s\S]*?<div`

func GetContent(contents []byte, url string) collect.ParseResult {
	re := regexp.MustCompile(ContentRe)

	ok := re.Match(contents)
	if !ok {
		return collect.ParseResult{
			Items: []interface{}{},
		}
	}

	result := collect.ParseResult{
		Items: []interface{}{url},
	}

	return result
}
