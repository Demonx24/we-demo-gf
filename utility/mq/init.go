package mq

import (
	"context"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/streadway/amqp"
)

var (
	MQConn    *amqp.Connection
	MQChannel *amqp.Channel
)

// Init 初始化 MQ 连接并启动消费者
func Init(ctx context.Context) {
	cfg := g.Cfg().MustGet(nil, "mq.url").String()
	conn, err := amqp.Dial(cfg)
	if err != nil {
		g.Log().Fatalf(ctx, "Failed to connect to RabbitMQ: %v", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		g.Log().Fatalf(ctx, "Failed to open a channel: %v", err)
	}
	MQConn = conn
	MQChannel = ch
	g.Log().Info(ctx, "RabbitMQ initialized")

	// 启动消费者
	go StartConsumer(ctx)
}
