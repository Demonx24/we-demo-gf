package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"sync"
	"time"
)

// WebSocket 相关结构与管理
type WSUser struct {
	Conn       *websocket.Conn
	Id         string // visitor_id 或 kefu_name
	Avator     string
	ToId       string // visitor's to_id or target
	Mux        sync.Mutex
	UpdateTime time.Time
}

var (
	visitorConn = make(map[string]*WSUser) // visitor_id -> conn
	kefuConn    = make(map[string]*WSUser) // kefu_name -> conn
	upgrader    = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	wsMsgChan = make(chan WSIncoming, 200)
)

type WSIncoming struct {
	Conn        *websocket.Conn
	Content     []byte
	MessageType int
	Ctx         *ghttp.Request
}

func StartWsBackend(db *sql.DB) {
	// 后端 message 处理 goroutine: 仅处理 ping（示例）等
	go func() {
		for msg := range wsMsgChan {
			var t map[string]interface{}
			if err := json.Unmarshal(msg.Content, &t); err != nil {
				continue
			}
			if ttype, ok := t["type"].(string); ok && ttype == "ping" {
				resp := map[string]string{"type": "pong"}
				b, _ := json.Marshal(resp)
				msg.Conn.WriteMessage(websocket.TextMessage, b)
			}
		}
	}()
}

// Handler: visitor websocket: /ws_visitor?visitor_id=xxx
func NewVisitorServer(r *ghttp.Request) {
	conn, err := upgrader.Upgrade(r.Response.Writer, r.Request, nil)
	if err != nil {
		r.Response.WriteStatusExit(400, err.Error())
		return
	}
	visitorId := r.Get("visitor_id").String()
	if visitorId == "" {
		visitorId = fmt.Sprintf("v%d", time.Now().UnixNano())
	}

	// create or find visitor in DB
	ctx := context.Background()
	v, err := FindOrCreateVisitor(ctx, db, visitorId)
	if err != nil {
		log.Println("FindOrCreateVisitor err:", err)
		conn.Close()
		return
	}

	user := &WSUser{Conn: conn, Id: v.VisitorID, Avator: v.Avator, ToId: v.ToId, UpdateTime: time.Now()}
	// 存入 map
	visitorConn[user.Id] = user

	// 通知对应客服（若在线）
	last, _ := FindLastMessageByVisitorId(ctx, db, user.Id)
	info := map[string]interface{}{
		"uid":          user.Id,
		"username":     user.Avator,
		"avator":       user.Avator,
		"last_message": last.Content,
	}
	notify := map[string]interface{}{"type": "userOnline", "data": info}
	if user.ToId != "" {
		if k, ok := kefuConn[user.ToId]; ok {
			k.Mux.Lock()
			_ = k.Conn.WriteJSON(notify)
			k.Mux.Unlock()
		}
	}

	// read loop
	for {
		mt, msg, err := conn.ReadMessage()
		if err != nil {
			// cleanup
			delete(visitorConn, user.Id)
			VisitorOffline(user.ToId, user.Id, user.Avator)
			conn.Close()
			return
		}
		wsMsgChan <- WSIncoming{Conn: conn, Content: msg, MessageType: mt, Ctx: r}
	}
}

// Handler: kefu websocket: /ws_kefu?kefu_name=xxx
func NewKefuServer(r *ghttp.Request) {
	conn, err := upgrader.Upgrade(r.Response.Writer, r.Request, nil)
	if err != nil {
		r.Response.WriteStatusExit(400, err.Error())
		return
	}
	kefuName := r.Get("kefu_name").String()
	if kefuName == "" {
		kefuName = fmt.Sprintf("kefu%d", time.Now().UnixNano())
	}
	user := &WSUser{Conn: conn, Id: kefuName, Avator: "", UpdateTime: time.Now()}
	kefuConn[kefuName] = user

	for {
		mt, msg, err := conn.ReadMessage()
		if err != nil {
			delete(kefuConn, kefuName)
			conn.Close()
			return
		}
		wsMsgChan <- WSIncoming{Conn: conn, Content: msg, MessageType: mt}
	}
}

// OneKefuMessage 将消息发送给指定客服
func OneKefuMessage(kefuId string, payload interface{}) {
	if k, ok := kefuConn[kefuId]; ok && k != nil {
		k.Mux.Lock()
		_ = k.Conn.WriteJSON(payload)
		k.Mux.Unlock()
	}
}

// VisitorMessage 将消息发送给访客
func VisitorMessage(visitorId, content, kefuName string) {
	if v, ok := visitorConn[visitorId]; ok && v != nil {
		msg := map[string]interface{}{
			"type": "message",
			"data": map[string]interface{}{
				"name":    kefuName,
				"avator":  "/static/images/kefu.png",
				"id":      kefuName,
				"time":    time.Now().Format("2006-01-02 15:04:05"),
				"to_id":   visitorId,
				"content": content,
				"is_kefu": "yes",
			},
		}
		v.Mux.Lock()
		_ = v.Conn.WriteJSON(msg)
		v.Mux.Unlock()
	}
}

// VisitorOffline 通知客服访客离线
func VisitorOffline(kefuId, visitorId, visitorName string) {
	// update db
	ctx := context.Background()
	_ = UpdateVisitorStatus(ctx, db, visitorId, 0)
	if k, ok := kefuConn[kefuId]; ok && k != nil {
		info := map[string]interface{}{
			"uid":  visitorId,
			"name": visitorName,
		}
		_ = k.Conn.WriteJSON(map[string]interface{}{"type": "userOffline", "data": info})
	}
}
