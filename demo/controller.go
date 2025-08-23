package main

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"io"
	"os"
	"path"
	"strconv"
	"time"
)

// RegisterRoutes same as before
func RegisterRoutes(s *ghttp.Server) {
	s.BindHandler("/visitor_login", VisitorLogin)
	s.BindHandler("/2/message", SendMessageV2)
	s.BindHandler("/2/messagesPages", GetMessagespages)
	s.BindHandler("/notice", GetNotice)
	s.BindHandler("/uploadimg", UploadImg)
	s.BindHandler("/uploadfile", UploadFile)
	// WebSocket endpoints
	s.BindHandler("/ws_visitor", NewVisitorServer)
	s.BindHandler("/ws_kefu", NewKefuServer)
}

// VisitorLogin: POST
func VisitorLogin(r *ghttp.Request) {
	// 使用 r.Get(...).String() 替代 r.GetString(...)
	visitorId := r.Get("visitor_id").String()
	toId := r.Get("to_id").String()
	if visitorId == "" {
		visitorId = strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	ctx := context.Background()
	v, err := FindOrCreateVisitor(ctx, db, visitorId)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": err.Error()})
		return
	}
	if toId != "" {
		// set to_id in DB
		_, _ = db.Exec("UPDATE visitor SET to_id = ? WHERE visitor_id = ?", toId, visitorId)
		v.ToId = toId
	}
	r.Response.WriteJsonExit(g.Map{"code": 200, "msg": "ok", "result": g.Map{
		"visitor_id": v.VisitorID, "name": v.Name, "avator": v.Avator, "to_id": v.ToId,
	}})
}

// SendMessageV2: POST /2/message
func SendMessageV2(r *ghttp.Request) {
	fromId := r.Get("from_id").String()
	toId := r.Get("to_id").String()
	content := r.Get("content").String()
	cType := r.Get("type").String()
	if content == "" {
		r.Response.WriteJsonExit(g.Map{"code": 400, "msg": "content empty"})
		return
	}
	ctx := context.Background()
	if cType == "visitor" {
		// persist and push to kefu
		_ = CreateMessageRecord(ctx, db, toId, fromId, content, "visitor")
		// push to kefu
		msg := map[string]interface{}{
			"type": "message",
			"data": map[string]interface{}{
				"avator":  "/static/images/default.png",
				"id":      fromId,
				"name":    "visitor",
				"to_id":   toId,
				"content": content,
				"time":    time.Now().Format("2006-01-02 15:04:05"),
				"is_kefu": "no",
			},
		}
		OneKefuMessage(toId, msg)
	} else {
		// kefu -> visitor
		_ = CreateMessageRecord(ctx, db, fromId, toId, content, "kefu")
		VisitorMessage(toId, content, fromId)
	}
	r.Response.WriteJsonExit(g.Map{"code": 200, "msg": "ok"})
}

// GetMessagespages: GET
func GetMessagespages(r *ghttp.Request) {
	visitorId := r.Get("visitor_id").String()

	// 使用 r.Get(...).Int() 取整型，如果失败返回 0
	page := 1
	if v := r.Get("page"); v != nil {
		if pi := v.Int(); pi > 0 {
			page = pi
		}
	}
	pagesize := 20
	if v := r.Get("pagesize"); v != nil {
		if ps := v.Int(); ps > 0 {
			pagesize = ps
		}
	}

	if page <= 0 {
		page = 1
	}
	if pagesize <= 0 {
		pagesize = 20
	}
	ctx := context.Background()
	msgs, _ := FindMessagesByVisitorId(ctx, db, visitorId)
	// build response similar to原前端期待结构
	list := make([]map[string]interface{}, 0)
	for _, m := range msgs {
		item := map[string]interface{}{
			"mes_type":       m.MesType,
			"content":        m.Content,
			"create_time":    m.CreatedAt.Format("2006-01-02 15:04:05"),
			"kefu_id":        m.KefuId,
			"visitor_id":     m.VisitorId,
			"kefu_avator":    "/static/images/kefu.png",
			"visitor_avator": "/static/images/default.png",
		}
		list = append(list, item)
	}
	// simple pagination (注意：这里是在内存slice上分页，数据量大需改为 DB 分页)
	start := (page - 1) * pagesize
	end := start + pagesize
	if start < 0 {
		start = 0
	}
	if end > len(list) {
		end = len(list)
	}
	sub := list
	if start < len(list) {
		sub = list[start:end]
	} else {
		sub = []map[string]interface{}{}
	}
	r.Response.WriteJsonExit(g.Map{"code": 200, "msg": "ok", "result": g.Map{
		"count":    len(list),
		"page":     page,
		"list":     sub,
		"pagesize": pagesize,
	}})
}

// GetNotice: return kefu info (simplified)
func GetNotice(r *ghttp.Request) {
	kefuId := r.Get("kefu_id").String()
	// sample response
	result := map[string]interface{}{
		"avatar":    "/static/images/kefu.png",
		"nickname":  kefuId,
		"welcome":   "Hi, I'm here to help you.",
		"allNotice": "Welcome",
	}
	r.Response.WriteJsonExit(g.Map{"code": 200, "msg": "ok", "result": result})
}

// UploadImg
func UploadImg(r *ghttp.Request) {
	file := r.GetUploadFile("imgfile")
	if file == nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "msg": "no file"})
		return
	}
	// 打开上传文件以获得 io.Reader
	src, err := file.Open()
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": "open file fail"})
		return
	}
	defer src.Close()

	// 确保目标目录存在
	os.MkdirAll("uploads", os.ModePerm)
	ext := path.Ext(file.Filename)
	dst := fmt.Sprintf("uploads/%d%s", time.Now().UnixNano(), ext)

	out, err := os.Create(dst)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": "save fail"})
		return
	}
	defer out.Close()

	// 将上传的内容复制到文件，并获取写入的字节数
	written, err := io.Copy(out, src)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": "save fail"})
		return
	}

	r.Response.WriteJsonExit(g.Map{
		"code": 200, "msg": "ok",
		"result": g.Map{
			"path": dst,
			"size": written,
			"name": file.Filename,
		},
	})
}

// UploadFile (realfile)
func UploadFile(r *ghttp.Request) {
	file := r.GetUploadFile("realfile")
	if file == nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "msg": "no file"})
		return
	}

	src, err := file.Open()
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": "open file fail"})
		return
	}
	defer src.Close()

	os.MkdirAll("uploads", os.ModePerm)
	ext := path.Ext(file.Filename)
	dst := fmt.Sprintf("uploads/%d%s", time.Now().UnixNano(), ext)

	out, err := os.Create(dst)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": "save fail"})
		return
	}
	defer out.Close()

	written, err := io.Copy(out, src)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": "save fail"})
		return
	}

	r.Response.WriteJsonExit(g.Map{
		"code": 200, "msg": "ok",
		"result": g.Map{
			"path": dst,
			"ext":  ext,
			"size": written,
			"name": file.Filename,
		},
	})
}
