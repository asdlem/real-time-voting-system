package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"realtime-voting-backend/cache"

	"github.com/gin-gonic/gin"
)

// CleanupCacheInput 定义清理缓存的输入结构
type CleanupCacheInput struct {
	AdminKey string   `json:"admin_key" binding:"required"`
	Patterns []string `json:"patterns" binding:"required"` // 要清理的键模式列表
}

// CleanupRedisCache 清理Redis缓存
func CleanupRedisCache(c *gin.Context) {
	var input CleanupCacheInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("无效的输入: %v", err)})
		return
	}

	// 验证管理员密钥
	if input.AdminKey != "admin123" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的管理员密钥"})
		return
	}

	// 获取Redis客户端
	redisClient, err := cache.GetClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("获取Redis客户端失败: %v", err)})
		return
	}

	if redisClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Redis客户端未初始化"})
		return
	}

	// 设置超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 清理指定模式的键
	totalDeleted := 0
	errors := []string{}

	for _, pattern := range input.Patterns {
		keys, err := redisClient.Keys(ctx, pattern).Result()
		if err != nil {
			errors = append(errors, fmt.Sprintf("查找键失败 (模式: %s): %v", pattern, err))
			log.Printf("查找Redis键失败 (模式: %s): %v", pattern, err)
			continue
		}

		if len(keys) > 0 {
			deletedCount, err := redisClient.Del(ctx, keys...).Result()
			if err != nil {
				errors = append(errors, fmt.Sprintf("删除键失败 (模式: %s): %v", pattern, err))
				log.Printf("删除Redis键失败 (模式: %s): %v", pattern, err)
			} else {
				log.Printf("已删除 %d 个Redis键 (模式: %s)", deletedCount, pattern)
				totalDeleted += int(deletedCount)
			}
		} else {
			log.Printf("未找到匹配模式 %s 的Redis键", pattern)
		}
	}

	log.Printf("缓存清理完成，总共删除了 %d 个键", totalDeleted)

	// 返回结果
	result := gin.H{
		"success":       len(errors) == 0,
		"total_deleted": totalDeleted,
		"message":       fmt.Sprintf("缓存清理完成，总共删除了 %d 个键", totalDeleted),
	}

	if len(errors) > 0 {
		result["errors"] = errors
	}

	c.JSON(http.StatusOK, result)
}
