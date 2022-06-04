package main

import (
	"bank_spike_backend/internal/db"
	"bank_spike_backend/internal/mq"
	"bank_spike_backend/internal/orm"
	"bank_spike_backend/internal/pb/order"
	redisx "bank_spike_backend/internal/redis"
	"bank_spike_backend/internal/util"
	"bank_spike_backend/internal/util/config"
	jwtx "bank_spike_backend/internal/util/jwt"
	"context"
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

	router := r.Group("/order")
	router.Use(authMiddleware.MiddlewareFunc())

	router.GET("/all", listOrderHandler)
	router.GET("/:id", getOrderByIdHandler)
	router.POST("/pay/:id", payHandler)
	router.POST("/cancel/:id", cancelHandler)

	c := mq.NewClient()
	dealMqOrder(c.Consume())
	defer c.Close()

	util.WatchSignalGrace(r, port)
}

func payHandler(c *gin.Context) {
	orderId := c.Param("id")
	// ordered -> paid
	res, err := db.SetOrderState(orderId, orm.OrderPaid, orm.OrderOrdered)
	if err != nil || !res {
		log.Println(err)
		c.JSON(500, gin.H{"error": "pay order fail"})
		return
	}
	// 减库存
	success, err := db.DecreaseStock(orderId)
	if err != nil || !success {
		log.Println(err)
		c.JSON(500, gin.H{"error": "pay order fail"})
		return
	}
	c.JSON(200, gin.H{"status": "success", "msg": "payment successful"})
}

func cancelHandler(c *gin.Context) {
	t, _ := c.Get(jwtx.IdentityKey)
	user := t.(*jwtx.TokenUserInfo)

	orderId := c.Param("id")
	// ordered -> cancelled
	res, err := db.SetOrderState(orderId, orm.OrderCancelled, orm.OrderOrdered)
	if err != nil || !res {
		log.Println(err)
		c.JSON(500, gin.H{"error": "cancel order fail"})
		return
	}
	// 加库存
	success, err := db.IncreaseStock(orderId)
	if err != nil || !success {
		log.Println(err)
		c.JSON(500, gin.H{"error": "cancel order fail"})
		return
	}
	o, err := db.GetOrder(user.ID, orderId)
	if err != nil {
		log.Println(err)
		c.JSON(500, gin.H{"error": "cancel order fail"})
		return
	}
	_, err = redisx.DecStore(context.Background(), redisx.SpikeStoreKey+o.SpikeID, -1)
	if err != nil {
		log.Println(err)
		c.JSON(500, gin.H{"error": "cancel order fail"})
		return
	}

	c.JSON(200, gin.H{"status": "success", "msg": "cancel order successful"})
}

func getOrderByIdHandler(c *gin.Context) {
	t, _ := c.Get(jwtx.IdentityKey)
	user := t.(*jwtx.TokenUserInfo)

	orderId := c.Param("id")

	o, err := db.GetOrder(user.ID, orderId)
	if err != nil {
		log.Println(err)
		c.JSON(500, gin.H{"error": "get order fail"})
		return
	}
	if o == nil {
		c.JSON(404, gin.H{"error": "order not found"})
		return
	}
	c.JSON(200, o)
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
			// 判断订单是否存在
			o := &orm.Order{
				UserID:     info.UserId,
				SpikeID:    info.SpikeId,
				Quantity:   int(info.Quantity),
				State:      orm.OrderOrdered,
				CreateTime: time.Now(),
			}
			res, err := db.InsertOrderAffair(o)
			if err != nil || !res {
				if err != nil {
					log.Println(err)
				} else {
					log.Println("order already exists")
				}
				// 加库存
				lua, err := redisx.DecStore(context.Background(), redisx.SpikeStoreKey+info.SpikeId, -1)
				if err != nil {
					log.Println(err)
				}
				log.Println(lua)
				continue
			}
			log.Println("success create id: " + o.ID)
		}
	}()
}
