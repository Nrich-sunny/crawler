package main

import (
	"context"
	"fmt"
	"github.com/Nrich-sunny/crawler/collect"
	"github.com/Nrich-sunny/crawler/engine"
	"github.com/Nrich-sunny/crawler/limiter"
	"github.com/Nrich-sunny/crawler/log"
	pb "github.com/Nrich-sunny/crawler/proto/greeter"
	"github.com/Nrich-sunny/crawler/proxy"
	etcdReg "github.com/go-micro/plugins/v4/registry/etcd"
	gs "github.com/go-micro/plugins/v4/server/grpc"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go-micro.dev/v4"
	"go-micro.dev/v4/registry"
	"go-micro.dev/v4/server"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"net/http"
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
	proxyURLs := []string{"http://127.0.0.1:8888"}
	p, err := proxy.RoundRobinProxySwitcher(proxyURLs...)
	if err != nil {
		logger.Error("RoundRobinProxySwitcher failed.")
		return
	}

	//// storage
	//var storage storage.Storage
	//storage, err = sqlstorage.New(
	//	sqlstorage.WithSqlUrl("root:root@tcp(127.0.0.1:3326)/crawler?charset=utf8"),
	//	sqlstorage.WithLogger(logger.Named("sqlDB")),
	//	sqlstorage.WithBatchCount(2),
	//)
	//if err != nil {
	//	logger.Error("create sqlstorage failed")
	//	return
	//}

	// fetcher
	var fetcher collect.Fetcher = &collect.BrowserFetch{
		Timeout: 3000 * time.Millisecond,
		Proxy:   p,
		Logger:  logger,
	}

	// speed limiter
	// 令牌产生速率：2秒一个，桶容量：1
	secondLimit := rate.NewLimiter(limiter.Per(1, 2*time.Second), 1)
	// 令牌产生速率：1分钟20个
	minuteLimit := rate.NewLimiter(limiter.Per(20, 1*time.Minute), 20)
	// 多层限速器
	multiLimiter := limiter.NewMultiLimiter(secondLimit, minuteLimit)

	// init tasks
	var seeds = make([]*collect.Task, 0, 1000) // 在一开始就要分配好切片的容量, 否则频繁地扩容会影响程序的性能
	seeds = append(seeds, &collect.Task{
		Property: collect.Property{
			Name: "douban_book_list",
		},
		Fetcher: fetcher,
		//Storage: storage,
		Limit: multiLimiter,
	})

	crawler := engine.NewEngine(
		engine.WithFetcher(fetcher),
		engine.WithLogger(logger),
		engine.WithWorkCount(5),
		engine.WithSeeds(seeds),
		engine.WithScheduler(engine.NewSchedule()),
	)

	// worker start
	go crawler.Run()

	// start http proxy to GRPC
	go handleHttp()

	// start grpc server
	reg := etcdReg.NewRegistry(
		registry.Addrs(":2379"),
	)
	service := micro.NewService(
		micro.Server(gs.NewServer(
			server.Id("1"),
		)),
		micro.Address(":9090"),
		micro.Registry(reg),
		micro.Name("go.micro.server.worker"),
	)
	service.Init()
	pb.RegisterGreeterHandler(service.Server(), new(Greeter))
	if err := service.Run(); err != nil {
		logger.Fatal("grpc server stop")
	}
}

type Greeter struct{}

func (g *Greeter) Hello(ctx context.Context, req *pb.Request, rsp *pb.Response) error {
	rsp.Greeting = "Hello " + req.Name
	return nil
}

func handleHttp() {
	ctx := context.Background()
	ctx, cancle := context.WithCancel(ctx)
	defer cancle()

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}
	err := pb.RegisterGreeterGwFromEndpoint(ctx, mux, "localhost:9090", opts)
	if err != nil {
		fmt.Println(err)
	}
	http.ListenAndServe(":8080", mux)
}
