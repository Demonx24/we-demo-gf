package logic

import (
	"context"
	"encoding/json"
	"fmt"
	service "we-demo-gf/internal/service/message"
	"we-demo-gf/utility/mq"
)

type messageLogic struct{}

func init() {
	service.RegisterMessage(&messageLogic{})
}

func (l *messageLogic) Dispatch(ctx context.Context, event string, data interface{}) error {
	// 序列化消息体
	msgBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal data failed: %w", err)
	}
	// 发送到 MQ
	return mq.Send(event, msgBytes)
}
