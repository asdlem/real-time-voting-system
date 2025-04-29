package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"realtime-voting-backend/cache"
	"realtime-voting-backend/database"
	"realtime-voting-backend/handlers"
	"realtime-voting-backend/mq"
	"realtime-voting-backend/routes"
	"strconv"
	"syscall"
	"time"
)

// 全局消息队列适配器
var mqAdapter *mq.MQAdapter

// 初始化Redis客户端，布隆过滤器，分布式锁和限流器
func initCacheAndLimiter() {
	// 初始化Redis
	err := cache.InitRedis()
	if err != nil {
		log.Printf("初始化Redis失败: %v", err)
	}

	// 初始化分布式锁
	cache.InitDistLock()

	// 初始化布隆过滤器
	bloomFilter := cache.InitBloomFilter()
	if bloomFilter != nil {
		log.Println("布隆过滤器初始化成功")
	}

	// 初始化热点缓存
	redisClient, err := cache.GetRedisClient()
	if err == nil {
		lockService := cache.GetLockService()
		hotCache := cache.NewHotCache(redisClient, lockService, bloomFilter)

		// 示例：预热一些热点数据
		go func() {
			hotKeys := []string{"poll:1", "poll:2", "poll:3"}
			hotCache.PrewarmCache(context.Background(), hotKeys, func(key string) (interface{}, error) {
				// 这里应该是从数据库加载数据的逻辑
				return map[string]interface{}{
					"id":   key,
					"name": "热点投票" + key,
				}, nil
			}, 1*time.Hour)
		}()

		// 示例：启动热点数据刷新器
		hotCache.StartHotDataRefresher(5*time.Minute,
			func() ([]string, error) {
				// 获取热点键的逻辑
				return []string{"poll:1", "poll:2"}, nil
			},
			func(key string) (interface{}, error) {
				// 刷新数据的逻辑
				return map[string]interface{}{
					"id":      key,
					"name":    "热点投票" + key,
					"updated": time.Now().String(),
				}, nil
			})

		log.Println("热点缓存管理器初始化成功")
	}

	// 初始化限流器
	if err == nil {
		// 全局限流器：每秒100个请求，最大突发200
		_ = cache.NewTokenBucketRateLimiter(redisClient, "global", 100, 200)

		// 用户级别限流：全局每秒100请求，每个用户每秒10请求
		_ = cache.NewUserRateLimiter(redisClient, "vote_api", 100, 200, 10, 20)

		// 注册到全局变量或其他地方供API使用
		log.Println("限流器初始化成功")
	}
}

func main() {
	// 初始化数据库连接
	err := database.InitDB()
	if err != nil {
		log.Fatalf("无法初始化数据库: %v", err)
	}
	log.Println("数据库连接初始化成功")

	// 初始化Redis连接
	err = cache.InitRedis()
	if err != nil {
		log.Printf("警告: Redis初始化失败: %v", err)
	} else {
		log.Println("Redis连接初始化成功")
	}

	// 初始化消息队列适配器（自动选择RocketMQ或Redis MQ）
	mqAdapter = mq.NewMQAdapter()
	err = mqAdapter.Initialize()
	if err != nil {
		log.Printf("警告: 消息队列初始化失败，将使用内存模式: %v", err)
	}

	// 注册消息处理函数
	err = mqAdapter.RegisterHandler(updateVoteInDatabase)
	if err != nil {
		log.Printf("警告: 注册消息处理函数失败: %v", err)
	} else {
		log.Println("消息队列处理函数注册成功")
	}

	// 将消息队列适配器传递给处理程序
	handlers.InitHandler(mqAdapter)
	log.Println("已将消息队列适配器传递给处理程序")

	// 设置路由
	router := routes.SetupRouter()
	log.Println("路由设置完成")

	// 启动服务器
	srv := routes.StartServer(router)
	log.Println("服务器启动成功")

	// 输出消息队列状态
	stats := mqAdapter.GetQueueStats()
	log.Printf("消息队列状态: %v", stats)

	// 初始化缓存和限流相关功能
	initCacheAndLimiter()

	// 等待中断信号以优雅地关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("关闭服务器...")

	// 创建一个5秒超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 不接受新请求并等待现有请求完成
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("服务器强制关闭: %v", err)
	}

	// 关闭数据库和消息队列连接
	database.CloseDB()
	cache.CloseRedis()
	mqAdapter.Close()

	log.Println("服务器优雅关闭")
}

// updateVoteInDatabase 处理投票消息并更新数据库
func updateVoteInDatabase(pollID string, optionID string) error {
	// 增加选项投票计数
	err := database.IncrementVote(pollID, optionID)
	if err != nil {
		log.Printf("错误: 无法更新选项 %s 的投票: %v", optionID, err)
		return err
	}

	// 获取更新后的投票结果
	pollResults, err := database.GetPollResults(uint(getPollIDUint(pollID)))
	if err != nil {
		log.Printf("错误: 无法获取投票 %s 的结果: %v", pollID, err)
		return err
	}

	// 构建返回结果
	poll := map[string]interface{}{
		"poll_id": pollID,
		"results": pollResults,
	}

	// 使用WebSocket广播更新
	handlers.BroadcastPollUpdateStr(pollID, poll)

	fmt.Printf("投票已更新: 投票ID=%s, 选项ID=%s\n", pollID, optionID)
	return nil
}

// getPollIDUint 转换字符串ID为uint
func getPollIDUint(pollID string) uint {
	id, err := strconv.ParseUint(pollID, 10, 32)
	if err != nil {
		return 0
	}
	return uint(id)
}
