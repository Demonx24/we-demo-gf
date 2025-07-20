package boot

import (
	"context"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/streadway/amqp"
	"log"
)

var MQConn *amqp.Connection
var MQChannel *amqp.Channel
var QueueName = "wx-demo" // 你的队列名，改成你项目需要的

// InitMQ 初始化 RabbitMQ 连接、频道，并声明队列
func InitMQ() {
	cfg := g.Cfg().MustGet(nil, "mq.url").String()
	conn, err := amqp.Dial(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}

	// 声明队列，确保队列存在
	_, err = ch.QueueDeclare(
		QueueName, // 队列名
		true,      // durable 持久化
		false,     // autoDelete 是否自动删除
		false,     // exclusive 是否排他
		false,     // noWait 是否阻塞
		nil,       // arguments 参数
	)
	if err != nil {
		log.Fatalf("Queue Declare failed: %v", err)
	}

	MQConn = conn
	MQChannel = ch

	log.Printf("RabbitMQ initialized, queue: %s", QueueName)
	// 注册关闭逻辑
	RegisterCloser(func(ctx context.Context) error {
		_ = MQChannel.Close()
		return MQConn.Close()
	})
}
