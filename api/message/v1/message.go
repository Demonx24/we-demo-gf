package v1

type DispatchMessageReq struct {
	Event string      `json:"event" v:"required"`
	Data  interface{} `json:"data"`
}

type DispatchMessageRes struct {
	Status string `json:"status"`
}
