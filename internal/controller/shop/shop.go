package controller

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	service "we-demo-gf/internal/service/shop"
)

var reqData struct {
	ShopIds []string `json:"shop_ids"`
}

type ShopController struct{}

func (c *ShopController) POSTTotalNum(r *ghttp.Request) {
	if err := r.Parse(&reqData); err != nil {
		r.Response.WriteJsonExit(g.Map{
			"code": 400,
			"msg":  "参数解析错误: " + err.Error(),
		})
		return
	}

	req, err := service.BatchQueryTotalNumByAppIds(r.Context(), reqData.ShopIds)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": err.Error()})
		return
	}

	r.Response.WriteJsonExit(g.Map{
		"code": 0,
		"data": req,
	})
}
