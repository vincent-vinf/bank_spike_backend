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

func addSpike(context *gin.Context) {

}

func deleteSpike(context *gin.Context) {

}

func updateSpike(context *gin.Context) {

}
