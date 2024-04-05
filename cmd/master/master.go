package master

import (
	"github.com/Nrich-sunny/crawler/log"
	pb "github.com/Nrich-sunny/crawler/proto/greeter"
	"github.com/go-micro/plugins/v4/config/encoder/toml"
	etcdReg "github.com/go-micro/plugins/v4/registry/etcd"
	gs "github.com/go-micro/plugins/v4/server/grpc"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go-micro.dev/v4"
	"go-micro.dev/v4/client"
	"go-micro.dev/v4/config"
	"go-micro.dev/v4/config/reader"
	"go-micro.dev/v4/config/reader/json"
	"go-micro.dev/v4/config/source"
	"go-micro.dev/v4/config/source/file"
	"go-micro.dev/v4/registry"
	"go-micro.dev/v4/server"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"net/http"
	"time"
)

func Run() {
	// load config
	enc := toml.NewEncoder()
	cfg, err := config.NewConfig(config.WithReader(json.NewReader(reader.WithEncoder(enc))))
	err = cfg.Load(file.NewSource(
		file.WithPath("config.toml"),
		source.WithEncoder(enc),
	))
	if err != nil {
		panic(err)
	}

	// log
	logText := cfg.Get("logLevel").String("INFO")
	logLevel, err := zapcore.ParseLevel(logText)
	if err != nil {
		panic(err)
	}
	plugin := log.NewStdoutPlugin(logLevel)
	logger := log.NewLogger(plugin)
	logger.Info("log init end")

	// set zap global logger
	zap.ReplaceGlobals(logger)

	var sConfig ServerConfig
	if err := cfg.Get("MasterServer").Scan(&sConfig); err != nil {
		logger.Error("get GRPC Server config failed", zap.Error(err))
	}
	logger.Sugar().Debugf("grpc server config,%+v", sConfig)

	// start http proxy to GRPC
	go RunHTTPServer(sConfig)

	// start grpc server
	RunGRPCServer(logger, sConfig)

}

func RunGRPCServer(logger *zap.Logger, cfg ServerConfig) {
	reg := etcdReg.NewRegistry(registry.Addrs(cfg.RegistryAddress))
	service := micro.NewService(
		micro.Server(gs.NewServer(
			server.Id(cfg.ID),
		)),
		micro.Address(cfg.GRPCListenAddress),
		micro.Registry(reg),
		micro.RegisterTTL(time.Duration(cfg.RegisterTTL)*time.Second),
		micro.RegisterInterval(time.Duration(cfg.RegisterInterval)*time.Second),
		micro.Name(cfg.Name),
	)

	// 设置micro 客户端默认超时时间为10秒钟
	if err := service.Client().Init(client.RequestTimeout(time.Duration(cfg.ClientTimeOut) * time.Second)); err != nil {
		logger.Sugar().Error("micro client init error. ", zap.String("error:", err.Error()))

		return
	}

	service.Init()

	if err := pb.RegisterGreeterHandler(service.Server(), new(Greeter)); err != nil {
		logger.Fatal("register handler failed")
	}

	if err := service.Run(); err != nil {
		logger.Fatal("grpc server stop")
	}
}

type Greeter struct{}

func (g *Greeter) Hello(ctx context.Context, req *pb.Request, rsp *pb.Response) error {
	rsp.Greeting = "Hello " + req.Name
	return nil
}

func RunHTTPServer(cfg ServerConfig) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	defer cancel()

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	if err := pb.RegisterGreeterGwFromEndpoint(ctx, mux, cfg.GRPCListenAddress, opts); err != nil {
		zap.L().Fatal("Register backend grpc server endpoint failed")
	}
	zap.S().Debugf("start http server listening on %v proxy to grpc server;%v", cfg.HTTPListenAddress, cfg.GRPCListenAddress)
	if err := http.ListenAndServe(cfg.HTTPListenAddress, mux); err != nil {
		zap.L().Fatal("http listenAndServe failed")
	}
}

type ServerConfig struct {
	HTTPListenAddress string
	GRPCListenAddress string
	ID                string
	RegistryAddress   string
	RegisterTTL       int
	RegisterInterval  int
	ClientTimeOut     int
	Name              string
}
