package boot

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/streadway/amqp"
	"log"
)

var MQConn *amqp.Connection
var MQChannel *amqp.Channel

// InitMQ 初始化 RabbitMQ 连接和频道
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
	MQConn = conn
	MQChannel = ch
	log.Println("RabbitMQ initialized")
}
