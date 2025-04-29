package routes

import (
	"log"
	"net/http"
	"os"
	"time"

	"realtime-voting-backend/handlers"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// Server 是HTTP服务器的封装
type Server struct {
	*http.Server
}

// SetupRouter 设置和配置Gin路由
func SetupRouter() *gin.Engine {
	// 创建Gin路由器
	router := gin.Default()

	// 配置CORS中间件
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // 生产环境中应限制为前端域名
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// 初始化限流器
	handlers.InitRateLimiters()

	// 启动轮询过期检查器
	go startPollExpirationChecker()

	// 定义API路由
	api := router.Group("/api")
	{
		// 全局API限流中间件
		api.Use(handlers.RateLimitMiddleware())

		// 健康检查和指标端点
		api.GET("/health", handlers.HealthCheck)
		api.GET("/status", handlers.SystemStatus)
		api.GET("/metrics", handlers.MetricsHandler)

		// 投票管理端点
		polls := api.Group("/polls")
		{
			polls.POST("", handlers.CreatePoll)
			polls.GET("", handlers.GetPolls)
			polls.GET("/:id", handlers.GetPoll)
			polls.PUT("/:id", handlers.UpdatePoll)
			polls.DELETE("/:id", handlers.DeletePoll)
			polls.POST("/:id/vote", handlers.SubmitVote)

			// 增强版投票端点 - 使用幂等性控制等高级特性
			polls.POST("/:id/vote/enhanced", handlers.SubmitEnhancedVote)

			// 重置投票计数
			polls.POST("/:id/reset", handlers.ResetPollVotes)

			// 实时更新端点（WebSocket和SSE）
			polls.GET("/:id/ws", handlers.HandleWebSocket) // WebSocket方式
			polls.GET("/:id/live", handlers.HandleSSE)     // SSE方式
		}

		// 管理员相关API
		admin := api.Group("/admin")
		{
			admin.POST("/polls/:id/reset", handlers.ResetPollVotes)
			admin.POST("/cache/clean", handlers.CleanupRedisCache)
		}

		// 高并发处理示例路由
		highConcurrency := api.Group("/hc")
		{
			// 布隆过滤器示例
			highConcurrency.GET("/poll/:id/exists", handlers.CheckPollExists)

			// 分布式锁示例
			highConcurrency.POST("/resource/:id/update", handlers.UpdateWithLock)

			// 热点缓存示例
			highConcurrency.GET("/poll/:id/hot", handlers.GetHotPoll)

			// 限流器管理API
			highConcurrency.GET("/ratelimit/stats", handlers.GetRateLimiterStats)
			highConcurrency.POST("/ratelimit/config", handlers.UpdateRateLimiterConfig)
		}
	}

	return router
}

// StartServer 启动HTTP服务器
func StartServer(router *gin.Engine) *Server {
	// 从环境变量获取端口，默认为8090
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8090" // 默认端口
	}

	addr := ":" + port

	srv := &Server{
		&http.Server{
			Addr:    addr,
			Handler: router,
		},
	}

	// 在单独的goroutine中启动服务器
	go func() {
		log.Printf("服务器启动在 %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务器启动失败: %v", err)
		}
	}()

	return srv
}

// startPollExpirationChecker 检查并关闭过期的投票
func startPollExpirationChecker() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		handlers.CheckAndCloseExpiredPolls()
	}
}
