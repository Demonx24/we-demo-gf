package entity

import "github.com/gogf/gf/v2/os/gtime"

type ShopSet struct {
	Id           int         `json:"id"           orm:"id,primary"`
	AppId        string      `json:"app_id"       orm:"app_id"`
	Cookie       string      `json:"cookie"       orm:"cookie"`
	BizMagic     string      `json:"biz_magic"    orm:"biz_magic"`
	CookieStatus int         `json:"cookie_status" orm:"cookie_status"`
	CookieTime   *gtime.Time `json:"cookie_time"  orm:"cookie_time"`
}
