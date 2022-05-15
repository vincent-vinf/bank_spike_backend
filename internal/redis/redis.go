package redisx

import (
	"bank_spike_backend/internal/util/config"
	"context"
	"github.com/go-redis/redis/v8"
	"log"
	"strconv"
	"sync"
	"time"
)

const (
	RandKey       = "rand:"
	SpikeStoreKey = "spikeStore:"
	decLuaScript  = `
	if (redis.call('exists', KEYS[1]) == 1) then
    	local stock = redis.call('get', KEYS[1]);
    	if (stock - KEYS[2] >= 0) then
        	local leftStock = redis.call('DecrBy', KEYS[1], KEYS[2]);
        	return leftStock;
    	end;
    	return -1;
	end;
	return -1;
	`
)

var (
	rdb  *redis.Client
	once sync.Once

	decLuaHash string
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

			var err error
			decLuaHash, err = rdb.ScriptLoad(context.Background(), decLuaScript).Result()
			if err != nil {
				log.Fatalln("init lua:" + err.Error())
			}
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
	getInstance()
	resp := rdb.SetNX(ctx, key, value, timeout)
	return resp.Result()
}

func Get(ctx context.Context, key string) (string, error) {
	getInstance()
	return rdb.Get(ctx, key).Result()
}

func DecStore(ctx context.Context, key string, num int) (int, error) {
	getInstance()
	r, err := rdb.EvalSha(ctx, decLuaHash, []string{key, strconv.Itoa(num)}).Result()
	if err != nil {
		return 0, err
	}
	return int(r.(int64)), nil
}

// CheckUrl 根据活动id查询redis中的随机数, 与用户传入参数对比
func CheckUrl(ctx context.Context, spikeId, rand string) (bool, error) {
	getInstance()
	r, err := Get(ctx, RandKey+spikeId)
	if err == redis.Nil {
		return false, nil
	} else if err != nil {
		return false, err
	}
	if r == rand {
		return true, nil
	}
	return false, nil
}
