package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/gorilla/websocket"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

/*
表结构（与需求对齐）
1) 消息表: talk_message
   - id (pk, int, ai)
   - nickname (varchar)
   - receiver_id (int)
   - send_id (int)
   - msg_type (int)
   - avatar (varchar)
   - content (varchar)
   - sid (int)  会话ID
   - is_read (int)  1已读,0未读
   - created_at (timestamp, default CURRENT_TIMESTAMP)

2) 会话表: talk_session
   - id (pk, int, ai)
   - receiver_id (int)   接收者id
   - is_online (tinyint) 1 在线 2 离线
   - name (varchar)      会话名称
   - un_read_num (int)   未读数量
   - msg_text (varchar)  最后一条消息
   - updated_at (datetime) 更新时间
   - send_id (int)       发送人id
   - status (tinyint)    0 隐藏 1 显示

3) 用户表: talk_user
   - id (pk, int, ai)
   - username (varchar)
   - user_id (int)
   - user_avatar (varchar)
*/

// ---------------------- GORM 模型 ----------------------
type TalkMessage struct {
	ID         int    `gorm:"primaryKey;column:id" json:"id"`
	Nickname   string `gorm:"column:nickname" json:"nickname"`
	ReceiverID int    `gorm:"column:receiver_id" json:"receiver_id"`
	SendID     int    `gorm:"column:send_id" json:"send_id"`
	MsgType    int    `gorm:"column:msg_type" json:"msg_type"`
	Avatar     string `gorm:"column:avatar" json:"avatar"`
	Content    string `gorm:"column:content" json:"content"`
	//Sid        int       `gorm:"column:sid" json:"sid"`
	IsRead    int       `gorm:"column:is_read" json:"is_read"` // 1已读(在线送达), 0未读(离线)
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (TalkMessage) TableName() string { return "message" }

type TalkSession struct {
	ID         int       `gorm:"primaryKey;column:id" json:"id"`
	ReceiverID int       `gorm:"column:receiver_id" json:"receiver_id"`
	IsOnline   int       `gorm:"column:is_online" json:"is_online"` // 1在线 2离线
	Name       string    `gorm:"column:name" json:"name"`
	UnReadNum  int       `gorm:"column:un_read_num" json:"un_read_num"`
	MsgText    string    `gorm:"column:msg_text" json:"msg_text"`
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	SendID     int       `gorm:"column:send_id" json:"send_id"`
	Status     int       `gorm:"column:status;default:1" json:"status"` // 0隐藏 1显示
}

func (TalkSession) TableName() string { return "session" }

type TalkUser struct {
	ID         int    `gorm:"primaryKey;column:id" json:"id"`
	Username   string `gorm:"column:username" json:"username"`
	UserID     int    `gorm:"column:user_id" json:"user_id"`
	UserAvatar string `gorm:"column:user_avatar" json:"user_avatar"`
}

func (TalkUser) TableName() string { return "users" }

// ---------------------- WebSocket 相关 ----------------------
type Client struct {
	Conn      *websocket.Conn
	UserID    int
	Name      string
	SendCh    chan []byte
	LastPong  time.Time
	Heartbeat time.Duration
	FirstPing bool
}

var (
	upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

	clientsByConn   = make(map[*websocket.Conn]*Client)
	clientsByUserID = make(map[int]*Client)
	clientsMu       sync.Mutex

	db *gorm.DB
)

// ---------------------- 初始化数据库 ----------------------
func initDB(ctx context.Context) {
	// 优先用配置文件 database.default.link；否则用环境变量 MYSQL_DSN
	dsn := g.Cfg().MustGet(ctx, "database.default.link").String()
	if dsn == "" {
		dsn = os.Getenv("MYSQL_DSN")
	}
	if dsn == "" {
		panic("missing database dsn: set config \"database.default.link\" or env MYSQL_DSN")
	}

	var err error
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect database: " + err.Error())
	}

	// 仅确保表存在（不会破坏已有字段约束）
	_ = db.AutoMigrate(&TalkMessage{}, &TalkSession{}, &TalkUser{})
}

