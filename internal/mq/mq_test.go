package mq

import (
	"bank_spike_backend/internal/pb/order"
	"bank_spike_backend/internal/util/config"
	"log"
	"testing"
)

func TestSend(t *testing.T) {
	config.InitViper()
	c := NewClient()
	defer c.Close()
	for i := 0; i < 10; i++ {
		o := order.OrderInfo{
			UserId:  "uid",
			SpikeId: "sid",
			Num:     int32(i),
		}
		err := c.Publish(&o)
		if err != nil {
			log.Fatalln(err)
		}
	}
}

func TestReceive(t *testing.T) {
	config.InitViper()
	c := NewClient()
	defer c.Close()
	ch := c.Consume()
	for d := range ch {
		log.Println(d)
	}
}
