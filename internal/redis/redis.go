package redisx

import (
	"bank_spike_backend/internal/util/config"
	"context"
	"github.com/go-redis/redis/v8"
	"sync"
	"time"
)

var (
	rdb  *redis.Client
	once sync.Once
)

func getInstance() *redis.Client {
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
	return rdb
}

func Close() {
	if rdb != nil {
		_ = rdb.Close()
	}
}

func Set(ctx context.Context, key, value string, timeout time.Duration) {
	rdb := getInstance()
	rdb.Set(ctx, key, value, timeout)
}

func Get(ctx context.Context, key string) (string, error) {
	rdb := getInstance()
	return rdb.Get(ctx, key).Result()
}

//func SetNX(key,value string)  {
//	rdb := getInstance()
//	resp := rdb.SetNX(key, value, time.Second*5)
//	lockSuccess, err := resp.Result()
//}
