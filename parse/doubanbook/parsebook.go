package doubanbook

import (
	"github.com/Nrich-sunny/crawler/collect"
	"regexp"
	"strconv"
	"time"
)

var DoubanBookTask = &collect.Task{
	Property: collect.Property{
		Name:     "douban_book_list",
		WaitTime: 1 * time.Second,
		MaxDepth: 5,
		Cookie:   "bid=-UXUw--yL5g; dbcl2=\"214281202:q0BBm9YC2Yg\"; __yadk_uid=jigAbrEOKiwgbAaLUt0G3yPsvehXcvrs; push_noty_num=0; push_doumail_num=0; __utmz=30149280.1665849857.1.1.utmcsr=accounts.douban.com|utmccn=(referral)|utmcmd=referral|utmcct=/; __utmv=30149280.21428; ck=SAvm; _pk_ref.100001.8cb4=%5B%22%22%2C%22%22%2C1665925405%2C%22https%3A%2F%2Faccounts.douban.com%2F%22%5D; _pk_ses.100001.8cb4=*; __utma=30149280.2072705865.1665849857.1665849857.1665925407.2; __utmc=30149280; __utmt=1; __utmb=30149280.23.5.1665925419338; _pk_id.100001.8cb4=fc1581490bf2b70c.1665849856.2.1665925421.1665849856.",
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
	// FIXME: 在添加 limit 之前，临时减少抓取数量，防止被服务器封禁
	result.Requests = result.Requests[:1]
	return result, nil
}

const BookListRe = `<a.*?href="([^"]+)" title="([^"]+)"`

func ParseBookList(ctx *collect.Context) (collect.ParseResult, error) {
	re := regexp.MustCompile(BookListRe)
	matches := re.FindAllSubmatch(ctx.Body, -1)
	result := collect.ParseResult{}

	for _, m := range matches {
		req := &collect.Request{
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

	// FIXME: 在添加limit之前，临时减少抓取数量,防止被服务器封禁
	result.Requests = result.Requests[:1]

	return result, nil
}

var authorRe = regexp.MustCompile(`<span class="pl"> 作者</span>:[\d\D]*?<a.*?>([^<]+)</a>`)
var publicRe = regexp.MustCompile(`<span class="pl">出版社:</span>([^<]+)<br/>`)
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