// ---------------------- WebSocket 心跳 & 读写 ----------------------
func sendWS(c *Client, payload any) {
	data, _ := json.Marshal(payload)
	select {
	case c.SendCh <- data:
	default:
		// 下游阻塞，关闭连接回收
		close(c.SendCh)
		clientsMu.Lock()
		delete(clientsByConn, c.Conn)
		delete(clientsByUserID, c.UserID)
		clientsMu.Unlock()
		_ = c.Conn.Close()
	}
}

// GET /user/list
func userListHandler(r *ghttp.Request) {
	var users []TalkUser
	if err := db.Find(&users).Error; err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": "查询失败"})
		return
	}

	// 组装返回：附带在线状态
	type UserDTO struct {
		ID         int    `json:"id"`
		UserID     int    `json:"user_id"`
		Username   string `json:"username"`
		UserAvatar string `json:"user_avatar"`
		IsOnline   int    `json:"is_online"` // 1在线 2离线
	}
	res := make([]UserDTO, 0, len(users))

	clientsMu.Lock()
	for _, u := range users {
		isOnline := 2
		if _, ok := clientsByUserID[u.UserID]; ok {
			isOnline = 1
		}
		res = append(res, UserDTO{
			ID:         u.ID,
			UserID:     u.UserID,
			Username:   u.Username,
			UserAvatar: u.UserAvatar,
			IsOnline:   isOnline,
		})
	}
	clientsMu.Unlock()

	r.Response.WriteJsonExit(g.Map{"code": 0, "msg": "success", "data": res})
}

func broadcastPresence(uid int, online bool) {
	payload := map[string]any{
		"event": "user_presence",
		"data": map[string]any{
			"user_id": uid,
			"online":  online, // true=在线, false=离线
		},
	}
	clientsMu.Lock()
	defer clientsMu.Unlock()
	for _, cl := range clientsByUserID {
		sendWS(cl, payload)
	}
}
func readPump(c *Client) {
	defer func() {
		clientsMu.Lock()
		delete(clientsByConn, c.Conn)
		delete(clientsByUserID, c.UserID)
		clientsMu.Unlock()
		_ = c.Conn.Close()
		//broadcastPresence(c.UserID, false)
	}()

	c.Conn.SetPongHandler(func(appData string) error {
		c.LastPong = time.Now()
		return nil
	})

	type InMsg struct {
		Event string          `json:"event"`
		Data  json.RawMessage `json:"data"`
	}

	for {
		_, raw, err := c.Conn.ReadMessage()
		if err != nil {
			return
		}

		var in InMsg
		if err := json.Unmarshal(raw, &in); err != nil {
			continue
		}

		switch in.Event {
		case "ping":
			// 先回 pong
			sendWS(c, map[string]any{"event": "pong"})

			// 首次心跳同时返回 connect
			if c.FirstPing {
				connectPayload := map[string]any{
					"event": "connect",
					"content": map[string]any{
						"message":       "连接成功",
						"ping_interval": int(c.Heartbeat.Seconds()),           // 30
						"ping_timeout":  int((c.Heartbeat * 5 / 2).Seconds()), // 75
					},
				}
				sendWS(c, connectPayload)
				c.FirstPing = false
			}
			c.LastPong = time.Now()

			// 这里可以扩展更多 ws 侧业务事件，但当前消息发送走 HTTP
		}
	}
}

func writePump(c *Client) {
	ticker := time.NewTicker(c.Heartbeat)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-c.SendCh:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
			// 心跳 2 倍未收到 pong 认为断开
			if time.Since(c.LastPong) > c.Heartbeat*2 {
				return
			}
		}
	}
}

// ---------------------- HTTP Handlers ----------------------

// 创建会话
// POST /talk/session/save
// body: { "send_id":1, "receiver_id":2, "name":"张三&李四" }
func createSessionHandler(r *ghttp.Request) {
	var req struct {
		SendID     int    `json:"send_id"`
		ReceiverID int    `json:"receiver_id"`
		Name       string `json:"name"`
	}
	if err := r.Parse(&req); err != nil || req.SendID == 0 || req.ReceiverID == 0 {
		r.Response.WriteJsonExit(g.Map{"code": 400, "msg": "参数错误"})
		return
	}

	s := &TalkSession{
		SendID:     req.SendID,
		ReceiverID: req.ReceiverID,
		Name:       req.Name,
		IsOnline:   2,
		Status:     1,
		UpdatedAt:  time.Now(),
	}
	if err := db.Create(s).Error; err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": "创建会话失败"})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "msg": "操作成功", "data": g.Map{"sid": s.ID}})
}

