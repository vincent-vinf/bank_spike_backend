package redisx

import (
	"bank_spike_backend/internal/util/config"
	"context"
	"github.com/go-redis/redis/v8"
	"sync"
	"time"
)

const (
	RandKey = "rand:"
)

var (
	rdb  *redis.Client
	once sync.Once
)

func getInstance() {
	if rdb == nil {
		once.Do(func() {
			cfg := config.GetConfig().Redis
			rdb = redis.NewClient(&redis.Options{
				Addr:     cfg.Endpoint,
				Password: cfg.Passwd,
				DB:       cfg.DB,
			})
		})
	}
}

func Close() {
	if rdb != nil {
		_ = rdb.Close()
	}
}

func Set(ctx context.Context, key, value string, timeout time.Duration) {
	getInstance()
	rdb.Set(ctx, key, value, timeout)
}

func SetNX(ctx context.Context, key, value string, timeout time.Duration) (bool, error) {
	resp := rdb.SetNX(ctx, key, value, timeout)
	return resp.Result()
}

func Get(ctx context.Context, key string) (string, error) {
	getInstance()
	return rdb.Get(ctx, key).Result()
}

// CheckUrl 根据活动id查询redis中的随机数, 与用户传入参数对比
func CheckUrl(ctx context.Context, spikeId, rand string) (bool, error) {
	r, err := Get(ctx, RandKey+spikeId)
	if err != nil {
		return false, err
	}
	if r == rand {
		return true, nil
	}
	return false, nil
}
