package master

import (
	"github.com/Nrich-sunny/crawler/cmd/worker"
	"github.com/Nrich-sunny/crawler/collect"
	"github.com/Nrich-sunny/crawler/log"
	"github.com/Nrich-sunny/crawler/master"
	"github.com/Nrich-sunny/crawler/proto/crawler"
	"github.com/go-micro/plugins/v4/config/encoder/toml"
	etcdReg "github.com/go-micro/plugins/v4/registry/etcd"
	gs "github.com/go-micro/plugins/v4/server/grpc"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/spf13/cobra"
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

var MasterCmd = &cobra.Command{
	Use:   "master",
	Short: "run master service.",
	Long:  "run master service.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		Run()
	},
}

var masterId string
var HTTPListenAddress string
var GRPCListenAddress string

func init() {
	MasterCmd.Flags().StringVar(&masterId, "id", "1", "set master id")
	MasterCmd.Flags().StringVar(&HTTPListenAddress, "http", ":8081", "set HTTP listen address")
	MasterCmd.Flags().StringVar(&GRPCListenAddress, "grpc", ":9091", "set GRPC listen address")
}

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

	// init tasks
	var tConfig []collect.TaskConfig
	if err := cfg.Get("Tasks").Scan(&tConfig); err != nil {
		logger.Error("init seed tasks,", zap.Error(err))
	}
	seeds := worker.ParseTaskConfig(logger, nil, nil, tConfig)

	// start master
	reg := etcdReg.NewRegistry(registry.Addrs(sConfig.RegistryAddress))
	m, err := master.New(
		masterId,
		master.WithLogger(logger.Named("master")),
		master.WithGRPCAddress(GRPCListenAddress),
		master.WithRegistryURL(sConfig.RegistryAddress),
		master.WithRegistry(reg),
		master.WithSeeds(seeds),
	)
	if err != nil {
		logger.Error("init master failed", zap.Error(err))
	}

	// start http proxy to GRPC
	go RunHTTPServer(sConfig)

	// start grpc server
	RunGRPCServer(m, logger, sConfig, reg)

}

func RunGRPCServer(masterService *master.Master, logger *zap.Logger, cfg ServerConfig, reg registry.Registry) {
	service := micro.NewService(
		micro.Server(gs.NewServer(
			server.Id(masterId),
		)),
		micro.Address(GRPCListenAddress),
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

	if err := crawler.RegisterCrawlerMasterHandler(service.Server(), masterService); err != nil {
		logger.Fatal("register handler failed", zap.Error(err))
	}

	if err := service.Run(); err != nil {
		logger.Fatal("grpc server stop", zap.Error(err))
	}
}

func RunHTTPServer(cfg ServerConfig) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	defer cancel()

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	if err := crawler.RegisterCrawlerMasterGwFromEndpoint(ctx, mux, GRPCListenAddress, opts); err != nil {
		zap.L().Fatal("Register backend grpc server endpoint failed", zap.Error(err))
	}

	zap.S().Debugf("start http server listening on %v proxy to grpc server;%v", HTTPListenAddress, GRPCListenAddress)
	if err := http.ListenAndServe(HTTPListenAddress, mux); err != nil {
		zap.L().Fatal("http listenAndServe failed", zap.Error(err))
	}
}

type ServerConfig struct {
	RegistryAddress  string
	RegisterTTL      int
	RegisterInterval int
	ClientTimeOut    int
	Name             string
}
