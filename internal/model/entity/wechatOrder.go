package entity

import "time"

type WechatOrder struct {
	ID              int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	ShopID          *int      `json:"shop_id"` // int 可空，用指针表示NULL
	OrderID         int64     `json:"order_id" gorm:"uniqueIndex"`
	Status          int       `json:"status"`
	CreateTime      int64     `json:"create_time"` // 存时间戳，int64
	UpdateTime      int64     `json:"update_time"`
	PayTime         int64     `json:"pay_time"`
	ProductID       string    `json:"product_id"`
	ProductTitle    string    `json:"product_title"`
	SkuID           string    `json:"sku_id"`
	SkuTitle        string    `json:"sku_title"`
	SkuCnt          int       `json:"sku_cnt"`
	RealPrice       int       `json:"real_price"`
	SalePrice       int       `json:"sale_price"`
	OrderPrice      int       `json:"order_price"`
	FinderName      string    `json:"finder_name"`
	AddTime         time.Time `json:"add_time" gorm:"autoCreateTime"`
	FinderStatus    int       `json:"finder_status"` // 0未洗，1正常，2有视频号但没人，3没视频号，4没订单数据
	UserID          int       `json:"user_id"`       // 业绩算谁
	AftersaleStatus int       `json:"aftersale_status"`
	AftersaleType   int       `json:"aftersale_type"` // 1退货，2退货退款
	OrderSkuID      int64     `json:"order_sku_id"`
}

func (WechatOrder) TableName() string {
	return "wechat_order"
}
