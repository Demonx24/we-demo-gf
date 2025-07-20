package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"go.uber.org/zap"
	"time"
	"wx-demo/global"
	"wx-demo/model/database"
	"wx-demo/model/request"
	"wx-demo/model/response"
	"wx-demo/utils"
)

var validEvents = map[string]struct{}{
	"channels_ec_order_ext_info_update": {},
	"channels_ec_order_settle":          {},
	"channels_ec_order_confirm":         {},
	"channels_ec_order_deliver":         {},
	"channels_ec_order_wait_shipping":   {},
	"channels_ec_order_pay":             {},
	"channels_ec_order_cancel":          {},
	"channels_ec_order_new":             {},
	"demo-wlc":                          {},
	"product_spu_listing":               {},
	"channels_ec_aftersale_update":      {},
}

type WxdemoController struct{}

// OrderCallbackHandlerdemo 处理测试回调，模拟异步任务分发
func (c *WxdemoController) OrderCallbackHandlerdemo(r *ghttp.Request) {
	var req database.EventDatademo
	if err := r.Parse(&req); err != nil {
		global.Log.Error("缺少必要的 req 参数", zap.Error(err))
		response.FailWithMessage("缺少必要的 req 参数", r)
		return
	}

	token := "testToken123"
	shopAppID := r.GetString("shop_app_id")

	var body database.Encrypt
	if err := r.Parse(&body); err != nil {
		global.Log.Error("请求体解析失败", zap.Error(err))
		response.FailWithMessage("请求体解析失败", r)
		return
	}
	encryptBody := body.Encrypt

	if !wxdemoService.VerifySignature(token, "text", "text", encryptBody, "text") {
		global.Log.Info("校验已完成")
	}

	utils.BuildEncryptedResponse(`{"demo_resp":"good luck"}`, "wx6c7fa8b3718353ca", "n41RIiidmwTFXRYvg2ojBUcDoMyfXa2rRFpyH8vIIYS", token, "text")
	global.Log.Info("回包已完成")

	// 构建任务
	task := database.EventDatademo{
		ShopAppID:    shopAppID,
		ToUserName:   req.ToUserName,
		FromUserName: req.FromUserName,
		CreateTime:   req.CreateTime,
		MsgType:      req.MsgType,
		Event:        req.Event,
		Order_info:   req.Order_info,
	}

	response.OkWithMessage("成功", r)
	if _, ok := validEvents[req.Event]; ok {
		taskJson, _ := json.Marshal(task)
		err := kafkaService.SendKafkaMessage(global.Config.Kafka.DiffTopic, req.Event, string(taskJson))
		if err != nil {
			global.Log.Error("任务派发失败")
			return
		}
		global.Log.Info("任务派发成功")
	}
}

// Get 处理 GET 请求，微信回调验证echostr
func (c *WxdemoController) Get(r *ghttp.Request) {
	var req request.WechatCallbackRequest
	if err := r.Parse(&req); err != nil {
		global.Log.Error("缺少必要的 req 参数", zap.Error(err))
		response.FailWithMessage("缺少必要的 req 参数", r)
		return
	}
	echostr := r.GetQueryString("echostr")
	shopAppID := r.GetString("shop_app_id")

	weshop, err := wechat_shopService.GetWechatShopFromRedis(shopAppID)
	if err != nil {
		weshop, err = wechat_shopService.GetShopByAppID(shopAppID)
		if err != nil {
			r.Response.WriteStatus(403)
			r.Response.WriteJson(g.Map{"errcode": 403, "errmsg": "获取商家信息失败"})
			return
		}
	}
	weshop.IsPush = 1
	if err := wechat_shopService.SaveOrUpdateShopByAppID(context.Background(), weshop); err != nil {
		r.Response.WriteStatus(403)
		r.Response.WriteJson(g.Map{"errcode": 403, "errmsg": "修改商品配置信息错误"})
		return
	}

	var body struct {
		Encrypt string `json:"Encrypt"`
	}
	if err := r.Parse(&body); err != nil {
		r.Response.WriteStatus(400)
		return
	}
	encryptBody := body.Encrypt
	if !wxdemoService.VerifySignature(weshop.Token, req.Timestamp, req.Nonce, encryptBody, req.Signature) {
		r.Response.WriteStatus(403)
		r.Response.WriteJson(g.Map{"errcode": 403, "errmsg": "签名无效"})
		return
	}
	r.Response.WriteStatus(200)
	r.Response.Write([]byte(echostr)) // 原样返回echostr
}

