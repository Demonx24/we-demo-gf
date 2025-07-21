package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type WechatTokentService struct{}

func (s *WechatTokentService) GetStableAccessToken(ctx context.Context, appID string) (string, error) {

	// 尝试从 Redis 读取
	redisShop, err := ServiceGroupApp.GetWechatShopFromRedis(appID)
	if err != nil {
		return redisShop.AccessToken, nil
	}

	//从数据库读取 appSecret
	shop, err := ServiceGroupApp.GetShopByAppID(appID)
	if err != nil {
		return "", err
	}
	body := map[string]string{
		"grant_type": "client_credential",
		"appid":      appID,
		"secret":     shop.AppSecret,
	}
	bodyData, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("failed to marshal body: %w", err)
	}
	// 构造请求 URL
	reqURL := "https://api.weixin.qq.com/cgi-bin/stable_token"

	// 创建 POST 请求
	req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(bodyData))
	if err != nil {
		return "", fmt.Errorf("http request failed: %w", err)
	}
	// 设置请求头
	req.Header.Set("Content-Type", "application/json")

	// 执行请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	// 解析返回内容
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// 检查是否成功获取 access_token
	// 获取 access_token
	accessToken, ok := result["access_token"].(string)
	if !ok {
		return "", fmt.Errorf("failed to get access_token from response")
	}

	// 获取 expires_in 并转换为 time.Time
	expiresIn, ok := result["expires_in"].(float64) // expires_in 是整数，JSON 解码后为 float64
	if !ok {
		return "", fmt.Errorf("failed to get expires_in from response")
	}

	// 计算过期时间
	expirationTime := time.Now().Add(time.Duration(expiresIn) * time.Second)

	shop.AccessToken = accessToken
	shop.AccessTokenExpires = &expirationTime
	// 4. 缓存到 Redis
	if err = ServiceGroupApp.SaveWechatShopToRedis(shop, int(expiresIn)); err != nil {
		return "", fmt.Errorf("warning: update DB token failed: %v\n", err)
	}
	wechatTokenLog := database.WechatTokenLog{
		AppID:       appID,
		AccessToken: shop.AccessToken,
		ExpiresIn:   int(expiresIn),
	}
	//保存到shop表中
	shop.AccessToken = accessToken
	shop.AccessTokenExpires = &expirationTime
	err = ServiceGroupApp.SaveOrUpdateShopByAppID(ctx, shop)
	if err != nil {
		log.Printf("保存shop失败：%v", err)
	}
	//保存到tokenlog表
	err = DB.SaveOrUpdateWechatTokenLog(context.Background(), &wechatTokenLog)
	if err != nil {
		log.Printf("保存token_log失败：%v", err)
	}
	return shop.AccessToken, nil
}
