package main

import (
	"github.com/Nrich-sunny/crawler/collect"
	"github.com/Nrich-sunny/crawler/collector"
	"github.com/Nrich-sunny/crawler/collector/sqlstorage"
	"github.com/Nrich-sunny/crawler/engine"
	"github.com/Nrich-sunny/crawler/log"
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
	//	return
	//}

	var storage collector.Storage
	storage, err := sqlstorage.New(
		sqlstorage.WithSqlUrl("root:123456@tcp(127.0.0.1:3326)/crawler?charset=utf8"),
		sqlstorage.WithLogger(logger.Named("sqlDB")),
		sqlstorage.WithBatchCount(2),
	)
	if err != nil {
		logger.Error("create sqlstorage failed")
		return
	}

	// douban cookies
	var seeds = make([]*collect.Task, 0, 1000) // 在一开始就要分配好切片的容量, 否则频繁地扩容会影响程序的性能
	seeds = append(seeds, &collect.Task{
		Property: collect.Property{
			Name: "douban_book_list",
		},
		Storage: storage,
	})

	var fetcher collect.Fetcher = &collect.BrowserFetch{
		Timeout: 3000 * time.Millisecond,
		//Proxy: p,
		Logger: logger,
	}

	crawler := engine.NewEngine(
		engine.WithFetcher(fetcher),
		engine.WithLogger(logger),
		engine.WithWorkCount(5),
		engine.WithSeeds(seeds),
		engine.WithScheduler(engine.NewSchedule()),
	)

	crawler.Run()
}
