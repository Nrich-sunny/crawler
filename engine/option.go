package engine

import (
	"github.com/Nrich-sunny/crawler/collect"
	"go.uber.org/zap"
)

type Option func(opts *Options)

type Options struct {
	WorkCount int
	Fetcher   collect.Fetcher
	Logger    *zap.Logger
	Seeds     []*collect.Task
	Scheduler Scheduler
}

var defaultOptions = Options{
	Logger: zap.NewNop(),
}

func WithLogger(logger *zap.Logger) Option {
	return func(opts *Options) {
		opts.Logger = logger
	}
}

func WithFetcher(fetcher collect.Fetcher) Option {
	return func(opts *Options) {
		opts.Fetcher = fetcher
	}
}

func WithWorkCount(workCount int) Option {
	return func(opts *Options) {
		opts.WorkCount = workCount
	}
}

func WithSeeds(seed []*collect.Task) Option {
	return func(opts *Options) {
		opts.Seeds = seed
	}
}

func WithScheduler(schedule Scheduler) Option {
	return func(opts *Options) {
		opts.Scheduler = schedule
	}
}
