package main

import (
	"github.com/Nrich-sunny/crawler/collect"
	"github.com/Nrich-sunny/crawler/engine"
	"github.com/Nrich-sunny/crawler/limiter"
	"github.com/Nrich-sunny/crawler/log"
	"github.com/Nrich-sunny/crawler/storage"
	"github.com/Nrich-sunny/crawler/storage/sqlstorage"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/time/rate"
	"time"
)

func main() {
	// log
	plugin := log.NewStdoutPlugin(zapcore.DebugLevel) // 日志级别现在是写死的，后续放入配置文件
	logger := log.NewLogger(plugin)
	logger.Info("log init end")

	// set zap global logger
	zap.ReplaceGlobals(logger)

	// proxy
	//proxyURLs := []string{"http://127.0.0.1:8888"}
	//p, err := proxy.RoundRobinProxySwitcher(proxyURLs...)
	//if err != nil {
	//	logger.Error("RoundRobinProxySwitcher failed.")
	//	return
	//}

	var storage storage.Storage
	storage, err := sqlstorage.New(
		sqlstorage.WithSqlUrl("root:123456@tcp(127.0.0.1:3326)/crawler?charset=utf8"),
		sqlstorage.WithLogger(logger.Named("sqlDB")),
		sqlstorage.WithBatchCount(2),
	)
	if err != nil {
		logger.Error("create sqlstorage failed")
		return
	}

	var fetcher collect.Fetcher = &collect.BrowserFetch{
		Timeout: 3000 * time.Millisecond,
		//Proxy: p,
		Logger: logger,
	}

	// 令牌产生速率：2秒一个，桶容量：1
	secondLimit := rate.NewLimiter(limiter.Per(1, 2*time.Second), 1)
	// 令牌产生速率：1分钟20个
	minuteLimit := rate.NewLimiter(limiter.Per(20, 1*time.Minute), 20)
	// 多层限速器
	multiLimiter := limiter.NewMultiLimiter(secondLimit, minuteLimit)

	// douban cookies
	var seeds = make([]*collect.Task, 0, 1000) // 在一开始就要分配好切片的容量, 否则频繁地扩容会影响程序的性能
	seeds = append(seeds, &collect.Task{
		Property: collect.Property{
			Name: "douban_book_list",
		},
		Fetcher: fetcher,
		Storage: storage,
		Limit:   multiLimiter,
	})

	crawler := engine.NewEngine(
		engine.WithFetcher(fetcher),
		engine.WithLogger(logger),
		engine.WithWorkCount(5),
		engine.WithSeeds(seeds),
		engine.WithScheduler(engine.NewSchedule()),
	)

	crawler.Run()
}
