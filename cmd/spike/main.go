package main

import (
	"bank_spike_backend/internal/db"
	"bank_spike_backend/internal/mq"
	"bank_spike_backend/internal/orm"
	"bank_spike_backend/internal/pb/access"
	"bank_spike_backend/internal/pb/order"
	redisx "bank_spike_backend/internal/redis"
	"bank_spike_backend/internal/util"
	"bank_spike_backend/internal/util/config"
	jwtx "bank_spike_backend/internal/util/jwt"
	"bank_spike_backend/pkg/singleflight"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/patrickmn/go-cache"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"math/rand"
	"strconv"
	"time"
)

const (
	defaultMaxRequestNum = 1500
	defaultLimit         = 100
)

var (
	accessEndpoint string
	port           int

	client access.AccessClient
	loader = &singleflight.Group{}
	sender *mq.Client

	spikeUtilCache = cache.New(10*time.Minute, 20*time.Minute)
)

func init() {
	flag.StringVar(&accessEndpoint, "access-endpoint", "127.0.0.1:8082", "")
	flag.IntVar(&port, "port", 8081, "")
	flag.Parse()
	config.InitViper()
}

type spikeUtil struct {
	token   string
	limiter *rate.Limiter
}

func main() {
	defer db.Close()
	defer redisx.Close()
	conn, err := grpc.Dial(accessEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client = access.NewAccessClient(conn)

	r := gin.New()
	gin.SetMode(gin.ReleaseMode)
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	r.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"message": "Page not found"})
	})

	// 初始化JWT中间件
	authMiddleware, err := jwtx.GetAuthMiddleware(false)
	if err != nil {
		log.Fatal(err)
	}

	router := r.Group("/spike")
	router.Use(authMiddleware.MiddlewareFunc())
	router.GET("/:id", getRandHandler)
	router.POST("/:id/:rand", spikeHandler)

	sender = mq.NewClient()
	defer sender.Close()

	util.WatchSignalGrace(r, port)
}

func getRandHandler(c *gin.Context) {
	spikeId := c.Param("id")
	var randStr string
	var err error
	// 从本地获取
	if t, ok := spikeUtilCache.Get(spikeId); ok {
		randStr = t.(*spikeUtil).token
	} else {
		randStr, err = redisx.Get(c, redisx.RandKey+spikeId)
		su := &spikeUtil{
			token:   "",
			limiter: rate.NewLimiter(defaultLimit, defaultMaxRequestNum),
		}
		if err == redis.Nil {
			spike, err := loader.Do(spikeId, func() (interface{}, error) {
				spike, err := db.GetSpikeById(spikeId)
				if err != nil {
					return nil, err
				}
				return spike, nil
			})
			if err != nil {
				internalErr(c, err)
				return
			}
			s := spike.(*orm.Spike)
			if s == nil {
				c.JSON(404, gin.H{"error": "activity does not exist"})
				log.Println(err)
				return
			}
			now := time.Now()
			if now.Before(s.StartTime) || now.After(s.EndTime) {
				c.JSON(404, gin.H{"error": "not at activity time"})
				return
			}

			randStr = getRandUrl(s.ID)

			ok, err := redisx.SetNX(c, redisx.RandKey+s.ID, randStr, s.EndTime.Sub(time.Now()))
			if err != nil {
				internalErr(c, err)
				return
			}
			if !ok {
				randStr, err = redisx.Get(c, redisx.RandKey+spikeId)
				if err != nil {
					internalErr(c, err)
					return
				}
			}
			if s.Withholding*2 < defaultMaxRequestNum {
				su.limiter.SetBurst(s.Withholding * 2)
			}
		} else if err != nil {
			internalErr(c, err)
			return
		}
		su.token = randStr
		spikeUtilCache.Set(spikeId, su, time.Minute*20)
	}

	c.JSON(200, gin.H{"token": randStr})
	return
}

func spikeHandler(c *gin.Context) {
	spikeId := c.Param("id")
	r := c.Param("rand")
	var pass bool
	var su *spikeUtil
	if s, ok := spikeUtilCache.Get(spikeId); ok {
		su = s.(*spikeUtil)
		pass = r == su.token
	} else {
		randStr, err := redisx.Get(c, redisx.RandKey+spikeId)

		if err == redis.Nil {
			pass = false
		} else if err != nil {
			internalErr(c, err)
			return
		} else {
			pass = r == randStr
		}
	}

	if !pass {
		c.JSON(404, gin.H{"message": "page not found"})
		return
	}

	if su != nil {
		if !su.limiter.Allow() {
			c.JSON(503, gin.H{"status": "fail", "msg": "server is busy"})
		}
	}

	if getRestStock(c, spikeId) <= 0 {
		c.JSON(200, gin.H{"status": "fail", "msg": "sold out"})
		return
	}

	t, _ := c.Get(jwtx.IdentityKey)
	user := t.(*jwtx.TokenUserInfo)
	accessible, err := client.IsAccessible(c, &access.AccessReq{
		UserId:  user.ID,
		SpikeId: spikeId,
	})
	if err != nil {
		c.JSON(500, gin.H{
			"error": "access server err",
		})
		log.Println(err)
		return
	}

	if !accessible.Result {
		c.JSON(403, gin.H{"error": "no access: " + accessible.Reason})
		return
	}

	restStore, err := redisx.DecStore(c, redisx.SpikeStoreKey+spikeId, 1)
	if err != nil {
		internalErr(c, err)
		return
	}
	if restStore == -1 {
		c.JSON(200, gin.H{"status": "fail", "msg": "sold out"})
		return
	}

	err = sender.Publish(&order.OrderInfo{
		UserId:   user.ID,
		SpikeId:  spikeId,
		Quantity: 1,
	})
	if err != nil {
		internalErr(c, err)
		return
	}

	c.JSON(200, gin.H{"status": "success", "msg": "rest: " + strconv.Itoa(restStore)})
}

func getRestStock(ctx context.Context, spikeId string) (res int) {
	numStr, err := redisx.Get(ctx, redisx.SpikeStoreKey+spikeId)
	if err == redis.Nil {
		spike, err := loader.Do(spikeId, func() (interface{}, error) {
			spike, err := db.GetSpikeById(spikeId)
			if err != nil {
				return nil, err
			}
			return spike, nil
		})

		if err != nil {
			log.Println(err)
			return
		}
		s := spike.(*orm.Spike)
		numStr = strconv.Itoa(s.Withholding)
		ok, err := redisx.SetNX(ctx, redisx.SpikeStoreKey+spikeId, numStr, s.EndTime.Sub(time.Now()))
		if err != nil {
			log.Println(err)
			return
		}
		if !ok {
			numStr, err = redisx.Get(ctx, redisx.SpikeStoreKey+spikeId)
			if err != nil {
				log.Println(err)
				return
			}
		}
	} else if err != nil {
		log.Println(err)
		return
	}
	res, err = strconv.Atoi(numStr)
	if err != nil {
		log.Println(err)
	}
	return
}

func getRandUrl(spikeId string) string {
	rand.Seed(time.Now().Unix() + 928534782)
	token := make([]byte, 32)
	rand.Read(token)
	b := sha256.Sum256(append(token, []byte(config.GetConfig().Spike.RandUrlKey+spikeId)...))
	return hex.EncodeToString(b[:])
}

func internalErr(c *gin.Context, err error) {
	c.JSON(500, gin.H{"error": "spike server err"})
	log.Println(err)
}
