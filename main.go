package main

import (
	"github.com/chromedp/chromedp"
	"golang.org/x/net/context"
	"log"
	"time"
)

func main() {
	// 1. 创建谷歌浏览器实例
	ctx, cancel := chromedp.NewContext(
		context.Background(),
	)
	defer cancel()

	// 2. 设置 context 超时时间
	ctx, cancel = context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// 3. 爬取页面，等待某一个元素出现,接着模拟鼠标点击，最后获取数据
	var example string
	err := chromedp.Run(ctx,
		chromedp.Navigate(`https://pkg.go.dev/time`),
		chromedp.WaitVisible(`body > footer`), // 代表正文已经加载完毕
		chromedp.Click(`#example-After`, chromedp.NodeVisible),
		chromedp.Value(`#example-After textarea`, &example),
	)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Go's time.After example:\\n%s", example)
}
