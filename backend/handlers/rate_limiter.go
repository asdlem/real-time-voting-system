package handlers

import (
	"log"
	"net/http"
	"os"
	"realtime-voting-backend/cache"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// 全局限流器
var (
	globalLimiter     cache.RateLimiter
	userLimiter       *cache.UserRateLimiter
	globalRateLimit   int
	userRateLimit     int
	rateLimitEnabled  bool
	limitStatistics   = make(map[string]int64)
	limitStatsLock    = &sync.RWMutex{}
	rateLimiterConfig = RateLimiterConfig{
		GlobalRate:  100,
		GlobalBurst: 200,
		UserRate:    10,
		UserBurst:   20,
	}
)

// RateLimiterConfig 限流器配置结构
type RateLimiterConfig struct {
	Enabled     bool `json:"enabled"`
	GlobalRate  int  `json:"globalRate"`
	GlobalBurst int  `json:"globalBurst"`
	UserRate    int  `json:"userRate"`
	UserBurst   int  `json:"userBurst"`
}

// RateLimiterStats 限流器统计信息
type RateLimiterStats struct {
	TotalRequests     int64             `json:"totalRequests"`
	AllowedRequests   int64             `json:"allowedRequests"`
	RejectedRequests  int64             `json:"rejectedRequests"`
	UserRequestStats  map[string]int64  `json:"userRequestStats"`
	RateLimiterConfig RateLimiterConfig `json:"config"`
}

// InitRateLimiters 初始化限流器
func InitRateLimiters() {
	// 从环境变量读取配置
	enabledStr := os.Getenv("ENABLE_RATE_LIMIT")
	if enabledStr == "true" {
		rateLimitEnabled = true
	}

	globalRateStr := os.Getenv("GLOBAL_RATE_LIMIT")
	if globalRateStr != "" {
		if rate, err := strconv.Atoi(globalRateStr); err == nil && rate > 0 {
			rateLimiterConfig.GlobalRate = rate
			rateLimiterConfig.GlobalBurst = rate * 2
		}
	}

	userRateStr := os.Getenv("USER_RATE_LIMIT")
	if userRateStr != "" {
		if rate, err := strconv.Atoi(userRateStr); err == nil && rate > 0 {
			rateLimiterConfig.UserRate = rate
			rateLimiterConfig.UserBurst = rate * 2
		}
	}

	rateLimiterConfig.Enabled = rateLimitEnabled

	// 如果启用了限流，初始化限流器
	if rateLimitEnabled {
		resetRateLimiters()
	}
}

// 重置限流器配置
func resetRateLimiters() {
	// 获取Redis客户端
	redisClient, err := cache.GetRedisClient()
	if err != nil {
		log.Printf("无法获取Redis客户端: %v", err)
		return
	}

	// 初始化全局限流器
	globalLimiter = cache.NewTokenBucketRateLimiter(
		redisClient,
		"global_api",
		rateLimiterConfig.GlobalRate,
		rateLimiterConfig.GlobalBurst,
	)

	// 初始化用户级别限流器
	userLimiter = cache.NewUserRateLimiter(
		redisClient,
		"user_api",
		rateLimiterConfig.GlobalRate,
		rateLimiterConfig.GlobalBurst,
		rateLimiterConfig.UserRate,
		rateLimiterConfig.UserBurst,
	)

	// 初始化统计信息
	limitStatsLock.Lock()
	limitStatistics = map[string]int64{
		"total":    0,
		"allowed":  0,
		"rejected": 0,
	}
	limitStatsLock.Unlock()

	log.Printf("限流器已初始化：全局速率=%d/秒，用户速率=%d/秒",
		rateLimiterConfig.GlobalRate, rateLimiterConfig.UserRate)
}

// RateLimitMiddleware 限流中间件
func RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 如果限流未启用，直接通过
		if !rateLimitEnabled || globalLimiter == nil {
			c.Next()
			return
		}

		// 更新统计信息
		limitStatsLock.Lock()
		limitStatistics["total"]++
		limitStatsLock.Unlock()

		// 全局限流检查
		allowed, err := globalLimiter.Allow(c)
		if err != nil || !allowed {
			limitStatsLock.Lock()
			limitStatistics["rejected"]++
			limitStatsLock.Unlock()

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "请求频率过高，请稍后再试",
			})
			c.Abort()
			return
		}

		// 如果有用户ID，进行用户级别限流
		userID := c.GetHeader("X-User-ID")
		if userID != "" && userLimiter != nil {
			allowed, err := userLimiter.AllowUser(c, userID)
			if err != nil || !allowed {
				// 更新用户级别统计信息
				limitStatsLock.Lock()
				limitStatistics["rejected"]++
				userKey := "user:" + userID
				if _, exists := limitStatistics[userKey]; exists {
					limitStatistics[userKey]++
				} else {
					limitStatistics[userKey] = 1
				}
				limitStatsLock.Unlock()

				c.JSON(http.StatusTooManyRequests, gin.H{
					"error": "您的请求频率过高，请稍后再试",
				})
				c.Abort()
				return
			}
		}

		// 更新允许请求的统计信息
		limitStatsLock.Lock()
		limitStatistics["allowed"]++
		limitStatsLock.Unlock()

		c.Next()
	}
}