// 会话列表
// GET /talk/session/list?user_id=1
func sessionListHandler(r *ghttp.Request) {
	var req struct {
		UserID int `json:"id"`
	}
	if err := r.Parse(&req); err != nil || req.UserID == 0 {
		r.Response.WriteJsonExit(g.Map{"code": 400, "msg": "参数错误"})
		return
	}

	var list []TalkSession
	if err := db.Where("status=1 AND (receiver_id=? OR send_id=?)", req.UserID, req.UserID).
		Order("updated_at desc").Find(&list).Error; err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": "查询失败"})
		return
	}

	// 标注在线状态（基于 WS 内存）
	clientsMu.Lock()
	for i := range list {
		if _, ok := clientsByUserID[list[i].ReceiverID]; ok {
			list[i].IsOnline = 1
		} else {
			list[i].IsOnline = 2
		}
	}
	clientsMu.Unlock()

	r.Response.WriteJsonExit(g.Map{"code": 0, "msg": "success", "data": list})
}

// 消息列表
// GET /talk/message/list?sid=1001&page=1&size=20
func messageListHandler(r *ghttp.Request) {
	var req struct {
		SessionID int `json:"id"`   // 会话ID
		Page      int `json:"page"` // 页码
		Size      int `json:"size"` // 每页大小
	}

	if err := r.Parse(&req); err != nil || req.SessionID == 0 {
		r.Response.WriteJsonExit(g.Map{"code": 400, "msg": "参数错误"})
		return
	}

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 20
	}
	offset := (req.Page - 1) * req.Size

	var msgs []TalkMessage
	if err := db.Where("sid = ?", req.SessionID).
		Order("id desc").
		Limit(req.Size).
		Offset(offset).
		Find(&msgs).Error; err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": "查询失败"})
		return
	}

	r.Response.WriteJsonExit(g.Map{"code": 0, "msg": "success", "data": msgs})
}

