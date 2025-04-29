package websocket

import (
	"log"
	"sync"

	"github.com/gorilla/websocket"

	"backend/model"
)

// Client 代表一个WebSocket连接客户端
type Client struct {
	// 连接的投票ID
	PollID int64

	// WebSocket连接
	conn *websocket.Conn

	// 消息发送通道
	send chan []byte
}

// Hub 维护活跃的客户端集合并向客户端广播消息
type Hub struct {
	// 已注册的客户端，按投票ID分组
	clients map[int64]map[*Client]bool

	// 注册请求
	register chan *Client

	// 注销请求
	unregister chan *Client

	// 互斥锁保护clients map
	mu sync.RWMutex
}

// NewHub 创建一个新的Hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[int64]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run 启动Hub消息处理循环
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if _, ok := h.clients[client.PollID]; !ok {
				h.clients[client.PollID] = make(map[*Client]bool)
			}
			h.clients[client.PollID][client] = true
			h.mu.Unlock()
			log.Printf("Client registered for poll %d, total clients: %d", client.PollID, len(h.clients[client.PollID]))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.PollID]; ok {
				if _, ok := h.clients[client.PollID][client]; ok {
					delete(h.clients[client.PollID], client)
					close(client.send)
					if len(h.clients[client.PollID]) == 0 {
						delete(h.clients, client.PollID)
					}
				}
			}
			h.mu.Unlock()
			log.Printf("Client unregistered for poll %d", client.PollID)
		}
	}
}

// BroadcastToPoll 向特定投票的所有连接客户端广播消息
func (h *Hub) BroadcastToPoll(pollID int64, message *model.WebSocketMessage) {
	// 将消息转换为JSON
	payload, err := message.ToJSON()
	if err != nil {
		log.Printf("Error converting message to JSON: %v", err)
		return
	}

	// 广播消息
	h.mu.RLock()
	clients := h.clients[pollID]
	h.mu.RUnlock()

	for client := range clients {
		select {
		case client.send <- payload:
		default:
			// 如果客户端的发送缓冲区已满，关闭连接
			h.mu.Lock()
			delete(h.clients[pollID], client)
			close(client.send)
			if len(h.clients[pollID]) == 0 {
				delete(h.clients, pollID)
			}
			h.mu.Unlock()
		}
	}
	log.Printf("Broadcast message to %d clients for poll %d", len(clients), pollID)
}

// RegisterClient 注册客户端到Hub
func (h *Hub) RegisterClient(client *Client) {
	h.register <- client
}

// UnregisterClient 从Hub中注销客户端
func (h *Hub) UnregisterClient(client *Client) {
	h.unregister <- client
}
