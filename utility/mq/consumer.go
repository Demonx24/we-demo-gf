package mq

import (
	"context"
	"encoding/json"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/streadway/amqp"
)

// StartConsumer 开始消费消息
func StartConsumer(ctx context.Context) {
	msgs, err := MQChannel.Consume(
		"wx-demo", // queue name
		"",        // consumer
		true,      // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,
	)
	if err != nil {
		g.Log().Fatalf(ctx, "Failed to register consumer: %v", err)
	}
	for d := range msgs {
		HandleDelivery(ctx, &d)
	}
}

// HandleDelivery 处理来自 MQ 的消息
func HandleDelivery(ctx context.Context, d *amqp.Delivery) {
	var payload interface{}
	if err := json.Unmarshal(d.Body, &payload); err != nil {
		g.Log().Errorf(ctx, "Failed to unmarshal MQ message: %v", err)
		d.Nack(false, false)
		return
	}
	// 业务处理: 这里只做简单打印
	g.Log().Infof(ctx, "Received message: %s", string(d.Body))
	d.Ack(false)
}
