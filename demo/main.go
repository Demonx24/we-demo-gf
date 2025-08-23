package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// Client 表示一个连接
type Client struct {
	Conn *websocket.Conn
	Send chan []byte // 发送队列
}

// readPump：专门负责读消息
func (c *Client) readPump() {
	defer c.Conn.Close()
	for {
		_, msg, err := c.Conn.ReadMessage()
		if err != nil {
			fmt.Println("read error:", err)
			return
		}
		fmt.Println("收到消息:", string(msg))
		// 把收到的消息原样放到发送队列
		c.Send <- msg
	}
}

// writePump：专门负责写消息 + 心跳
func (c *Client) writePump() {
	ticker := time.NewTicker(10 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.Send:
			if !ok {
				// channel 被关闭
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			// 写消息
			c.Conn.WriteMessage(websocket.TextMessage, msg)
		case <-ticker.C:
			// 定时心跳
			c.Conn.WriteMessage(websocket.PingMessage, []byte("ping"))
		}
	}
}

// websocket 入口
func serveWs(w http.ResponseWriter, r *http.Request) {
	conn, _ := upgrader.Upgrade(w, r, nil)
	client := &Client{
		Conn: conn,
		Send: make(chan []byte, 256),
	}

	// 一个 goroutine 写消息
	go client.writePump()

	// 当前 goroutine 读消息（阻塞）
	client.readPump()
}

func main() {
	http.HandleFunc("/ws", serveWs)
	fmt.Println("启动在 :8080")
	http.ListenAndServe(":8080", nil)
}
