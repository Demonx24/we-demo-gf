package entity

import "time"

type WechatShop struct {
	ID                 int        `json:"id" gorm:"primaryKey;autoIncrement"`
	Name               string     `json:"name" gorm:"default:''"` // 设置默认值为空字符串
	AppID              string     `json:"app_id" gorm:"uniqueIndex"`
	AppSecret          string     `json:"app_secret" gorm:"default:''"`   // 设置默认值为空字符串
	OldID              string     `json:"old_id" gorm:"default:''"`       // 设置默认值为空字符串
	Token              string     `json:"token" gorm:"default:''"`        // 设置默认值为空字符串
	AesKey             string     `json:"aes_key" gorm:"default:''"`      // 设置默认值为空字符串
	Status             int        `json:"status" gorm:"default:1"`        // 默认状态为 1
	SubjectType        string     `json:"subject_type" gorm:"default:''"` // 设置默认值为空字符串
	AddUserID          int        `json:"add_user_id" gorm:"default:0"`   // 默认值为 0
	CreatedAt          time.Time  `json:"created_at" gorm:"autoCreateTime"`
	AccessToken        string     `json:"access_token" gorm:"default:''"`           // 设置默认值为空字符串
	AccessTokenExpires *time.Time `json:"access_token_expires" gorm:"default:NULL"` // 默认值为 NULL
	HotMoney           int        `json:"hot_money" gorm:"default:0"`               // 默认值为 0
	SyncOrderTime      *time.Time `json:"sync_order_time" gorm:"default:NULL"`      // 默认值为 NULL
	SyncAftersaleTime  *time.Time `json:"sync_aftersale_time" gorm:"default:NULL"`  // 默认值为 NULL
	IsPush             int        `json:"is_push" gorm:"default:0"`                 // 默认值为 0
}

func (WechatShop) TableName() string {
	return "wechat_shop"
}
