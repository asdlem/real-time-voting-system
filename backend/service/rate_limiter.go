package service

import (
	"context"
)

// RateLimiter 定义限流器接口
type RateLimiter interface {
	// IsAllowed 检查请求是否允许通过
	IsAllowed(ctx context.Context, key string) (bool, error)

	// UpdateConfig 更新限流配置
	UpdateConfig(ctx context.Context, config map[string]interface{}) error

	// GetStatistics 获取限流统计信息
	GetStatistics(ctx context.Context) (map[string]interface{}, error)
}
