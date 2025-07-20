package service

import (
	"context"
)

type IMessage interface {
	Dispatch(ctx context.Context, event string, data interface{}) error
}

var localMessage IMessage

func Message() IMessage {
	return localMessage
}

func RegisterMessage(svc IMessage) {
	localMessage = svc
}
