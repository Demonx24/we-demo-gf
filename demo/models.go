package main

import (
	"context"
	"database/sql"
	"time"
)

// Visitor struct 映射 visitor 表（仅需要的字段）
type Visitor struct {
	ID        int
	VisitorID string
	Name      string
	Avator    string
	ToId      string
	SourceIP  string
	Status    int
	Refer     string
	City      string
	ClientIP  string
	Extra     string
	CreatedAt time.Time
}

// MessageRecord struct
type MessageRecord struct {
	ID        int
	KefuId    string
	VisitorId string
	Content   string
	MesType   string
	Status    string
	CreatedAt time.Time
}

// FindOrCreateVisitor：若不存在则插入一条
func FindOrCreateVisitor(ctx context.Context, db *sql.DB, visitorId string) (Visitor, error) {
	var v Visitor
	err := db.QueryRowContext(ctx, "SELECT id, visitor_id, name, avator, to_id, status, created_at FROM visitor WHERE visitor_id = ? LIMIT 1", visitorId).
		Scan(&v.ID, &v.VisitorID, &v.Name, &v.Avator, &v.ToId, &v.Status, &v.CreatedAt)
	if err != nil && err != sql.ErrNoRows {
		return v, err
	}
	if err == sql.ErrNoRows {
		// insert
		res, err := db.ExecContext(ctx, "INSERT INTO visitor (visitor_id, name, avator, status) VALUES (?, ?, ?, ?)", visitorId, "Visitor-"+visitorId, "/static/images/default.png", 1)
		if err != nil {
			return v, err
		}
		id, _ := res.LastInsertId()
		v.ID = int(id)
		v.VisitorID = visitorId
		v.Name = "Visitor-" + visitorId
		v.Avator = "/static/images/default.png"
		v.Status = 1
		v.CreatedAt = time.Now()
	}
	return v, nil
}

func UpdateVisitorStatus(ctx context.Context, db *sql.DB, visitorId string, status int) error {
	_, err := db.ExecContext(ctx, "UPDATE visitor SET status = ? WHERE visitor_id = ?", status, visitorId)
	return err
}

func FindVisitorByVisitorId(ctx context.Context, db *sql.DB, visitorId string) (Visitor, error) {
	var v Visitor
	err := db.QueryRowContext(ctx, "SELECT id, visitor_id, name, avator, to_id, status, refer, city, client_ip, extra, created_at FROM visitor WHERE visitor_id = ? LIMIT 1", visitorId).
		Scan(&v.ID, &v.VisitorID, &v.Name, &v.Avator, &v.ToId, &v.Status, &v.Refer, &v.City, &v.ClientIP, &v.Extra, &v.CreatedAt)
	if err == sql.ErrNoRows {
		return Visitor{}, nil
	}
	return v, err
}

func CreateMessageRecord(ctx context.Context, db *sql.DB, kefuId, visitorId, content, mesType string) error {
	_, err := db.ExecContext(ctx, "INSERT INTO message (kefu_id, visitor_id, content, mes_type, status) VALUES (?, ?, ?, ?, 'unread')", kefuId, visitorId, content, mesType)
	return err
}

func FindMessagesByVisitorId(ctx context.Context, db *sql.DB, visitorId string) ([]MessageRecord, error) {
	rows, err := db.QueryContext(ctx, "SELECT id, kefu_id, visitor_id, content, mes_type, status, created_at FROM message WHERE visitor_id = ? ORDER BY id ASC", visitorId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []MessageRecord
	for rows.Next() {
		var m MessageRecord
		_ = rows.Scan(&m.ID, &m.KefuId, &m.VisitorId, &m.Content, &m.MesType, &m.Status, &m.CreatedAt)
		list = append(list, m)
	}
	return list, nil
}

func FindLastMessageByVisitorId(ctx context.Context, db *sql.DB, visitorId string) (MessageRecord, error) {
	var m MessageRecord
	err := db.QueryRowContext(ctx, "SELECT id, kefu_id, visitor_id, content, mes_type, status, created_at FROM message WHERE visitor_id = ? ORDER BY id DESC LIMIT 1", visitorId).
		Scan(&m.ID, &m.KefuId, &m.VisitorId, &m.Content, &m.MesType, &m.Status, &m.CreatedAt)
	if err == sql.ErrNoRows {
		return MessageRecord{}, nil
	}
	return m, err
}
