package mq

import (
	"bank_spike_backend/internal/pb/order"
	"bank_spike_backend/internal/util/config"
	"log"
	"testing"
	"time"
)

func TestSend(t *testing.T) {
	config.InitViper()
	c := NewClient()
	defer c.Close()
	for i := 0; i < 10; i++ {
		o := order.OrderInfo{
			UserId:   "uid",
			SpikeId:  "sid",
			Quantity: int64(i),
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
	ch := c.Consume()
	ti := time.After(time.Second * 6)
	go func() {
		for d := range ch {
			if d != nil {
				log.Println(d)
			} else {
				log.Println("nil")
			}
		}
	}()
	for {
		select {
		case <-ti:
			log.Println("timeout")
			c.Close()
			return
		}
	}

}

func TestReceive2(t *testing.T) {
	config.InitViper()
	c := NewClient()
	defer c.Close()
	ch := c.Consume()
	for d := range ch {
		if d != nil {
			log.Println(d)
		} else {
			log.Println("nil")
		}
	}
}
