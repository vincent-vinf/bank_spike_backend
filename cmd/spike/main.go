package main

import (
	"bank_spike_backend/internal/access"
	"bank_spike_backend/internal/db"
	redisx "bank_spike_backend/internal/redis"
	"bank_spike_backend/internal/util"
	"bank_spike_backend/internal/util/config"
	jwtx "bank_spike_backend/internal/util/jwt"
	"flag"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
)

var (
	accessEndpoint string
	client         access.AccessClient
	port           int
)

func init() {
	flag.StringVar(&accessEndpoint, "access-endpoint", "127.0.0.1:8082", "")
	flag.IntVar(&port, "port", 8081, "")
	flag.Parse()
	config.InitViper()
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
	router.POST("/:id/:rand", spikeHandler)

	util.WatchSignalGrace(r, port)
}

func spikeHandler(c *gin.Context) {
	spikeId := c.Param("id")
	rand := c.Param("rand")
	pass, err := redisx.CheckUrl(c, spikeId, rand)
	if err != nil {
		c.JSON(500, gin.H{"error": "access server err"})
		log.Println(err)
		return
	}
	if !pass {
		c.JSON(404, gin.H{"message": "Page not found"})
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

	/// TODO(vincent)秒杀逻辑

	c.JSON(200, gin.H{
		"msg": accessible,
	})
}
