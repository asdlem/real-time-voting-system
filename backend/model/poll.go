package model

import (
	"encoding/json"
	"time"
)

// Poll 投票活动模型
type Poll struct {
	ID          int64      `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	StartTime   time.Time  `json:"start_time"`
	EndTime     time.Time  `json:"end_time"`
	Status      PollStatus `json:"status"`
	Options     []Option   `json:"options,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// PollStatus 投票活动状态
type PollStatus string

const (
	PollStatusDraft     PollStatus = "draft"     // 草稿
	PollStatusActive    PollStatus = "active"    // 进行中
	PollStatusPaused    PollStatus = "paused"    // 暂停
	PollStatusCompleted PollStatus = "completed" // 已结束
)

// Option 投票选项模型
type Option struct {
	ID        int64     `json:"id"`
	PollID    int64     `json:"poll_id"`
	Content   string    `json:"content"`
	Count     int64     `json:"count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Vote 投票记录模型
type Vote struct {
	ID        int64     `json:"id"`
	PollID    int64     `json:"poll_id"`
	OptionID  int64     `json:"option_id"`
	UserID    string    `json:"user_id"` // 可以是IP地址或设备ID
	CreatedAt time.Time `json:"created_at"`
}

// PollRequest 创建投票请求
type PollRequest struct {
	Title       string   `json:"title" binding:"required"`
	Description string   `json:"description"`
	StartTime   string   `json:"start_time" binding:"required"`
	EndTime     string   `json:"end_time" binding:"required"`
	Options     []string `json:"options" binding:"required,min=2"`
}

// VoteRequest 提交投票请求
type VoteRequest struct {
	PollID   int64  `json:"poll_id" binding:"required"`
	OptionID int64  `json:"option_id" binding:"required"`
	UserID   string `json:"user_id,omitempty"`
}

// PollStatistics 投票统计结果
type PollStatistics struct {
	PollID      int64             `json:"poll_id"`
	Title       string            `json:"title"`
	TotalVotes  int64             `json:"total_votes"`
	OptionStats []OptionStatistic `json:"option_stats"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// OptionStatistic 选项统计结果
type OptionStatistic struct {
	OptionID   int64   `json:"option_id"`
	Content    string  `json:"content"`
	Count      int64   `json:"count"`
	Percentage float64 `json:"percentage"`
}

// WebSocketMessage 定义WebSocket消息格式
type WebSocketMessage struct {
	Type    string      `json:"type"`    // 消息类型
	PollID  int64       `json:"pollId"`  // 投票ID
	Payload interface{} `json:"payload"` // 消息内容
}

// ToJSON 将WebSocket消息转换为JSON字节数组
func (m *WebSocketMessage) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}