// GetRateLimiterStats 获取限流器状态
func GetRateLimiterStats(c *gin.Context) {
	// 复制统计信息以避免竞态条件
	limitStatsLock.RLock()
	stats := RateLimiterStats{
		TotalRequests:     limitStatistics["total"],
		AllowedRequests:   limitStatistics["allowed"],
		RejectedRequests:  limitStatistics["rejected"],
		UserRequestStats:  make(map[string]int64),
		RateLimiterConfig: rateLimiterConfig,
	}

	// 提取用户统计信息
	for key, value := range limitStatistics {
		if strings.HasPrefix(key, "user:") {
			stats.UserRequestStats[key] = value
		}
	}
	limitStatsLock.RUnlock()

	c.JSON(http.StatusOK, stats)
}

// UpdateRateLimiterConfig 更新限流器配置
func UpdateRateLimiterConfig(c *gin.Context) {
	var config RateLimiterConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的配置参数"})
		return
	}

	// 验证配置
	if config.GlobalRate <= 0 || config.GlobalBurst <= 0 ||
		config.UserRate <= 0 || config.UserBurst <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "速率和突发值必须大于0"})
		return
	}

	// 更新配置
	rateLimiterConfig = config
	rateLimitEnabled = config.Enabled

	// 如果启用了限流，重置限流器
	if rateLimitEnabled {
		resetRateLimiters()
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "限流器配置已更新",
		"config":  rateLimiterConfig,
	})
}

// 示例：使用布隆过滤器的处理函数
func CheckPollExists(c *gin.Context) {
	pollID := c.Param("id")

	// 获取布隆过滤器
	bloomFilter := cache.InitBloomFilter()
	if bloomFilter == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "布隆过滤器未初始化"})
		return
	}

	// 检查ID是否可能存在
	exists, err := bloomFilter.Contains(c, "poll:"+pollID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "检查失败: " + err.Error()})
		return
	}

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "投票不存在"})
		return
	}

	// ID可能存在，进一步查询数据库
	c.JSON(http.StatusOK, gin.H{"message": "投票可能存在，正在查询详情"})
}

// 示例：使用分布式锁的处理函数
func UpdateWithLock(c *gin.Context) {
	resourceID := c.Param("id")
	lockKey := "update:" + resourceID

	// 获取分布式锁服务
	lockService := cache.GetLockService()
	if lockService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "分布式锁服务未初始化"})
		return
	}

	// 使用分布式锁执行更新
	err := lockService.WithLock(lockKey, 5, func() error {
		// 执行需要分布式锁保护的业务逻辑
		// 例如：更新数据库，避免并发更新冲突
		return nil
	})

	if err != nil {
		if err == cache.ErrLockNotAcquired {
			c.JSON(http.StatusConflict, gin.H{"error": "资源正在被其他请求处理，请稍后再试"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

// 示例：热点数据缓存处理函数
func GetHotPoll(c *gin.Context) {
	pollID := c.Param("id")
	cacheKey := "poll:" + pollID

	// 获取Redis客户端
	redisClient, err := cache.GetRedisClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Redis未初始化"})
		return
	}

	// 获取布隆过滤器和分布式锁
	bloomFilter := cache.InitBloomFilter()
	lockService := cache.GetLockService()

	// 创建热点缓存管理器
	hotCache := cache.NewHotCache(redisClient, lockService, bloomFilter)

	// 使用缓存获取数据，防止缓存击穿
	data, err := hotCache.GetWithCache(c, cacheKey, 3600, func() (interface{}, error) {
		// 这里应该是从数据库加载数据的逻辑
		// 模拟从数据源加载数据
		return map[string]interface{}{
			"id":   pollID,
			"name": "热点投票" + pollID,
		}, nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取数据失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}
