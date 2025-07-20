package mq

import (
	"github.com/streadway/amqp"
)

// Send 发送消息到 RabbitMQ
func Send(routingKey string, body []byte) error {
	return MQChannel.Publish(
		"wx-demo",  // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
}
