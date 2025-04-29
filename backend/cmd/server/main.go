package main

import (
	"context"
	"database/sql"

	"realtime-voting-backend/api"
	"realtime-voting-backend/cache"
	"realtime-voting-backend/repository"
	"realtime-voting-backend/service"
	"realtime-voting-backend/websocket"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
)

// Config 服务器配置
type Config struct {
	DB          *sql.DB
	Redis       *redis.Client
	BloomFilter *cache.BloomFilter
	RateLimiter *rate.Limiter
}

// initServer 初始化服务器
func initServer(ctx context.Context, config *Config) (*gin.Engine, error) {
	router := gin.Default()

	// 初始化WebSocket Hub
	wsHub := websocket.NewHub()
	go wsHub.Run()

	// 初始化数据仓库
	pollRepo := repository.NewCachedPollRepository(
		repository.NewPollRepositoryImpl(config.DB),
		repository.NewPollCacheRepository(config.Redis),
		config.BloomFilter,
	)

	// 初始化服务
	pollService := service.NewPollService(pollRepo, config.RateLimiter, wsHub)

	// 初始化控制器
	pollController := api.NewPollController(pollService)
	wsHandler := websocket.NewHandler(wsHub)

	// 注册路由
	pollController.RegisterRoutes(router)
	wsHandler.RegisterRoutes(router)

	return router, nil
}

func main() {
	// ... existing code ...

	// ... existing code ...
}
