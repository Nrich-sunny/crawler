package main

import (
	"fmt"
	"github.com/Nrich-sunny/crawler/collect"
	"github.com/Nrich-sunny/crawler/proxy"
	"time"
)

func main() {
	//proxyURLs := []string{"http://127.0.0.1:8888", "http://127.0.0.1:8889"}
	proxyURLs := []string{"http://127.0.0.1:8888"}
	p, err := proxy.RoundRobinProxySwitcher(proxyURLs...)
	if err != nil {
		fmt.Println("RoundRobinProxySwitcher failed")
	}

	//url := "https://google.com"
	//url := "https://book.douban.com/subject/1007305/"
	url := "http://www.baidu.com"
	var fetcher collect.Fetcher = collect.BrowserFetch{
		Timeout: 3000 * time.Millisecond,
		Proxy:   p,
	}

	body, err := fetcher.Get(url)
	if err != nil {
		fmt.Printf("read content failed:%v\n", err)
		return
	}

	fmt.Println(string(body))

}
