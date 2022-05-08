package main

import (
	"bank_spike_backend/internal/access"
	"bank_spike_backend/internal/db"
	redisx "bank_spike_backend/internal/redis"
	"bank_spike_backend/internal/util"
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
	router.POST("/", spikeHandlerFactory("1"))

	util.WatchSignalGrace(r, port)
}

// spikeHandlerFactory 为传入的活动制造handler
func spikeHandlerFactory(spikeId string) func(c *gin.Context) {
	return func(c *gin.Context) {
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
		c.JSON(200, gin.H{
			"msg": accessible,
		})
	}
}
