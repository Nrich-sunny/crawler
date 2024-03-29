package limiter

import (
	"context"
	"golang.org/x/time/rate"
	"sort"
	"time"
)

// RateLimiter 对限速器的抽象，golang.org/x/time/rate实现的 Limiter 自动就实现了该接口
type RateLimiter interface {
	Wait(ctx context.Context) error
	Limit() rate.Limit
}

// MultiLimiter 多层限速器
type MultiLimiter struct {
	limiters []RateLimiter
}

func NewMultiLimiter(limiters ...RateLimiter) *MultiLimiter {
	byLimit := func(i, j int) bool {
		return limiters[i].Limit() < limiters[j].Limit()
	}
	// 将速率由小到大排序
	sort.Slice(limiters, byLimit)
	return &MultiLimiter{
		limiters: limiters,
	}
}

func (l *MultiLimiter) Wait(ctx context.Context) error {
	for _, l := range l.limiters {
		if err := l.Wait(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (l *MultiLimiter) Limit() rate.Limit {
	// 返回最小速率的限速器的速率
	return l.limiters[0].Limit()
}

// Per 每一个爬虫任务可能有不同的限速。Per 用来生成速率
func Per(eventCount int, duration time.Duration) rate.Limit {
	return rate.Every(duration / time.Duration(eventCount))
}
