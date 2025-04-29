package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"realtime-voting-backend/cache"
	"realtime-voting-backend/models"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// Hub 管理WebSocket连接的中心
type Hub struct {
	// 分组存储的客户端连接，按投票ID组织
	clients map[uint]map[*Client]bool

	// 添加新客户端的注册通道
	register chan *Client

	// 删除客户端的注销通道
	unregister chan *Client

	// 广播特定投票的更新消息
	broadcast chan *BroadcastMessage

	// 锁，用于保护clients字典
	mu sync.RWMutex

	// 用于跟踪每个投票的连接数
	pollConnections map[uint]int

	// 定期清理过期连接
	expireTicker *time.Ticker

	// 最大连接数限制
	maxConnections int

	// 当前连接总数
	totalConnections int

	// 消息历史缓存，用于新连接时的初始同步和重试
	messageHistory map[uint]map[string][]byte

	// 消息历史锁
	historyMu sync.RWMutex

	// 历史消息保留时间
	historyRetention time.Duration

	// 消息清理计时器
	historyCleanupTicker *time.Ticker
}

// Client 表示一个WebSocket客户端连接
type Client struct {
	// 所属Hub
	hub *Hub

	// WebSocket连接
	conn *websocket.Conn

	// 发送消息的通道
	send chan []byte

	// 客户端关注的投票ID
	pollID uint

	// 客户端上次活动时间
	lastActivity time.Time

	// 是否为keepalive连接
	isKeepalive bool
}

// BroadcastMessage 定义广播消息的结构
type BroadcastMessage struct {
	PollID  uint        `json:"poll_id"`
	Results interface{} `json:"results"`
}

// 定义WebSocket升级器
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 允许所有CORS请求，生产环境应限制
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// 全局Hub实例
var (
	GlobalHub *Hub
	hubOnce   sync.Once
)

// 初始化函数，创建并启动Hub
func init() {
	// 确保Hub只被初始化一次
	hubOnce.Do(func() {
		GlobalHub = &Hub{
			clients:              make(map[uint]map[*Client]bool),
			register:             make(chan *Client),
			unregister:           make(chan *Client),
			broadcast:            make(chan *BroadcastMessage),
			pollConnections:      make(map[uint]int),
			expireTicker:         time.NewTicker(5 * time.Minute),
			maxConnections:       10000, // 默认最大连接数
			messageHistory:       make(map[uint]map[string][]byte),
			historyRetention:     5 * time.Minute, // 保留消息历史5分钟
			historyCleanupTicker: time.NewTicker(1 * time.Minute),
		}
		go GlobalHub.run()
	})
}