// OrderCallbackHandler 处理微信回调事件
func (c *WxdemoController) OrderCallbackHandler(r *ghttp.Request) {
	var req request.WechatCallbackRequest
	if err := r.Parse(&req); err != nil {
		global.Log.Error("缺少必要的 req 参数", zap.Error(err))
		response.FailWithMessage("缺少必要的 req 参数", r)
		return
	}
	shopAppID := r.GetString("shop_app_id")

	var body database.Encrypt
	if err := r.Parse(&body); err != nil {
		global.Log.Error("获取密文失败")
		r.Response.WriteStatus(403)
		r.Response.WriteJson(g.Map{"errcode": 403, "errmsg": "获取密文失败"})
		return
	}

	weshop, err := wechat_shopService.GetWechatShopFromRedis(shopAppID)
	if weshop == nil || err != nil {
		weshop, err = wechat_shopService.GetShopByAppID(shopAppID)
		if err != nil {
			r.Response.WriteStatus(403)
			r.Response.WriteJson(g.Map{"errcode": 403, "errmsg": "获取商家信息失败或商家不在服务范围内"})
			weshop = &database.WechatShop{
				AppID:     shopAppID,
				AppSecret: "",
			}
			_ = wechat_shopService.SaveWechatShopToRedis(weshop, 3600*8)
			return
		}
	}

	if weshop != nil {
		_ = wechat_shopService.SaveWechatShopToRedis(weshop, 3600*8)
	}
	if weshop == nil {
		global.Log.Error("weshop is nil")
		r.Response.WriteStatus(403)
		r.Response.WriteJson(g.Map{"error": "invalid shop info"})
		return
	}

	encryptBody := body.Encrypt
	if !wxdemoService.VerifySignature(weshop.Token, req.Timestamp, req.Nonce, encryptBody, req.MsgSignature) {
		global.Log.Error("签名无效")
		r.Response.WriteStatus(403)
		r.Response.WriteJson(g.Map{"errcode": 403, "errmsg": "签名无效"})
		return
	}

	personJSON, _ := json.Marshal(req)
	message := database.WechatMessage{
		ShopID:     0,
		AppID:      shopAppID,
		ContentRaw: encryptBody,
		Params:     fmt.Sprintf("shop_app_id:%s, %s", shopAppID, string(personJSON)),
		Status:     0,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := wechatMessageService.CreateMessage(&message); err != nil {
		global.Log.Error("保存事件信息错误", zap.Error(err))
		return
	}

	aesKey := weshop.AesKey
	decrypted, err := wxdemoService.DecryptWxMessage(encryptBody, aesKey, shopAppID)
	if err != nil {
		r.Response.WriteStatus(403)
		r.Response.WriteJson(g.Map{"errmsg": "解密失败"})
		return
	}

	var eventBase database.EventData
	if err := json.Unmarshal(decrypted, &eventBase); err != nil {
		r.Response.WriteStatus(403)
		r.Response.WriteJson(g.Map{"errmsg": "结构体解析失败"})
		return
	}

	response.WXMessage(r) // 按需加密返回（业务层自定义）

	eventType := eventBase.Event
	var eventData interface{}

	if _, ok := validEvents[eventBase.Event]; ok {
		obj := utils.GetEventStruct(eventType)
		if err := json.Unmarshal(decrypted, obj); err != nil {
			r.Response.WriteStatus(400)
			r.Response.WriteJson(g.Map{"errmsg": "结构体解析失败"})
			return
		}
		eventData = obj
	}

	switch v := eventData.(type) {
	case *database.ProductSpuListingEvent:
		fmt.Println("外部使用SPU商品ID：", v.ProductSpuListing.ProductID)
		taskJson, _ := json.Marshal(v)
		err = kafkaService.SendKafkaMessage(global.Config.Kafka.DiffTopic, v.Event, string(taskJson))
		if err != nil {
			global.Log.Error("任务派发失败")
			return
		}
		global.Log.Info("任务派发成功")

	case *database.Order_info:
		fmt.Println("外部使用订单ID：", v.OrderInfo.Order_id)
		if _, ok := validEvents[v.Event]; ok {
			taskJson, _ := json.Marshal(v)
			err = kafkaService.SendKafkaMessage(global.Config.Kafka.DiffTopic, v.Event, string(taskJson))
			if err != nil {
				global.Log.Error("任务派发失败")
				return
			}
			global.Log.Info("任务派发成功")
		}
	case *database.FinderShopAftersaleStatusUpdateEvent:
		fmt.Println("外部使用售后订单ID：", v.FinderShopAftersaleStatusUpdate.AfterSaleOrderID)
		taskJson, _ := json.Marshal(v)
		err = kafkaService.SendKafkaMessage(global.Config.Kafka.DiffTopic, v.Event, string(taskJson))
		if err != nil {
			global.Log.Error("任务派发失败")
			return
		}
		global.Log.Info("任务派发成功")
	default:
		fmt.Println("外部未知事件类型")
		global.Log.Error("外部未知事件类型")
	}

	fmt.Println("decrypted", string(decrypted))

	message.Event = eventBase.Event
	message.CreateTime = eventBase.CreateTime
	message.Content = string(decrypted)
	message.ContentRaw = encryptBody
	message.Status = 1

	if err := wechatMessageService.SaveMessage(&message); err != nil {
		global.Log.Error("保存事件信息错误", zap.Error(err))
		return
	}
}
