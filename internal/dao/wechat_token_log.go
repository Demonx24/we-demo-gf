package dao

import (
	"context"
	"demo/internal/model/entity"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
)

var TokenLog = TokenLogDao{}

type TokenLogDao struct{}

func (d *TokenLogDao) SaveWechatTokenLog(ctx context.Context, tokenLog *entity.WechatTokenLog) error {
	if tokenLog == nil {
		return gerror.New("tokenLog 为空")
	}

	db := g.DB()

	// 使用 Save 并带 Where 条件，GF 会根据条件判断记录是否存在，存在则更新，不存在则插入
	_, err := db.Ctx(ctx).Model("wechat_token_log").
		Where("app_id", tokenLog.AppID).
		Data(tokenLog).
		Save()

	if err != nil {
		return err
	}

	return nil
}

func (d *shopSetDao) GetAppIdByShopId(ctx context.Context, shopid string) (string, error) {
	var appId string
	err := g.Model("wechat_shop").
		Fields("app_id").
		Where("shop_id = ?", shopid).
		Scan(&appId)
	if err != nil {
		return "", err
	}
	return appId, nil
}
