package doubanbook

import (
	"github.com/Nrich-sunny/crawler/collect"
	"go.uber.org/zap"
	"regexp"
	"strconv"
)

var DoubanBookTask = &collect.Task{
	Options: collect.Options{
		Name: "douban_book_list",
	},
	Rule: collect.RuleTree{
		Root: func() ([]*collect.Request, error) {
			roots := []*collect.Request{
				{
					Priority: 1,
					Url:      "https://book.douban.com",
					Method:   "GET",
					RuleName: "数据tag",
				},
			}
			return roots, nil
		},
		Trunk: map[string]*collect.Rule{
			"数据tag": {ParseFunc: ParseTag},
			"书籍列表":  {ParseFunc: ParseBookList},
			"书籍简介": {
				ItemFields: []string{
					"书名",
					"作者",
					"页数",
					"出版社",
					"得分",
					"价格",
					"简介",
				},
				ParseFunc: ParseBookDetail,
			},
		},
	},
}

const TagRe = `<a href="([^"]+)" class="tag">([^<]+)</a>`

func ParseTag(ctx *collect.Context) (collect.ParseResult, error) {
	re := regexp.MustCompile(TagRe)
	matches := re.FindAllSubmatch(ctx.Body, -1)
	result := collect.ParseResult{}

	for _, m := range matches {
		result.Requests = append(result.Requests, &collect.Request{
			Method:   "GET",
			Task:     ctx.Req.Task,
			Url:      "https://book.douban.com" + string(m[1]),
			Depth:    ctx.Req.Depth + 1,
			RuleName: "书籍列表",
		})
	}
	zap.S().Debugln("parse book tag,count:", len(result.Requests))
	return result, nil
}

const BookListRe = `<a.*?href="([^"]+)" title="([^"]+)"`

func ParseBookList(ctx *collect.Context) (collect.ParseResult, error) {
	re := regexp.MustCompile(BookListRe)
	matches := re.FindAllSubmatch(ctx.Body, -1)
	result := collect.ParseResult{}

	for _, m := range matches {
		req := &collect.Request{
			Priority: 100,
			Method:   "GET",
			Task:     ctx.Req.Task,
			Url:      string(m[1]),
			Depth:    ctx.Req.Depth + 1,
			RuleName: "书籍简介",
		}
		req.TempData = &collect.Temp{}
		req.TempData.Set("book_name", string(m[2]))
		result.Requests = append(result.Requests, req)
	}
	zap.S().Debugln("parse book tag,count:", len(result.Requests))

	return result, nil
}

var authorRe = regexp.MustCompile(`<span class="pl"> 作者</span>:[\d\D]*?<a.*?>([^<]+)</a>`)
var publicRe = regexp.MustCompile(`<span class="pl">出版社:</span>[\d\D]*?<a.*?>([^<]+)</a>`)
var pageRe = regexp.MustCompile(`<span class="pl">页数:</span> ([^<]+)<br/>`)
var priceRe = regexp.MustCompile(`<span class="pl">定价:</span>([^<]+)<br/>`)
var scoreRe = regexp.MustCompile(`<strong class="ll rating_num " property="v:average">([^<]+)</strong>`)
var introRe = regexp.MustCompile(`<div class="intro">[\d\D]*?<p>([^<]+)</p></div>`)

func ParseBookDetail(ctx *collect.Context) (collect.ParseResult, error) {
	bookName := ctx.Req.TempData.Get("book_name")
	page, _ := strconv.Atoi(ExtractStr(ctx.Body, pageRe))

	book := map[string]interface{}{
		"书名":  bookName,
		"作者":  ExtractStr(ctx.Body, authorRe),
		"页数":  page,
		"出版社": ExtractStr(ctx.Body, publicRe),
		"得分":  ExtractStr(ctx.Body, scoreRe),
		"价格":  ExtractStr(ctx.Body, priceRe),
		"简介":  ExtractStr(ctx.Body, introRe),
	}
	data := ctx.Output(book)
	result := collect.ParseResult{
		Items: []interface{}{data},
	}
	return result, nil
}

func ExtractStr(contents []byte, re *regexp.Regexp) string {
	match := re.FindSubmatch(contents)
	if len(match) >= 2 {
		return string(match[1])
	} else {
		return ""
	}
}
