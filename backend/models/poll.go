package models

import (
	"time"

	"gorm.io/gorm"
)

// PollType defines the type of the poll (single or multiple choice)
// We use iota for enum-like behavior
type PollType int

const (
	SingleChoice PollType = iota // 0
	MultiChoice                  // 1
)

// Poll represents a voting poll
type Poll struct {
	gorm.Model               // Includes fields like ID, CreatedAt, UpdatedAt, DeletedAt
	Question    string       `gorm:"not null" json:"question"`
	Description string       `gorm:"type:text" json:"description"`        // 添加描述字段
	PollType    PollType     `gorm:"not null;default:0" json:"poll_type"` // 0 for single, 1 for multi
	Options     []PollOption `gorm:"foreignKey:PollID" json:"options"`
	IsActive    bool         `gorm:"default:true" json:"is_active"` // To easily enable/disable voting
	EndTime     *time.Time   `json:"end_time,omitempty"`            // Optional end date for the poll
}

// PollOption represents an option within a poll
type PollOption struct {
	gorm.Model
	PollID uint   `gorm:"not null;index" json:"poll_id"`
	Text   string `gorm:"not null" json:"text"`
	Votes  int64  `gorm:"default:0" json:"votes"`
}
