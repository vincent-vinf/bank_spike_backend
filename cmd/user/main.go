package main

import (
	"bank_spike_backend/internal/db"
	"bank_spike_backend/internal/pb/access"
	redisx "bank_spike_backend/internal/redis"
	"bank_spike_backend/internal/util"
	"bank_spike_backend/internal/util/config"
	jwtx "bank_spike_backend/internal/util/jwt"
	"context"
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"time"
)

var (
	accessEndpoint string
	port           int
	client         access.AccessClient
)

func init() {
	flag.StringVar(&accessEndpoint, "access-endpoint", "127.0.0.1:8082", "")
	flag.IntVar(&port, "port", 8080, "")
	flag.Parse()
	config.InitViper()
}

func main() {
	defer db.Close()

	conn, err := grpc.Dial(accessEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	client = access.NewAccessClient(conn)

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
	spike.GET("/access/:id", accessHandler)

	util.WatchSignalGrace(r, port)
}

func getSpikeById(context *gin.Context) {
	spikeId := context.Param("id")
	spike, err := db.GetSpikeByIdUser(spikeId)
	if err != nil {
		log.Println(err)
		context.JSON(500, gin.H{
			"error": "Server internal error",
		})
		return
	}

	spike.Status = getSpikeStatus(context, spike.StartTime, spike.EndTime, spike.ID)

	context.JSON(200, spike)
}

func getSpikeList(context *gin.Context) {
	list, err := db.GetSpikeListUser()
	if err != nil {
		log.Println(err)
		context.JSON(500, gin.H{
			"error": "Server internal error",
		})
		return
	}

	for i, _ := range list {
		list[i].Status = getSpikeStatus(context, list[i].StartTime, list[i].EndTime, list[i].ID)
	}

	context.JSON(200, gin.H{
		"list": list,
	})
}

func getSpikeStatus(ctx context.Context, startTime, endTime time.Time, spikeId string) string {
	if startTime.After(time.Now()) {
		return "未开始"
	}
	if endTime.Before(time.Now()) {
		return "已结束"
	}
	numStr, err := redisx.Get(ctx, redisx.SpikeStoreKey+spikeId)
	if err != nil {
		return "进行中"
	}
	if numStr == "0" {
		return "已售罄"
	}
	return "进行中"
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

func accessHandler(c *gin.Context) {
	spikeId := c.Param("id")
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

	c.JSON(200, gin.H{"status": "success"})
}
