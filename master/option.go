package master

import "go.uber.org/zap"

type options struct {
	logger      *zap.Logger
	registryURL string // 注册中心地址
	GRPCAddress string
}

var defaultOptions = options{
	logger: zap.NewNop(),
}

type Option func(opts *options)

func WithLogger(logger *zap.Logger) Option {
	return func(opts *options) {
		opts.logger = logger
	}
}

func WithRegistryURL(registryURL string) Option {
	return func(opts *options) {
		opts.registryURL = registryURL
	}
}

func WithGRPCAddress(GRPCAddress string) Option {
	return func(opts *options) {
		opts.GRPCAddress = GRPCAddress
	}
}
