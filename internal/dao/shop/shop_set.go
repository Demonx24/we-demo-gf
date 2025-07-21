package dao

import (
	"github.com/gogf/gf/v2/frame/g"
	"we-demo-gf/internal/model/entity"
)

var ShopSet = shopSetDao{}

type shopSetDao struct{}

func (d *shopSetDao) GetByShopId(shopid string) (*entity.ShopSet, error) {
	var shopSet entity.ShopSet
	err := g.Model("wechat_shop_set").
		Where("shop_id", shopid).
		Scan(&shopSet)
	if err != nil {
		return nil, err
	}
	return &shopSet, nil
}
func (d *shopSetDao) GetAppIdByShopId(shopid string) (string, error) {
	var appId string
	err := g.Model("wechat_shop_set").
		Fields("app_id").
		Where("shop_id", shopid).
		Scan(&appId)
	if err != nil {
		return "", err
	}
	return appId, nil
}

func (d *shopSetDao) UpdateCookieStatusByShopId(shopid string, status int) error {
	_, err := g.Model("wechat_shop_set").
		Where("shop_id", shopid).
		Data(g.Map{"cookie_status": status}).
		Update()
	return err
}