// 发送消息（HTTP）
// POST /talk/message/send
// body: { "session_id":1001, "send_id":1, "receiver_id":2, "msg_type":1, "content":"你好", "nickname":"张三", "avatar":"https://..." }
func sendMessageHandler(r *ghttp.Request) {
	var req struct {
		SessionID  int    `json:"session_id"`
		SendID     int    `json:"send_id"`
		ReceiverID int    `json:"receiver_id"`
		MsgType    int    `json:"msg_type"`
		Content    string `json:"content"`
		Nickname   string `json:"nickname"`
		Avatar     string `json:"avatar"`
	}
	if err := r.Parse(&req); err != nil ||
		req.SessionID == 0 || req.SendID == 0 || req.ReceiverID == 0 || req.MsgType == 0 || req.Content == "" {
		r.Response.WriteJsonExit(g.Map{"code": 400, "msg": "参数错误"})
		return
	}
	// —— 新增：会话归属校验 —— //
	var sess TalkSession
	if err := db.First(&sess, "id=?", req.SessionID).Error; err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 404, "msg": "会话不存在"})
		return
	}

	// 发送者必须在会话里
	if req.SendID != sess.SendID && req.SendID != sess.ReceiverID {
		r.Response.WriteJsonExit(g.Map{"code": 403, "msg": "你不在该会话中"})
		return
	}
	// 接收者必须是会话里除自己以外的那一方
	expectedReceiver := sess.SendID
	if req.SendID == sess.SendID {
		expectedReceiver = sess.ReceiverID
	}
	if req.ReceiverID != expectedReceiver {
		r.Response.WriteJsonExit(g.Map{"code": 400, "msg": "接收者与会话不匹配"})
		return
	}
	// 先写库（默认未读）
	msg := &TalkMessage{
		//Sid:        req.SessionID,
		SendID:     req.SendID,
		ReceiverID: req.ReceiverID,
		MsgType:    req.MsgType,
		Content:    req.Content,
		Nickname:   req.Nickname,
		Avatar:     req.Avatar,
		IsRead:     0,
	}
	if err := db.Create(msg).Error; err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": "保存消息失败"})
		return
	}

	// 更新会话最后消息 & 未读
	online := false
	clientsMu.Lock()
	rc, ok := clientsByUserID[req.ReceiverID]
	if ok && rc != nil {
		online = true
	}
	clientsMu.Unlock()

	sessionUpdate := map[string]any{
		"msg_text":   req.Content,
		"updated_at": time.Now(),
	}
	if !online {
		sessionUpdate["un_read_num"] = gorm.Expr("un_read_num + 1")
	}
	_ = db.Model(&TalkSession{}).Where("id=?", req.SessionID).Updates(sessionUpdate).Error

	// 若对方在线：经 WS 推送，并将该条消息置为已读
	if online {
		msg.IsRead = 1
		_ = db.Model(msg).Update("is_read", 1).Error

		push := map[string]any{
			"event": "im.message",
			"sid":   req.SessionID,
			"content": map[string]any{
				"data": map[string]any{
					"id":          msg.ID,
					"session_id":  req.SessionID,
					"send_id":     req.SendID,
					"receiver_id": req.ReceiverID,
					"nickname":    req.Nickname,
					"avatar":      req.Avatar,
					"msg_type":    msg.MsgType,
					"content":     msg.Content,
					"created_at":  msg.CreatedAt.Format("2006-01-02 15:04:05"),
					"is_read":     1,
				},
				"receiver_id": req.ReceiverID,
				"send_id":     req.SendID,
			},
		}
		sendWS(rc, push)
	}
	// —— 新增：推送最新会话信息给双方 —— //
	var fresh TalkSession
	_ = db.First(&fresh, "id=?", req.SessionID).Error

	clientsMu.Lock()
	sc := clientsByUserID[req.SendID]
	rc2 := clientsByUserID[req.ReceiverID]
	clientsMu.Unlock()

	if sc != nil {
		sendWS(sc, map[string]any{"event": "session_updated", "data": fresh})
	}
	if rc2 != nil {
		sendWS(rc2, map[string]any{"event": "session_updated", "data": fresh})
	}

	// —— 返回结果 —— //
	r.Response.WriteJsonExit(g.Map{"code": 0, "msg": "success", "data": msg})
}
func upsertUser(uid int, name, avatar string) {
	var u TalkUser
	err := db.Where("user_id = ?", uid).First(&u).Error
	switch {
	case err == nil:
		// 如昵称/头像有变化则更新
		needUpdate := false
		update := map[string]any{}
		if name != "" && name != u.Username {
			update["username"] = name
			needUpdate = true
		}
		if avatar != "" && avatar != u.UserAvatar {
			update["user_avatar"] = avatar
			needUpdate = true
		}
		if needUpdate {
			_ = db.Model(&TalkUser{}).Where("id=?", u.ID).Updates(update).Error
		}
	case err == gorm.ErrRecordNotFound:
		// 不存在就创建
		_ = db.Create(&TalkUser{
			Username:   name,
			UserID:     uid,
			UserAvatar: avatar,
		}).Error
	default:
		// 其他错误忽略
	}
}

