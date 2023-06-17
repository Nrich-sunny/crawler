package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
	"io/ioutil"
	"net/http"
)

func main() {
	url := "https://www.thepaper.cn/"
	body, err := Fetch(url)

	if err != nil {
		fmt.Printf("read content failed:%v\n", err)
		return
	}

	// 加载 HTML 文档
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		fmt.Printf("read content failed:%v\n", err)
	}

	doc.Find("div.index_linecard__wJq_3 a[target=_blank] h2").Each(func(i int, s *goquery.Selection) {
		//  获取匹配标签中的文本
		title := s.Text()
		fmt.Printf("Review %d: %s\n", i, title)
	})
}

func Fetch(url string) ([]byte, error) {

	resp, err := http.Get(url)

	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error status code: %v\n", resp.StatusCode)
	}

	bodyReader := bufio.NewReader(resp.Body)
	e := DeterminEncoding(bodyReader)
	utf8Reader := transform.NewReader(bodyReader, e.NewDecoder())
	return ioutil.ReadAll(utf8Reader)

}

func DeterminEncoding(r *bufio.Reader) encoding.Encoding {

	bytes, err := r.Peek(1024)

	if err != nil {
		fmt.Printf("fetch error:%v\n", err)
		return unicode.UTF8
	}

	e, _, _ := charset.DetermineEncoding(bytes, "")
	return e
}