// run 运行Hub处理循环
func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			// 注册新客户端
			h.mu.Lock()
			// 初始化该投票的客户端映射（如果不存在）
			if _, ok := h.clients[client.pollID]; !ok {
				h.clients[client.pollID] = make(map[*Client]bool)
				h.pollConnections[client.pollID] = 0
			}

			// 添加客户端并更新计数
			h.clients[client.pollID][client] = true
			h.pollConnections[client.pollID]++
			h.totalConnections++
			connCount := h.pollConnections[client.pollID]
			totalCount := h.totalConnections
			h.mu.Unlock()

			log.Printf("新WebSocket客户端已连接 [Poll ID: %d, 连接数: %d, 总连接: %d]",
				client.pollID, connCount, totalCount)

			// 发送历史消息以确保新客户端同步到最新状态
			h.sendHistoryToClient(client)

		case client := <-h.unregister:
			// 注销客户端
			h.mu.Lock()
			// 检查该投票的客户端映射是否存在
			if _, ok := h.clients[client.pollID]; ok {
				// 检查该客户端是否在映射中
				if _, ok := h.clients[client.pollID][client]; ok {
					// 删除客户端
					delete(h.clients[client.pollID], client)
					h.pollConnections[client.pollID]--
					h.totalConnections--

					// 关闭客户端发送通道
					close(client.send)

					log.Printf("WebSocket客户端已断开 [Poll ID: %d, 连接数: %d]",
						client.pollID, h.pollConnections[client.pollID])

					// 如果该投票没有连接了，清理映射
					if len(h.clients[client.pollID]) == 0 {
						delete(h.clients, client.pollID)
						delete(h.pollConnections, client.pollID)
					}
				}
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			// 广播消息给关注特定投票的所有客户端
			h.mu.RLock()
			clients := h.clients[message.PollID]
			clientCount := len(clients)
			h.mu.RUnlock()

			// 将结果序列化为JSON
			data, err := json.Marshal(message.Results)
			if err != nil {
				log.Printf("序列化广播消息失败: %v", err)
				continue
			}

			// 存储消息到历史记录
			h.storeMessageInHistory(message.PollID, data)

			// 如果没有客户端，直接跳过广播但保留历史
			if clientCount == 0 {
				log.Printf("没有已连接的客户端接收广播 [Poll ID: %d], 已将消息保存到历史", message.PollID)
				continue
			}

			// 计数成功发送和失败的客户端
			successCount := 0
			failureCount := 0

			// 广播给所有关注该投票的客户端
			h.mu.RLock()
			for client := range clients {
				select {
				case client.send <- data:
					// 消息发送成功
					successCount++
				default:
					// 客户端缓冲区已满，关闭连接
					failureCount++
					close(client.send)
					delete(h.clients[message.PollID], client)
					h.pollConnections[message.PollID]--
					h.totalConnections--
				}
			}
			h.mu.RUnlock()

			log.Printf("广播更新到 %d 个WebSocket客户端 [Poll ID: %d], 成功: %d, 失败: %d",
				clientCount, message.PollID, successCount, failureCount)

		case <-h.historyCleanupTicker.C:
			// 清理过期的历史消息
			h.cleanupOldMessages()

		case <-h.expireTicker.C:
			// 清理长时间不活跃的连接
			now := time.Now()
			timeout := 30 * time.Minute

			h.mu.Lock()
			for pollID, clients := range h.clients {
				for client := range clients {
					if client.lastActivity.Add(timeout).Before(now) {
						log.Printf("关闭不活跃的WebSocket连接 [Poll ID: %d, 不活跃时间: %v]",
							pollID, now.Sub(client.lastActivity))
						delete(clients, client)
						h.pollConnections[pollID]--
						h.totalConnections--
						close(client.send)
					}
				}

				// 如果该投票没有连接了，清理映射
				if len(clients) == 0 {
					delete(h.clients, pollID)
					delete(h.pollConnections, pollID)
				}
			}
			h.mu.Unlock()
		}
	}
}

// 在历史记录中存储消息
func (h *Hub) storeMessageInHistory(pollID uint, data []byte) {
	h.historyMu.Lock()
	defer h.historyMu.Unlock()

	if _, ok := h.messageHistory[pollID]; !ok {
		h.messageHistory[pollID] = make(map[string][]byte)
	}

	// 使用时间戳作为消息唯一标识
	messageID := fmt.Sprintf("%d", time.Now().UnixNano())
	h.messageHistory[pollID][messageID] = data

	// 限制每个投票的历史消息数量
	const maxHistoryPerPoll = 5
	if len(h.messageHistory[pollID]) > maxHistoryPerPoll {
		// 找到最旧的消息并删除
		var oldestID string
		var oldestTime int64 = math.MaxInt64

		for id := range h.messageHistory[pollID] {
			// 尝试将ID解析为时间戳
			if ts, err := strconv.ParseInt(id, 10, 64); err == nil {
				if ts < oldestTime {
					oldestTime = ts
					oldestID = id
				}
			}
		}

		if oldestID != "" {
			delete(h.messageHistory[pollID], oldestID)
		}
	}
}