// 上传文件
// POST /upload/file  (multipart/form-data)
// fields: file, send_id, receiver_id, msg_type, session_id, nickname(可选), avatar(可选)
func uploadHandler(r *ghttp.Request) {
	file := r.GetUploadFile("file")
	if file == nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "message": "缺少文件"})
		return
	}
	sendID := r.Get("send_id").Int()
	receiverID := r.Get("receiver_id").Int()
	msgType := r.Get("msg_type").Int()
	sessionID := r.Get("session_id").Int()
	nickname := r.Get("nickname").String()
	avatar := r.Get("avatar").String()
	if sendID == 0 || receiverID == 0 || msgType == 0 || sessionID == 0 {
		r.Response.WriteJsonExit(g.Map{"code": 400, "message": "参数不完整"})
		return
	}

	baseDir := "static/uploads/"
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": "创建目录失败"})
		return
	}
	timestampDir := strconv.FormatInt(time.Now().Unix(), 10)
	savePath := filepath.Join(baseDir, timestampDir)
	if err := os.MkdirAll(savePath, 0755); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": "创建目录失败"})
		return
	}
	savedFileName, err := file.Save(savePath)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": "保存失败"})
		return
	}
	fileURL := fmt.Sprintf("/uploads/%s/%s", timestampDir, savedFileName)

	// 以文件URL作为消息内容
	msg := &TalkMessage{
		//Sid:        sessionID,
		SendID:     sendID,
		ReceiverID: receiverID,
		MsgType:    msgType,
		Content:    fileURL,
		Nickname:   nickname,
		Avatar:     avatar,
		IsRead:     0,
	}
	if err := db.Create(msg).Error; err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": "保存消息失败"})
		return
	}

	// 会话更新
	online := false
	clientsMu.Lock()
	rc, ok := clientsByUserID[receiverID]
	if ok && rc != nil {
		online = true
	}
	clientsMu.Unlock()

	update := map[string]any{
		"msg_text":   fileURL,
		"updated_at": time.Now(),
	}
	if !online {
		update["un_read_num"] = gorm.Expr("un_read_num + 1")
	}
	_ = db.Model(&TalkSession{}).Where("id=?", sessionID).Updates(update).Error

	// 在线则推送
	if online {
		msg.IsRead = 1
		_ = db.Model(msg).Update("is_read", 1).Error

		push := map[string]any{
			"event": "im.message",
			"sid":   sessionID,
			"content": map[string]any{
				"data": map[string]any{
					"id":          msg.ID,
					"session_id":  sessionID,
					"send_id":     sendID,
					"receiver_id": receiverID,
					"nickname":    nickname,
					"avatar":      avatar,
					"msg_type":    msgType,
					"content":     fileURL,
					"created_at":  msg.CreatedAt.Format("2006-01-02 15:04:05"),
					"is_read":     1,
				},
				"receiver_id": receiverID,
				"send_id":     sendID,
			},
		}
		sendWS(rc, push)
	}

	r.Response.WriteJsonExit(g.Map{
		"code":    0,
		"message": "上传成功",
		"data": g.Map{
			"file_url":   fileURL,
			"msg_id":     msg.ID,
			"created_at": msg.CreatedAt.Format("2006-01-02 15:04:05"),
		},
	})
}

// ---------------------- WebSocket Handler ----------------------
func wsHandler(r *ghttp.Request) {
	ws, err := upgrader.Upgrade(r.Response.Writer, r.Request, nil)
	if err != nil {
		r.Response.WriteStatus(http.StatusBadRequest, err.Error())
		return
	}

	uid := r.Get("id").Int()
	if uid == 0 {
		uid = int(time.Now().UnixNano() / 1e6)
	}
	name := r.Get("name").String()
	if name == "" {
		name = fmt.Sprintf("U%d", uid)
	}
	avatar := r.Get("avatar").String()

	// ★ 先把用户写入/更新到 users 表
	upsertUser(uid, name, avatar)

	c := &Client{
		Conn:      ws,
		UserID:    uid,
		Name:      name,
		SendCh:    make(chan []byte, 256),
		LastPong:  time.Now(),
		Heartbeat: 30 * time.Second,
		FirstPing: true,
	}

	clientsMu.Lock()
	clientsByConn[ws] = c
	clientsByUserID[uid] = c
	clientsMu.Unlock()

	// 上线广播给所有在线用户（可选）
	//broadcastPresence(uid, true)

	// 上线即推送会话列表并同步离线未读消息
	//pushSessionListTo(uid)
	go deliverUnread(c)

	go writePump(c)
	readPump(c) // 阻塞到断开
}

// —— 新增：推送某用户的会话列表 —— //
func pushSessionListTo(uid int) {
	var list []TalkSession
	if err := db.Where("status=1 AND (receiver_id=? OR send_id=?)", uid, uid).
		Order("updated_at desc").Find(&list).Error; err != nil {
		return
	}

	clientsMu.Lock()
	// 标注在线状态
	for i := range list {
		if _, ok := clientsByUserID[list[i].ReceiverID]; ok {
			list[i].IsOnline = 1
		} else {
			list[i].IsOnline = 2
		}
	}
	c := clientsByUserID[uid]
	clientsMu.Unlock()

	if c != nil {
		sendWS(c, map[string]any{"event": "session_list", "data": list})
	}
}

