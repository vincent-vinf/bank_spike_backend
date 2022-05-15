package main

import (
	"bank_spike_backend/internal/db"
	redisx "bank_spike_backend/internal/redis"
	"bank_spike_backend/internal/util"
	"bank_spike_backend/internal/util/config"
	jwtx "bank_spike_backend/internal/util/jwt"
	"flag"
	"github.com/gin-gonic/gin"
	"log"
)

var (
	port int
)

func init() {
	flag.IntVar(&port, "port", 8080, "")
	flag.Parse()
	config.InitViper()
}

func main() {
	defer db.Close()
	defer redisx.Close()

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	r.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"message": "Page not found"})
	})

	// 初始化JWT中间件
	authMiddleware, err := jwtx.GetAuthMiddleware(true)
	if err != nil {
		log.Fatal(err)
	}

	router := r.Group("/admin")
	router.POST("/login", authMiddleware.LoginHandler)
	// 一组需要验证的路由
	auth := router.Group("/auth")
	auth.Use(authMiddleware.MiddlewareFunc())
	// Refresh time can be longer than token timeout
	auth.GET("/refresh_token", authMiddleware.RefreshHandler)

	auth.POST("/spike", addSpike)
	auth.DELETE("/spike", deleteSpike)
	auth.PUT("/spike", updateSpike)

	util.WatchSignalGrace(r, port)
}

// buildSpikeWork 启动时检查所有正在进行的活动，若未设置randUrl则生成并设置
// 对于即将开始的秒杀活动插入时间队列
//func buildSpikeWork() error {
//	spikes, err := db.GetActiveSpike()
//	if err != nil {
//		return err
//	}
//
//	for _, spike := range spikes {
//		// 用锁的方式保证随机url只被生成&插入redis一次
//		_, err = redisx.SetNX(context.Background(), redisx.RandKey+spike.ID, getRandUrl(spike.ID), spike.EndTime.Sub(time.Now()))
//		if err != nil {
//			return err
//		}
//	}
//
//	//time.Sleep(time.Until(until))
//	return nil
//}

func addSpike(context *gin.Context) {

}

func deleteSpike(context *gin.Context) {

}

func updateSpike(context *gin.Context) {

}
