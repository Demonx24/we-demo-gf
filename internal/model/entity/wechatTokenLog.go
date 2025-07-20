package entity

import "time"

type WechatTokenLog struct {
	ID          int       `json:"id" gorm:"primaryKey;autoIncrement"`
	AppID       string    `json:"app_id"`
	AccessToken string    `json:"access_token"`
	ExpiresIn   int       `json:"expires_in"` // 有效期（秒）
	Content     string    `json:"content"`    // 记录获取过程或响应内容
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
}

func (WechatTokenLog) TableName() string {
	return "wechat_token_log"
}
