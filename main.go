package main

import (
	"fmt"
	"github.com/Nrich-sunny/crawler/collect"
	"github.com/Nrich-sunny/crawler/log"
	"github.com/Nrich-sunny/crawler/parse/doubangroup"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"time"
)

func main() {
	// log
	plugin := log.NewStdoutPlugin(zapcore.InfoLevel) // 日志级别现在是写死的，后续放入配置文件
	logger := log.NewLogger(plugin)
	logger.Info("log init end")

	// proxy
	//proxyURLs := []string{"http://127.0.0.1:8888"}
	//p, err := proxy.RoundRobinProxySwitcher(proxyURLs...)
	//if err != nil {
	//	logger.Error("RoundRobinProxySwitcher failed.")
	//}

	// douban cookies
	var seeds []*collect.Request
	for i := 0; i <= 25; i += 25 {
		str := fmt.Sprintf("https://www.douban.com/group/szsh/discussion?start=%d", i)
		seeds = append(seeds, &collect.Request{
			Url:       str,
			ParseFunc: doubangroup.ParseURL,
		})
	}
	logger.Info(seeds[0].Url)
	logger.Info(seeds[1].Url)

	var fetcher collect.Fetcher = &collect.BrowserFetch{
		Timeout: 3000 * time.Millisecond,
		//Proxy: p,
	}

	for len(seeds) > 0 {
		items := seeds
		seeds = nil
		for _, item := range items {
			body, err := fetcher.Get(item.Url)
			time.Sleep(3 * time.Second)
			if err != nil {
				logger.Error("read content failed.", zap.Error(err))
				continue
			}
			logger.Info("body = " + string(body))
			res := item.ParseFunc(body)
			for _, item := range res.Items {
				logger.Info("result", zap.String("get url:", item.(string)))
			}
			seeds = append(seeds, res.Requesrts...)
		}
	}

}