// 发送历史消息给新客户端
func (h *Hub) sendHistoryToClient(client *Client) {
	h.historyMu.RLock()
	defer h.historyMu.RUnlock()

	pollHistory, exists := h.messageHistory[client.pollID]
	if !exists || len(pollHistory) == 0 {
		return
	}

	log.Printf("向新客户端发送 %d 条历史消息 [Poll ID: %d]", len(pollHistory), client.pollID)

	// 找到最新的消息发送给客户端
	var newestMessage []byte
	var newestTime int64 = 0

	for id, data := range pollHistory {
		// 尝试将ID解析为时间戳
		if ts, err := strconv.ParseInt(id, 10, 64); err == nil {
			if ts > newestTime {
				newestTime = ts
				newestMessage = data
			}
		}
	}

	if newestMessage != nil {
		select {
		case client.send <- newestMessage:
			log.Printf("成功向新客户端发送最新状态 [Poll ID: %d]", client.pollID)
		default:
			log.Printf("无法向新客户端发送历史消息 [Poll ID: %d]", client.pollID)
		}
	}
}

// 清理旧消息
func (h *Hub) cleanupOldMessages() {
	h.historyMu.Lock()
	defer h.historyMu.Unlock()

	cutoffTime := time.Now().Add(-h.historyRetention)
	cutoffNano := cutoffTime.UnixNano()

	for pollID, messages := range h.messageHistory {
		for id := range messages {
			// 尝试将ID解析为时间戳
			if ts, err := strconv.ParseInt(id, 10, 64); err == nil {
				if ts < cutoffNano {
					delete(messages, id)
				}
			}
		}

		// 如果该投票没有历史消息了，删除该投票的条目
		if len(messages) == 0 {
			delete(h.messageHistory, pollID)
		}
	}
}

// HandleWebSocket 处理WebSocket连接
func HandleWebSocket(c *gin.Context) {
	// 获取投票ID
	pollIDStr := c.Param("id")
	pollID, err := strconv.ParseUint(pollIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的投票ID"})
		return
	}

	// 检查是否有keepalive参数
	keepalive := c.Query("keepalive") == "true"

	// 检查连接数量是否达到上限
	GlobalHub.mu.RLock()
	if GlobalHub.totalConnections >= GlobalHub.maxConnections {
		GlobalHub.mu.RUnlock()
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "服务器连接已达上限，请稍后重试"})
		return
	}
	GlobalHub.mu.RUnlock()

	// 打印连接详情
	log.Printf("正在建立WebSocket连接 [Poll ID: %d, keepalive: %v]", pollID, keepalive)

	// 升级HTTP连接为WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("升级WebSocket连接失败: %v", err)
		return
	}

	// 创建新客户端
	client := &Client{
		hub:          GlobalHub,
		conn:         conn,
		send:         make(chan []byte, 256),
		pollID:       uint(pollID),
		lastActivity: time.Now(),
		isKeepalive:  keepalive, // 存储keepalive状态
	}

	// 设置更长的连接保持时间，如果请求了keepalive
	if keepalive {
		// 在投票后保持活跃的连接设置为3小时
		client.conn.SetReadDeadline(time.Now().Add(3 * time.Hour))

		// 发送欢迎消息，通知客户端连接已建立
		welcomeMsg := map[string]interface{}{
			"type":    "CONNECT_SUCCESS",
			"message": "连接已建立，将接收实时更新",
		}

		if msgData, err := json.Marshal(welcomeMsg); err == nil {
			// 直接写入消息，无需通过hub
			client.conn.WriteMessage(websocket.TextMessage, msgData)
		}
	}

	// 注册客户端
	client.hub.register <- client

	// 启动客户端goroutines
	go client.writePump()
	go client.readPump()
}

