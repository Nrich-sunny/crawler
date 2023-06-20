package main

import (
	"fmt"
	"github.com/Nrich-sunny/crawler/collect"
	"time"
)

func main() {
	url := "https://book.douban.com/subject/1007305/"

	var Fetcher collect.Fetcher = collect.BrowserFetch{
		time.Millisecond * 30000,
	}

	body, err := Fetcher.Get(url)
	if err != nil {
		fmt.Printf("read content failed:%v\n", err)
		return
	}

	fmt.Println(string(body))
}
