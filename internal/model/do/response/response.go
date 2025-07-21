package response

type Response struct {
	Code int    `json:"code"`
	Data []Shop `json:"data"`
}

type Shop struct {
	ShopID          int    `json:"shop_id"`
	AftersaleNum    string `json:"aftersale_num"`
	WaitCollectNum  string `json:"wait_collect_num"`
	ExpressErrorNum string `json:"express_error_num"`
}
type ResponsePendingShipOrderCnt struct {
	Shop_id                    string `json:"shop_id"`
	PendingShipOrderCnt        string `json:"aftersale_num"`
	WaitingReceivedCnt         string `json:"wait_collect_num"`
	LogisticsExceptionOrderCnt string `json:"express_error_num"` // 物流异常订单数量

}
type RespTypeA struct {
	TotalNum int `json:"totalNum"`
}
type RespTypeB struct {
	TotalNum int `json:"totalNum"`
}
type RespTypeC struct {
	TotalNum int `json:"totalNum"`
}
type CommonResp struct {
	TotalNum int `json:"totalNum"`
}

// 请求参数结构
type RequestTask[T any] struct {
	Url       string
	Body      any
	Headers   map[string]string
	ParseFunc func([]byte) (T, error)
	Result    T // 这里应该用泛型 T，而不是固定 CommonResp
	Err       error
}

type TaskMeta struct {
	OrderStatus   int
	WaybillStatus int
	ResultField   *string
}
type TaskInfo struct {
	Body      map[string]any
	SetResult func(int)
}
type TaskParamTemplate struct {
	OrderStatus   int
	WaybillStatus int
	OverTime      *int // 可选
	SetResult     func(int)
}
