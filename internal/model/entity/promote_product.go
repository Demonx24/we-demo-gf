package entity

import "time"

type Promote_product struct {
	ID          int       `gorm:"column:id;primaryKey;autoIncrement"`
	ShopID      *int      `gorm:"column:shop_id"`                                 // 可空
	AppID       string    `gorm:"column:app_id;size:100;default:'';not null"`     // 默认空字符串
	ProductID   string    `gorm:"column:product_id;size:100;default:'';not null"` // 默认空字符串
	Title       string    `gorm:"column:title;size:200;default:'';not null"`      // 默认空字符串
	ThumbImg    string    `gorm:"column:thumb_img;size:500;default:'';not null"`  // 默认空字符串
	Status      int       `gorm:"column:status;default:0;not null"`               // 默认 0
	RealPrice   int       `gorm:"column:real_price;default:0;not null"`           // 默认 0
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime"`               // 创建时自动填充
	BuildUserID int       `gorm:"column:build_user_id;default:0;not null"`        // 默认 0
	Memo        string    `gorm:"column:memo;size:50;default:'';not null"`        // 默认空字符串
}

// TableName 指定表名
func (Promote_product) TableName() string {
	return "promote_product"
}