// —— 新增：把该用户的未读消息标记为已读并一次性推送给他 —— //
func deliverUnread(c *Client) {
	var msgs []TalkMessage
	if err := db.Where("receiver_id=? AND is_read=0", c.UserID).
		Order("id asc").Find(&msgs).Error; err == nil && len(msgs) > 0 {

		// 置为已读
		_ = db.Model(&TalkMessage{}).
			Where("receiver_id=? AND is_read=0", c.UserID).
			Update("is_read", 1).Error

		// 会话未读清零
		_ = db.Model(&TalkSession{}).
			Where("receiver_id=?", c.UserID).
			Update("un_read_num", 0).Error

		for _, msg := range msgs {
			// 找到对应会话
			var sess TalkSession
			_ = db.Where("(send_id=? AND receiver_id=?) OR (send_id=? AND receiver_id=?)",
				msg.SendID, msg.ReceiverID, msg.ReceiverID, msg.SendID).First(&sess).Error

			push := map[string]any{
				"event": "im.message",
				"sid":   sess.ID,
				"content": map[string]any{
					"data": map[string]any{
						"id":          msg.ID,
						"session_id":  sess.ID,
						"send_id":     msg.SendID,
						"receiver_id": msg.ReceiverID,
						"nickname":    msg.Nickname,
						"avatar":      msg.Avatar,
						"msg_type":    msg.MsgType,
						"content":     msg.Content,
						"created_at":  msg.CreatedAt.Format("2006-01-02 15:04:05"),
						"is_read":     1,
					},
					"receiver_id": msg.ReceiverID,
					"send_id":     msg.SendID,
				},
			}
			sendWS(c, push)

		}
	}
}

func reviewHandler(r *ghttp.Request) {
	var req struct {
		SendID       int    `json:"send_id"`
		SendName     string `json:"send_name"`
		ReceiverID   int    `json:"receiver_id"`
		ReceiverName string `json:"receiver_name"`
	}
	if err := r.Parse(&req); err != nil || req.SendID == 0 || req.ReceiverID == 0 {
		r.Response.WriteJsonExit(g.Map{"code": 400, "msg": "参数错误"})
		return
	}

	now := time.Now()

	// --- 获取或创建发送者会话 ---
	var sendSession TalkSession
	if err := db.Where("send_id=? AND receiver_id=?", req.SendID, req.ReceiverID).First(&sendSession).Error; err != nil {
		sendSession = TalkSession{
			SendID:     req.SendID,
			ReceiverID: req.ReceiverID,
			Name:       req.ReceiverName,
			IsOnline:   2,
			Status:     1,
			UpdatedAt:  now,
			UnReadNum:  0,
		}
		_ = db.Create(&sendSession).Error
	}

	// --- 获取或创建接收者会话 ---
	var recvSession TalkSession
	if err := db.Where("send_id=? AND receiver_id=?", req.ReceiverID, req.SendID).First(&recvSession).Error; err != nil {
		recvSession = TalkSession{
			SendID:     req.ReceiverID,
			ReceiverID: req.SendID,
			Name:       req.SendName,
			IsOnline:   2,
			Status:     1,
			UpdatedAt:  now,
			UnReadNum:  1,
		}
		_ = db.Create(&recvSession).Error
	} else {
		// 已存在会话且接收者离线，增加未读
		clientsMu.Lock()
		rcReceiver := clientsByUserID[req.ReceiverID]
		clientsMu.Unlock()
		if rcReceiver == nil {
			db.Model(&recvSession).Update("un_read_num", gorm.Expr("un_read_num + 1"))
		}
	}

	// --- 创建消息 ---
	msg := &TalkMessage{
		SendID:     req.SendID,
		ReceiverID: req.ReceiverID,
		MsgType:    1000,
		Content:    "[复核报价]",
		Nickname:   req.SendName,
		IsRead:     0,
		CreatedAt:  now,
	}
	_ = db.Create(msg).Error

	// --- WS 推送 ---
	clientsMu.Lock()
	rcReceiver := clientsByUserID[req.ReceiverID]
	rcSender := clientsByUserID[req.SendID]
	clientsMu.Unlock()

	// --- 如果接收者在线，则发送消息 ---
	if rcReceiver != nil {
		msg.IsRead = 1
		_ = db.Model(msg).Update("is_read", 1)

		push := map[string]any{
			"event": "im.message",
			"sid":   sendSession.ID,
			"content": map[string]any{
				"data": map[string]any{
					"id":          msg.ID,
					"session_id":  sendSession.ID,
					"send_id":     req.SendID,
					"receiver_id": req.ReceiverID,
					"nickname":    req.SendName,
					"avatar":      "",
					"msg_type":    1000,
					"content":     msg.Content,
					"created_at":  msg.CreatedAt.Format("2006-01-02 15:04:05"),
					"is_read":     1,
				},
				"receiver_id": req.ReceiverID,
				"send_id":     req.SendID,
			},
		}
		sendWS(rcReceiver, push)
	}

	// --- 更新会话最后消息 ---
	update := map[string]any{
		"msg_text":   msg.Content,
		"updated_at": now,
	}
	db.Model(&sendSession).Updates(update)
	db.Model(&recvSession).Updates(update)

	// --- 推送最新会话状态 ---
	var freshSend, freshRecv TalkSession
	db.First(&freshSend, "id=?", sendSession.ID)
	db.First(&freshRecv, "id=?", recvSession.ID)

	clientsMu.Lock()
	if rcSender != nil {
		freshSend.IsOnline = 1
		sendWS(rcSender, map[string]any{"event": "session_updated", "data": freshSend})
	}
	if rcReceiver != nil {
		freshRecv.IsOnline = 1
		sendWS(rcReceiver, map[string]any{"event": "session_updated", "data": freshRecv})
	}
	clientsMu.Unlock()

	r.Response.WriteJsonExit(g.Map{"code": 0, "msg": "复核报价操作成功"})
}

