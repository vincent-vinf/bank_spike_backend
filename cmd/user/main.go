package main

import (
	"bank_spike_backend/internal/db"
	"bank_spike_backend/internal/util"
	"bank_spike_backend/internal/util/config"
	jwtx "bank_spike_backend/internal/util/jwt"
	"flag"
	"fmt"
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

	r := gin.New()
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

	router := r.Group("/users")
	router.POST("/login", authMiddleware.LoginHandler)
	router.POST("/register", registerHandler)
	// 一组需要验证的路由
	auth := router.Group("/auth")
	auth.Use(authMiddleware.MiddlewareFunc())
	// Refresh time can be longer than token timeout
	auth.GET("/refresh_token", authMiddleware.RefreshHandler)

	spike := router.Group("/spike")
	spike.Use(authMiddleware.MiddlewareFunc())
	spike.GET("/", getSpikeList)
	spike.GET("/:id", getSpikeById)

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

func registerHandler(c *gin.Context) {
	var registerForm jwtx.RegisterForm
	if err := c.ShouldBind(&registerForm); err != nil {
		c.JSON(400, gin.H{
			"error": "Bad request parameter",
		})
		return
	}

	isExist, err := db.IsExistPhone(registerForm.Phone)
	if err != nil {
		log.Println(err)
		c.JSON(500, gin.H{
			"error": "Server internal error",
		})
		return
	}
	if isExist {
		c.JSON(400, gin.H{
			"error": "phone already exists",
		})
		return
	}

	id, err := db.Register(registerForm.Username, registerForm.Phone, registerForm.Passwd, registerForm.IdNumber, registerForm.WorkStatus, registerForm.Age)
	if err != nil {
		log.Println(err)
		c.JSON(500, gin.H{
			"error": "Server internal error",
		})
		return
	}

	fmt.Printf("id (%v, %T)\n", id, id)

	c.JSON(200, gin.H{
		"token": jwtx.GenerateToken(id),
	})
}
