package entity

import "time"

type WechatMessage struct {
	ID         int       `json:"id" gorm:"primaryKey;autoIncrement"`
	ShopID     int       `json:"shop_id"`     // 商铺ID
	AppID      string    `json:"app_id"`      // 微信AppID
	Event      string    `json:"event"`       // 事件类型
	CreateTime int64     `json:"create_time"` // 消息生成时间戳（微信格式）
	Content    string    `json:"content"`     // 解密后的内容
	ContentRaw string    `json:"content_raw"` // 原始未解密内容
	Params     string    `json:"params"`      // 附带参数（可用于回调数据）
	Status     int       `json:"status"`      // 0未解密，1已解密，10已处理
	ErrorMsg   string    `json:"error_msg"`   // 错误信息记录
	CreatedAt  time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt  time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (WechatMessage) TableName() string {
	return "wechat_message"
}
