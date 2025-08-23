package entity

import "time"

// WechatAftersale 售后单详情表模型
type WechatAftersale struct {
	ID               int64     `gorm:"column:id;primaryKey;autoIncrement"`
	AfterSaleOrderID int64     `gorm:"column:after_sale_order_id"` // 可空
	Status           string    `gorm:"column:status;size:100;default:'';not null"`
	OrderID          int64     `gorm:"column:order_id;default:0;not null"`
	CreateTime       int64     `gorm:"column:create_time;default:0;not null"`
	UpdateTime       int64     `gorm:"column:update_time;default:0;not null"`
	Reason           string    `gorm:"column:reason;size:100;default:'';not null"`
	ReasonText       string    `gorm:"column:reason_text;size:100;default:'';not null"`
	Type             string    `gorm:"column:type;size:100;default:'';not null"`
	CreatedAt        time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt        time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

// TableName 指定数据库表名
func (WechatAftersale) TableName() string {
	return "wechat_order_aftersale"
}
