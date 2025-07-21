package service

import (
	"bytes"
	"context"
	dao "we-demo-gf/internal/dao/shop"
	"we-demo-gf/internal/model/do/response"

	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const wechatOrderSearchURL = "https://store.weixin.qq.com/shop-faas/mmchannelstradeorder/list/cgi/orderSearch?token=&lang=zh_CN"

// BatchQueryTotalNumByAppIds 控制最大并发90，批量查询订单数据
func BatchQueryTotalNumByAppIds(ctx context.Context, shopIds []string) ([]response.ResponsePendingShipOrderCnt, error) {
	maxConcurrency := 90
	sem := make(chan struct{}, maxConcurrency)

	var wg sync.WaitGroup
	var mu sync.Mutex
	results := make([]response.ResponsePendingShipOrderCnt, 0, len(shopIds))

	for _, shopId := range shopIds {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			res, err := QueryTotalNumByAppId(ctx, id)
			if err != nil {
				_ = dao.ShopSet.UpdateCookieStatusByShopId(id, 2)
				return
			}

			mu.Lock()
			results = append(results, res)
			mu.Unlock()
		}(shopId)
	}

	wg.Wait()
	return results, nil
}

// DoRequest 通用HTTP POST请求执行器
func DoRequest[T any](ctx context.Context, task *response.RequestTask[T], client *http.Client) {
	bodyBytes, err := json.Marshal(task.Body)
	if err != nil {
		task.Err = err
		return
	}

	req, err := http.NewRequestWithContext(ctx, "POST", task.Url, bytes.NewReader(bodyBytes))
	if err != nil {
		task.Err = err
		return
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range task.Headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		task.Err = err
		return
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		task.Err = err
		return
	}

	result, err := task.ParseFunc(respBytes)
	if err != nil {
		task.Err = err
		return
	}

	task.Result = result
}

// QueryTotalNumByAppId 查询单个shopId订单统计，三个状态并发查询
func QueryTotalNumByAppId(ctx context.Context, shopId string) (response.ResponsePendingShipOrderCnt, error) {
	var res response.ResponsePendingShipOrderCnt

	shopSet, err := dao.ShopSet.GetByShopId(shopId)
	if err != nil {
		return res, err
	}
	if shopSet == nil || strings.TrimSpace(shopSet.Cookie) == "" {
		return res, errors.New("cookie 为空")
	}

	re := regexp.MustCompile(`biz_magic=([^;]+)`)
	match := re.FindStringSubmatch(shopSet.Cookie)
	if len(match) < 2 {
		return res, errors.New("未找到 biz_magic")
	}
	shopSet.BizMagic = match[1]
	if strings.TrimSpace(shopSet.BizMagic) == "" {
		return res, errors.New("biz_magic 为空")
	}

	cleanCookie := strings.ReplaceAll(shopSet.Cookie, "\n", "")
	cleanCookie = strings.ReplaceAll(cleanCookie, "\r", "")
	cleanCookie = strings.TrimSpace(cleanCookie)

	headers := map[string]string{
		"Cookie":    cleanCookie,
		"biz_magic": shopSet.BizMagic,
	}

	client := &http.Client{Timeout: 5 * time.Second}
	baseBody := map[string]any{
		"pageSize":              30,
		"pageNum":               1,
		"onAftersaleOrderExist": 0,
	}
	taskTemplates := []response.TaskParamTemplate{
		{20, 1, ptrInt(0), func(v int) { res.PendingShipOrderCnt = strconv.Itoa(v) }},  // 待发货
		{30, 1, nil, func(v int) { res.WaitingReceivedCnt = strconv.Itoa(v) }},         // 等揽收
		{30, 3, nil, func(v int) { res.LogisticsExceptionOrderCnt = strconv.Itoa(v) }}, // 物流异常
	}

	taskParams := make([]response.TaskInfo, len(taskTemplates))
	for i, t := range taskTemplates {
		body := make(map[string]any)
		for k, v := range baseBody {
			body[k] = v
		}
		body["orderStatus"] = t.OrderStatus
		body["waybillStatus"] = t.WaybillStatus
		if t.OverTime != nil {
			body["overTime"] = *t.OverTime
		}
		taskParams[i] = response.TaskInfo{
			Body:      body,
			SetResult: t.SetResult,
		}
	}

	var wg sync.WaitGroup
	tasks := make([]*response.RequestTask[response.CommonResp], len(taskParams))

	for i, param := range taskParams {
		tasks[i] = NewRequestTask[response.CommonResp](wechatOrderSearchURL, param.Body, headers)
	}

	wg.Add(len(tasks))
	for i := range tasks {
		go func(i int) {
			defer wg.Done()
			DoRequest(ctx, tasks[i], client)
		}(i)
	}
	wg.Wait()

	for i, task := range tasks {
		if task.Err != nil {
			fmt.Printf("请求失败: %v\n", task.Err)
			continue
		}
		taskParams[i].SetResult(task.Result.TotalNum)
	}

	res.Shop_id = shopId
	return res, nil
}
func ptrInt(v int) *int { return &v }
func NewRequestTask[T any](url string, body map[string]any, headers map[string]string) *response.RequestTask[T] {
	return &response.RequestTask[T]{
		Url:     url,
		Body:    body,
		Headers: headers,
		ParseFunc: func(data []byte) (T, error) {
			var res T
			err := json.Unmarshal(data, &res)
			return res, err
		},
	}
}