// BroadcastPollUpdate 广播投票更新给所有关注该投票的客户端
func BroadcastPollUpdate(pollID uint, results interface{}) {
	// 确保结果是一个数组格式，便于前端处理
	var formattedResults []map[string]interface{}

	// 处理不同类型的results输入
	switch v := results.(type) {
	case []PollOptionResult:
		// 已经是PollOptionResult数组，转换为通用格式
		formattedResults = make([]map[string]interface{}, len(v))
		for i, result := range v {
			formattedResults[i] = map[string]interface{}{
				"id":    result.ID,
				"text":  result.Text,
				"votes": result.Votes,
				// 百分比由前端计算，不发送
			}
		}
	case []OptionResult:
		// OptionResult数组，转换为通用格式
		formattedResults = make([]map[string]interface{}, len(v))
		for i, result := range v {
			formattedResults[i] = map[string]interface{}{
				"id":    result.ID,
				"text":  result.Text,
				"votes": result.Votes,
				// 百分比由前端计算，不发送
			}
		}
	case []models.PollOption:
		// 原始PollOption数组，转换为通用格式
		formattedResults = make([]map[string]interface{}, len(v))
		for i, option := range v {
			formattedResults[i] = map[string]interface{}{
				"id":    option.ID,
				"text":  option.Text,
				"votes": option.Votes,
			}
		}
	case map[string]int64:
		// 如果是从Redis获取的键值对，转换为数组
		formattedResults = make([]map[string]interface{}, 0, len(v))
		for optionID, count := range v {
			// 转换optionID从字符串到数字
			id, err := strconv.ParseUint(optionID, 10, 32)
			if err != nil {
				continue
			}
			formattedResults = append(formattedResults, map[string]interface{}{
				"id":    uint(id),
				"votes": count,
			})
		}
	default:
		// 如果无法识别类型，则记录日志但继续广播
		log.Printf("WebSocket广播：无法识别的results类型: %T", results)

		// 尝试使用反射遍历数组
		if formattedResults == nil {
			log.Printf("尝试使用通用方法处理结果: %v", results)
			// 使用json序列化再反序列化为通用格式
			data, err := json.Marshal(results)
			if err == nil {
				var genericResults []map[string]interface{}
				if err := json.Unmarshal(data, &genericResults); err == nil {
					formattedResults = genericResults
				}
			}
		}
	}

	// 如果仍然无法格式化结果，使用空数组
	if formattedResults == nil {
		formattedResults = []map[string]interface{}{}
	}

	// 创建符合前端预期的消息格式
	formattedMessage := map[string]interface{}{
		"type": "VOTE_UPDATE",
		"data": map[string]interface{}{
			"poll_id":   pollID,
			"options":   formattedResults,
			"timestamp": time.Now().UnixNano(), // 添加时间戳以便客户端判断消息顺序
		},
	}

	log.Printf("WebSocket广播投票更新: 投票ID=%d, 数据结构=%T, 选项数=%d",
		pollID, formattedResults, len(formattedResults))

	message := &BroadcastMessage{
		PollID:  pollID,
		Results: formattedMessage,
	}

	// 使用goroutine异步发送广播，避免阻塞主流程
	go func() {
		// 重试逻辑：如果广播失败，等待短暂时间后重试
		maxRetries := 2
		for retry := 0; retry <= maxRetries; retry++ {
			select {
			case GlobalHub.broadcast <- message:
				if retry > 0 {
					log.Printf("WebSocket广播成功 (重试 %d): Poll ID=%d", retry, pollID)
				}
				return
			default:
				if retry < maxRetries {
					log.Printf("WebSocket广播通道已满，等待重试 (%d/%d): Poll ID=%d",
						retry+1, maxRetries, pollID)
					time.Sleep(time.Duration(20*(retry+1)) * time.Millisecond)
				} else {
					log.Printf("WebSocket广播失败，达到最大重试次数: Poll ID=%d", pollID)
				}
			}
		}
	}()
}

