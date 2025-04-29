package websocket

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	// 写入超时
	writeWait = 10 * time.Second

	// 读取超时
	pongWait = 60 * time.Second

	// 发送ping间隔时间，必须小于pongWait
	pingPeriod = (pongWait * 9) / 10

	// 最大消息大小
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 允许所有跨域请求，生产环境应限制
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Handler WebSocket处理器
type Handler struct {
	hub *Hub
}

// NewHandler 创建WebSocket处理器
func NewHandler(hub *Hub) *Handler {
	return &Handler{hub: hub}
}

// RegisterRoutes 注册WebSocket路由
func (h *Handler) RegisterRoutes(router *gin.Engine) {
	router.GET("/ws/polls/:id", h.HandleWebSocketConnection)
}

// HandleWebSocketConnection 处理WebSocket连接请求
func (h *Handler) HandleWebSocketConnection(c *gin.Context) {
	// 获取投票ID
	pollIDStr := c.Param("id")
	pollID, err := strconv.ParseInt(pollIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid poll ID"})
		return
	}

	// 升级HTTP连接为WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		return
	}

	// 创建客户端
	client := &Client{
		PollID: pollID,
		conn:   conn,
		send:   make(chan []byte, 256),
	}

	// 注册客户端
	h.hub.RegisterClient(client)

	// 启动客户端goroutine
	go h.writePump(client)
	go h.readPump(client)

	log.Printf("New WebSocket connection established for poll %d", pollID)
}

// readPump 从WebSocket连接读取消息
func (h *Handler) readPump(client *Client) {
	defer func() {
		h.hub.UnregisterClient(client)
		client.conn.Close()
	}()

	client.conn.SetReadLimit(maxMessageSize)
	client.conn.SetReadDeadline(time.Now().Add(pongWait))
	client.conn.SetPongHandler(func(string) error {
		client.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Error reading message: %v", err)
			}
			break
		}
		// 这个实现中，客户端只接收消息，不处理客户端发送的消息
	}
}

// writePump 向WebSocket连接发送消息
func (h *Handler) writePump(client *Client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		client.conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.send:
			client.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// 通道已关闭
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := client.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 添加队列中的消息
			n := len(client.send)
			for i := 0; i < n; i++ {
				w.Write(<-client.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			client.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
