package entity

type EventDatademo struct {
	ShopAppID    string     `json:"shop_app_id"`
	ToUserName   string     `json:"ToUserName"`
	FromUserName string     `json:"FromUserName"`
	CreateTime   int64      `json:"CreateTime"`
	MsgType      string     `json:"MsgType"`
	Event        string     `json:"Event"`
	Order_info   Order_info `json:"order_info"`
}
type EventData struct {
	ShopAppID    string `json:"shop_app_id"`
	ToUserName   string `json:"ToUserName"`
	FromUserName string `json:"FromUserName"`
	CreateTime   int64  `json:"CreateTime"`
	MsgType      string `json:"MsgType"`
	Event        string `json:"Event"`
}
type WaybillInfo struct {
	WaybillOrderID int64  `json:"ewaybill_order_id"`
	WaybillID      string `json:"waybill_id"`
	UpdateTime     int64  `json:"update_time"`
	Status         int    `json:"status"`
	Desc           string `json:"desc"`
}

type Order_info struct {
	EventData
	OrderInfo struct {
		Order_id        int64 `json:"order_id"`
		Cancel_type     int64 `json:"cancel_type"`
		Pay_time        int64 `json:"pay_time"`
		Finish_delivery int64 `json:"finish_delivery"`
		Confirm_type    int64 `json:"confirm_type"`
		Settle_time     int64 `json:"settle_time"`
		Type            int64 `json:"type"`
	} `json:"order_info"`
}

type EventBase struct {
	Event string `json:"Event"`
}
type ProductSpuListingEvent struct {
	EventData
	ProductSpuListing struct {
		ProductID string `json:"product_id"`
		Status    int    `json:"status"`
		Reason    string `json:"reason"`
	} `json:"ProductSpuListing"`
}
type FinderShopAftersaleStatusUpdateEvent struct {
	EventData
	FinderShopAftersaleStatusUpdate struct {
		Status                string  `json:"status"`
		AfterSaleOrderID      string  `json:"after_sale_order_id"`
		OrderID               string  `json:"order_id"`
		WxaVipDiscountedPrice float64 `json:"wxa_vip_discounted_price"`
	} `json:"finder_shop_aftersale_status_update"`
}
