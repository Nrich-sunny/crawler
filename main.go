package main

import (
	"fmt"
	"github.com/Nrich-sunny/crawler/collect"
	"time"
)

func main() {
	proxyURLs := []string{}

	url := "https://book.douban.com/subject/1007305/"

	var fetcher collect.Fetcher = collect.BrowserFetch{
		Timeout: 3000 * time.Millisecond,
	}
	body, err := fetcher.Get(url)

	if err != nil {
		fmt.Printf("read content failed:%v\n", err)
		return
	}

	fmt.Println(string(body))

	//// 加载 HTML 文档
	//doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	//if err != nil {
	//	fmt.Printf("read content failed:%v\n", err)
	//}
	//
	//doc.Find("div.index_linecard__wJq_3 a[target=_blank] h2").Each(func(i int, s *goquery.Selection) {
	//	//  获取匹配标签中的文本
	//	title := s.Text()
	//	fmt.Printf("Review %d: %s\n", i, title)
	//})
}
