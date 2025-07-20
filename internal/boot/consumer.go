package boot

import (
	"context"
	"github.com/gogf/gf/v2/frame/g"
	"we-demo-gf/utility/mq"
)

// StartConsumer 启动 MQ 消费者
func StartConsumer(ctx context.Context) {
	// 根据需要声明队列，绑定交换机等
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
		g.Log().Fatal(ctx, "Failed to register consumer: %v", err)
	}
	// 启动 goroutine 处理消息
	go func() {
		for d := range msgs {
			mq.HandleDelivery(ctx, &d)
		}
	}()
	g.Log().Info(ctx, "RabbitMQ consumer started")
}
