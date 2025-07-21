package controller

import (
	"context"
	v1 "we-demo-gf/api/message/v1"
	service "we-demo-gf/internal/service/message"
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
