package mq

import (
	"bank_spike_backend/internal/pb/order"
	"bank_spike_backend/internal/util/config"
	"fmt"
	"github.com/streadway/amqp"
	"google.golang.org/protobuf/proto"
	"log"
)

const (
	queueName = "orderRequire"
)

type client struct {
	conn *amqp.Connection
	ch   *amqp.Channel
	q    amqp.Queue
}

func NewClient() *client {
	s := &client{}
	cfg := config.GetConfig().RabbitMQ

	var err error
	s.conn, err = amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:%s/", cfg.User, cfg.Passwd, cfg.Host, cfg.Port))
	if err != nil {
		log.Fatalln(err)
	}

	s.ch, err = s.conn.Channel()
	if err != nil {
		log.Fatalln(err)
	}

	s.q, err = s.ch.QueueDeclare(
		queueName, // name
		false,     // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)

	if err != nil {
		log.Fatalln(err)
	}

	return s
}

func (c *client) Close() {
	if c.ch != nil {
		c.ch.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *client) Publish(info *order.OrderInfo) error {
	data, err := proto.Marshal(info)
	if err != nil {
		return err
	}
	return c.ch.Publish(
		"",       // exchange
		c.q.Name, // routing key
		false,    // mandatory
		false,    // immediate
		amqp.Publishing{
			ContentType: "application/octet-stream",
			Body:        data,
		})
}

func (c *client) Consume() <-chan *order.OrderInfo {
	msgs, err := c.ch.Consume(
		c.q.Name, // queue
		"",       // consumer
		true,     // auto-ack
		false,    // exclusive
		false,    // no-local
		false,    // no-wait
		nil,      // args
	)
	if err != nil {
		log.Fatalln(err)
	}

	ch := make(chan *order.OrderInfo, 1)

	go func() {
		for {
			select {
			case msg := <-msgs:
				info := &order.OrderInfo{}
				err = proto.Unmarshal(msg.Body, info)
				ch <- info
			}
		}
	}()

	return ch
}
