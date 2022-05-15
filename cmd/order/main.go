package main

import (
	"bank_spike_backend/internal/db"
	"bank_spike_backend/internal/mq"
	"bank_spike_backend/internal/orm"
	"bank_spike_backend/internal/pb/order"
	"bank_spike_backend/internal/util"
	"bank_spike_backend/internal/util/config"
	jwtx "bank_spike_backend/internal/util/jwt"
	"flag"
	"github.com/gin-gonic/gin"
	"log"
	"time"
)

var (
	port int
)

func init() {
	flag.IntVar(&port, "port", 8084, "")
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

	router := r.Group("/order")
	router.Use(authMiddleware.MiddlewareFunc())

	router.GET("", listOrderHandler)
	router.GET("/:id", getOrderByIdHandler)
	router.POST("/pay/:id", payHandler)

	c := mq.NewClient()
	dealMqOrder(c.Consume())
	defer c.Close()

	util.WatchSignalGrace(r, port)
}

func payHandler(c *gin.Context) {

}

func getOrderByIdHandler(c *gin.Context) {

}

func listOrderHandler(c *gin.Context) {
	t, _ := c.Get(jwtx.IdentityKey)
	user := t.(*jwtx.TokenUserInfo)
	list, err := db.GetOrderList(user.ID)
	if err != nil {
		log.Println(err)
		c.JSON(500, gin.H{"error": "get order list fail"})
		return
	}
	c.JSON(200, list)
}

func dealMqOrder(ch <-chan *order.OrderInfo) {
	go func() {
		for info := range ch {
			log.Println(info)
			o := &orm.Order{
				UserID:     info.UserId,
				SpikeID:    info.SpikeId,
				Quantity:   int(info.Quantity),
				State:      orm.OrderOrdered,
				CreateTime: time.Now(),
			}
			err := db.InsertOrder(o)
			if err != nil {
				log.Println(err)
			}
		}
	}()
}