// BroadcastPollUpdateStr 将投票更新广播给所有连接的客户端（支持字符串ID）
func BroadcastPollUpdateStr(pollIDStr string, results interface{}) {
	log.Printf("尝试广播投票更新 (字符串 PollID): %s", pollIDStr)

	// 将 pollIDStr 转换为 uint
	pollIDUint64, err := strconv.ParseUint(pollIDStr, 10, 64)
	if err != nil {
		log.Printf("无法将 pollIDStr '%s' 转换为 uint: %v", pollIDStr, err)
		return // 如果转换失败，则不继续执行
	}
	pollID := uint(pollIDUint64) // 将 uint64 转换为 uint

	// 如果结果为空，尝试从缓存获取
	if results == nil {
		var err error
		results, err = cache.GetVoteCounts(pollIDStr) // 注意：GetVoteCounts 可能仍需要 string
		if err != nil {
			log.Printf("获取投票结果失败: %v", err)
			// 错误情况下使用空结果数组继续执行
			results = []PollOptionResult{}
		}
	}

	// 使用统一的方法广播，复用BroadcastPollUpdate的逻辑
	BroadcastPollUpdate(pollID, results)
}

// 客户端读取循环
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	// 配置连接
	c.conn.SetReadLimit(512) // 限制消息大小

	// 根据keepalive标志设置超时时间
	if c.isKeepalive {
		// 设置为更长的超时时间
		c.conn.SetReadDeadline(time.Now().Add(3 * time.Hour))
	} else {
		// 默认超时时间
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	}

	c.conn.SetPongHandler(func(string) error {
		// 收到pong更新活动时间和截止时间
		c.lastActivity = time.Now()

		// 对于keepalive连接，使用更长的超时时间
		if c.isKeepalive {
			c.conn.SetReadDeadline(time.Now().Add(3 * time.Hour))
		} else {
			c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		}
		return nil
	})

	// 持续读取消息
	for {
		messageType, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket读取错误: %v", err)
			}
			break
		}

		// 更新最后活动时间
		c.lastActivity = time.Now()

		// 处理客户端消息，特别是ping
		if messageType == websocket.TextMessage {
			var msg map[string]interface{}
			if err := json.Unmarshal(message, &msg); err == nil {
				// 检查是否为ping消息
				if msgType, ok := msg["type"].(string); ok && msgType == "PING" {
					// 回复一个pong消息
					pongMsg := map[string]string{
						"type": "PONG",
						"time": time.Now().Format(time.RFC3339),
					}
					if pongData, err := json.Marshal(pongMsg); err == nil {
						c.conn.WriteMessage(websocket.TextMessage, pongData)
					}
				}
			}
		}
	}
}

// 客户端写入循环
func (c *Client) writePump() {
	ticker := time.NewTicker(60 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	// 如果是keepalive连接，使用更频繁的ping间隔
	if c.isKeepalive {
		// 每30秒ping一次
		ticker.Stop()
		ticker = time.NewTicker(30 * time.Second)
	}

	for {
		select {
		case message, ok := <-c.send:
			// 设置写入超时
			writeTimeout := 10 * time.Second
			if c.isKeepalive {
				writeTimeout = 30 * time.Second
			}
			c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))

			if !ok {
				// Hub关闭了channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// 更新最后活动时间
			c.lastActivity = time.Now()

			// 写入消息
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 添加排队的消息
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			// 发送ping保持连接活跃
			pingTimeout := 10 * time.Second
			if c.isKeepalive {
				pingTimeout = 30 * time.Second
			}
			c.conn.SetWriteDeadline(time.Now().Add(pingTimeout))

			// 对于keepalive连接，发送JSON ping而不是WebSocket ping
			if c.isKeepalive {
				pingMsg := map[string]string{
					"type": "PING",
					"time": time.Now().Format(time.RFC3339),
				}
				if pingData, err := json.Marshal(pingMsg); err == nil {
					if err := c.conn.WriteMessage(websocket.TextMessage, pingData); err != nil {
						return
					}
				}
			} else {
				// 普通WebSocket ping
				if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			}
		}
	}
}