func markSessionReadHandler(r *ghttp.Request) {
	var req struct {
		SessionID int `json:"session_id"`
		UserID    int `json:"user_id"`
	}
	if err := r.Parse(&req); err != nil || req.SessionID == 0 || req.UserID == 0 {
		r.Response.WriteJsonExit(g.Map{"code": 400, "msg": "参数错误"})
		return
	}
	_ = db.Model(&TalkMessage{}).
		Where("sid=? AND receiver_id=? AND is_read=0", req.SessionID, req.UserID).
		Update("is_read", 1).Error

	_ = db.Model(&TalkSession{}).
		Where("id=? AND receiver_id=?", req.SessionID, req.UserID).
		Update("un_read_num", 0).Error

	// 回推最新会话列表
	//pushSessionListTo(req.UserID)
	r.Response.WriteJsonExit(g.Map{"code": 0, "msg": "ok"})
}

// ---------------------- Main ----------------------
func main() {
	ctx := gctx.New()
	initDB(ctx)

	s := g.Server()

	// WebSocket
	s.BindHandler("/ws", wsHandler)

	// REST
	s.Group("/talk", func(group *ghttp.RouterGroup) {
		group.POST("/review", reviewHandler)
		s.BindHandler("/user/list", userListHandler)
		group.Group("/session", func(gp *ghttp.RouterGroup) {
			gp.POST("/save", createSessionHandler)
			gp.POST("/list", sessionListHandler)
		})
		group.Group("/message", func(gp *ghttp.RouterGroup) {
			gp.POST("/list", messageListHandler)
			gp.POST("/send", sendMessageHandler)
			gp.POST("/read", markSessionReadHandler)
		})
	})

	// 上传
	s.BindHandler("/upload/file", uploadHandler)

	// 静态资源
	s.SetServerRoot("static")

	// 端口
	var port string
	flag.StringVar(&port, "port", "", "server port")
	flag.Parse()
	if port == "" {
		port = g.Cfg().MustGet(ctx, "server.address").String() // 如 ":8000"
	}
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = ":8000"
	}
	// GoFrame 端口配置为数字，剪掉前面的冒号
	s.SetPort(gconv.Int(port[1:]))

	s.Run()
}
