package main

import (
	"bank_spike_backend/internal/db"
	"bank_spike_backend/internal/orm"
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
	flag.IntVar(&port, "port", 8085, "")
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

	spike := router.Group("/spike")
	spike.Use(authMiddleware.MiddlewareFunc())
	spike.GET("/all", getSpikeList)
	spike.GET("/:id", getSpikeById)
	spike.POST("/add", addSpike)
	spike.DELETE("/:id", deleteSpike)
	spike.PUT("/:id", updateSpike)

	util.WatchSignalGrace(r, port)
}

func getSpikeById(context *gin.Context) {
	spikeId := context.Param("id")
	spike, err := db.GetSpikeById(spikeId)
	if err != nil {
		log.Println(err)
		context.JSON(500, gin.H{
			"error": "Server internal error",
		})
		return
	}

	context.JSON(200, spike)
}

func getSpikeList(context *gin.Context) {
	list, err := db.GetSpikeList()
	if err != nil {
		log.Println(err)
		context.JSON(500, gin.H{
			"error": "Server internal error",
		})
		return
	}

	context.JSON(200, gin.H{
		"list": list,
	})
}

func addSpike(context *gin.Context) {
	var spike orm.Spike
	if err := context.ShouldBind(&spike); err != nil {
		context.JSON(400, gin.H{
			"error": "Bad request parameter",
		})
		return
	}

	id, err := db.AddSpike(&spike)
	if err != nil {
		log.Println(err)
		context.JSON(500, gin.H{
			"error": "Server internal error",
		})
		return
	}

	context.JSON(200, gin.H{
		"id": id,
	})
}

func deleteSpike(context *gin.Context) {
	spikeId := context.Param("id")

	ok, err := db.DelSpike(spikeId)
	if err != nil {
		log.Println(err)
		context.JSON(500, gin.H{
			"error": "Server internal error",
		})
		return
	}

	context.JSON(200, gin.H{
		"ok": ok,
	})
}

func updateSpike(context *gin.Context) {
	spikeId := context.Param("id")
	var spike orm.Spike
	if err := context.ShouldBind(&spike); err != nil {
		context.JSON(400, gin.H{
			"error": "Bad request parameter",
		})
		return
	}

	ok, err := db.UpdateSpike(spikeId, &spike)
	if err != nil {
		log.Println(err)
		context.JSON(500, gin.H{
			"error": "Server internal error",
		})
		return
	}

	context.JSON(200, gin.H{
		"ok": ok,
	})

}
