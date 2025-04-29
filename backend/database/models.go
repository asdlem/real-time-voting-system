package database

import (
	"database/sql"
	"time"
)

// Poll 投票结构
type Poll struct {
	ID          string       `db:"id" json:"id"`
	Title       string       `db:"title" json:"title"`
	Description string       `db:"description" json:"description"`
	CreatedAt   time.Time    `db:"created_at" json:"created_at"`
	EndTime     sql.NullTime `db:"end_time" json:"end_time"`
	Options     []Option     `json:"options"`
}

// Option 选项结构
type Option struct {
	ID      string `db:"id" json:"id"`
	Content string `db:"content" json:"content"`
	PollID  string `db:"poll_id" json:"poll_id"`
}
