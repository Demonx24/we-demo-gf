package dao

import (
	"context"
	"github.com/gogf/gf/v2/errors/gerror"

	"github.com/gogf/gf/v2/frame/g"
	"we-demo-gf/internal/model/entity"
)

var TokenLog = tokenLogDao{}

type tokenLogDao struct{}

// SaveOrUpdate 保存或更新 TokenLog，根据 app_id 判断唯一性
func (d *tokenLogDao) SaveOrUpdate(ctx context.Context, log *entity.WechatTokenLog) error {
	if log == nil {
		return gerror.New("TokenLog 为空")
	}

	// 使用 ON DUPLICATE KEY UPDATE 的方式（MySQL 特有）
	_, err := g.Model("wechat_token_log").
		Save(log) // Save 方法在有主键或唯一键时可自动执行插入或更新
	return err
}

// GetByAppId 根据 app_id 查询 token log
func (d *tokenLogDao) GetByAppId(ctx context.Context, appId string) (*entity.WechatTokenLog, error) {
	var log entity.WechatTokenLog
	err := g.Model("wechat_token_log").
		Where("app_id", appId).
		Scan(&log)
	if err != nil {
		return nil, err
	}
	return &log, nil
}
