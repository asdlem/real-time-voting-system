package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"realtime-voting-backend/database"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// 客户端SSE连接管理
type SSEClient struct {
	PollID  uint
	Writer  http.ResponseWriter
	Flusher http.Flusher
	Done    chan bool
}

var (
	// sseClients存储所有SSE连接，按投票ID进行分组
	sseClients      = make(map[uint][]*SSEClient)
	sseClientsMutex = make(chan bool, 1) // 简单的互斥锁实现
)

// HandleSSE处理SSE连接请求
func HandleSSE(c *gin.Context) {
	// 获取投票ID
	pollIDStr := c.Param("id")
	pollID, err := strconv.ParseUint(pollIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的投票ID格式"})
		return
	}
	pollUintID := uint(pollID)

	// 直接使用SQL语句原样执行
	var exists int64
	sql := "SELECT COUNT(*) FROM polls WHERE id = ? AND deleted_at IS NULL"
	log.Printf("执行SQL: %s [参数: %d]", sql, pollUintID)

	err = database.DB.Raw(sql, pollUintID).Count(&exists).Error

	if err != nil {
		log.Printf("数据库错误: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库查询出错"})
		return
	}

	if exists == 0 {
		log.Printf("找不到投票ID: %d", pollUintID)
		c.JSON(http.StatusNotFound, gin.H{"error": "投票不存在"})
		return
	}

	log.Printf("成功找到投票，ID: %d", pollUintID)

	// 设置SSE所需的HTTP头
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	c.Writer.Header().Set("X-Accel-Buffering", "no") // 禁用Nginx缓冲

	// 获取Flusher接口
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "不支持流式响应"})
		return
	}

	// 创建新客户端
	client := &SSEClient{
		PollID:  pollUintID,
		Writer:  c.Writer,
		Flusher: flusher,
		Done:    make(chan bool),
	}

	// 注册客户端
	sseClientsMutex <- true // 获取锁
	sseClients[pollUintID] = append(sseClients[pollUintID], client)
	<-sseClientsMutex // 释放锁

	log.Printf("已注册SSE客户端，投票ID: %d，客户端IP: %s", pollUintID, c.ClientIP())

	// 发送初始数据
	results, err := GetCurrentPollResults(pollUintID)
	if err == nil {
		log.Printf("获取初始数据成功，选项数量: %d", len(results))
		sendSSEEvent(client, results)
	} else {
		log.Printf("发送初始数据失败: %v", err)
	}

	// 发送初始连接确认
	sendSSEEvent(client, map[string]string{"status": "connected", "message": "SSE连接已建立"})

	// 设置定时发送心跳的goroutine
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	// 设置关闭通知
	notify := c.Request.Context().Done()

	// 保持连接直到客户端断开
	go func() {
		for {
			select {
			case <-notify:
				// 客户端断开连接
				log.Printf("SSE客户端已断开连接, 投票ID: %d", pollUintID)
				client.Done <- true
				return
			case <-client.Done:
				// 服务端关闭连接
				log.Printf("服务端关闭SSE连接, 投票ID: %d", pollUintID)
				return
			case <-heartbeat.C:
				// 发送心跳
				err := sendSSEEvent(client, map[string]string{"type": "heartbeat", "time": time.Now().Format(time.RFC3339)})
				if err != nil {
					log.Printf("发送心跳失败，关闭连接: %v", err)
					client.Done <- true
					return
				}
			}
		}
	}()

	// 等待连接关闭
	<-client.Done

	// 注销客户端
	unregisterSSEClient(client)
}

// 从列表中删除客户端
func unregisterSSEClient(client *SSEClient) {
	sseClientsMutex <- true              // 获取锁
	defer func() { <-sseClientsMutex }() // 释放锁

	clients := sseClients[client.PollID]
	for i, c := range clients {
		if c == client {
			// 从列表中移除
			sseClients[client.PollID] = append(clients[:i], clients[i+1:]...)
			break
		}
	}

	// 如果该投票没有更多客户端，清理映射
	if len(sseClients[client.PollID]) == 0 {
		delete(sseClients, client.PollID)
	}

	log.Printf("已注销SSE客户端，当前连接数: %d", len(sseClients[client.PollID]))
}

// 向单个SSE客户端发送事件
func sendSSEEvent(client *SSEClient, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("序列化数据失败，投票ID %d: %v", client.PollID, err)
		return err
	}

	// 构建SSE事件
	event := fmt.Sprintf("data: %s\n\n", jsonData)

	// 写入事件数据
	_, err = fmt.Fprint(client.Writer, event)
	if err != nil {
		log.Printf("写入SSE数据失败，投票ID %d: %v", client.PollID, err)
		return err
	}

	// 刷新缓冲区
	client.Flusher.Flush()
	return nil
}

// BroadcastSSEUpdate向所有监听特定投票的SSE客户端广播更新
func BroadcastSSEUpdate(pollID uint, data interface{}) {
	sseClientsMutex <- true // 获取锁
	clients := sseClients[pollID]
	<-sseClientsMutex // 释放锁

	if len(clients) == 0 {
		return // 没有客户端监听
	}

	log.Printf("通过SSE广播更新给%d个客户端, 投票ID: %d", len(clients), pollID)

	// 向所有客户端发送更新
	for _, client := range clients {
		sendSSEEvent(client, data)
	}
}

// 定期发送心跳以保持连接
func init() {
	go func() {
		for {
			time.Sleep(30 * time.Second)

			sseClientsMutex <- true // 获取锁
			for pollID, clients := range sseClients {
				for _, client := range clients {
					// 发送注释作为心跳
					_, err := fmt.Fprint(client.Writer, ": ping\n\n")
					if err != nil {
						log.Printf("心跳发送失败，投票ID %d: %v", pollID, err)
						client.Done <- true
						continue
					}
					client.Flusher.Flush()
				}
			}
			<-sseClientsMutex // 释放锁
		}
	}()
}
