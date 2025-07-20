package controller

import (
	"context"
	"my-app/api/message/v1"
	"my-app/internal/service"
)

type ControllerMessage struct{}

// PostDispatch 发送消息接口
func (c *ControllerMessage) PostDispatch(ctx context.Context, req *v1.DispatchMessageReq) (res *v1.DispatchMessageRes, err error) {
	err = service.Message().Dispatch(ctx, req.Event, req.Data)
	if err != nil {
		return nil, err
	}
	res = &v1.DispatchMessageRes{Status: "ok"}
	return
}
