package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// MQAdapter 消息队列适配器，支持在RocketMQ和Redis MQ之间自动切换
type MQAdapter struct {
	rocketEnabled  bool
	redisEnabled   bool
	redisMQ        *RedisMQ
	redisClient    *redis.Client
	processHandler func(pollID string, optionID string) error
	initOnce       sync.Once
	initialized    bool
}

// NewMQAdapter 创建新的消息队列适配器
func NewMQAdapter() *MQAdapter {
	return &MQAdapter{
		rocketEnabled: false,
		redisEnabled:  false,
		initialized:   false,
	}
}

// Initialize 初始化消息队列
func (a *MQAdapter) Initialize() error {
	var err error
	a.initOnce.Do(func() {
		// // 首先尝试初始化RocketMQ // <-- 注释掉这部分
		// rocketErr := InitRocketMQ()
		// if rocketErr == nil && !mockMode {
		// 	a.rocketEnabled = true
		// 	log.Println("成功初始化RocketMQ")
		// } else { // <-- 移除 else
		// if rocketErr != nil { // <-- 注释掉
		// 	log.Printf("RocketMQ初始化失败: %v", rocketErr) // <-- 注释掉
		// } else { // <-- 注释掉
		// 	log.Println("RocketMQ使用模拟模式，将尝试Redis MQ") // <-- 注释掉
		// } // <-- 注释掉

		// 直接尝试使用Redis MQ
		log.Println("正在尝试初始化 Redis MQ...") // 添加日志

		// 获取Redis客户端
		redisHost := os.Getenv("REDIS_HOST")
		if redisHost == "" {
			redisHost = "localhost" // Default host
		}
		redisPort := os.Getenv("REDIS_PORT")
		if redisPort == "" {
			redisPort = "6379" // Default port
		}
		// 优先使用 REDIS_ADDR 如果设置了，否则组合 HOST 和 PORT
		redisAddr := os.Getenv("REDIS_ADDR")
		if redisAddr == "" {
			// 默认使用16379端口，而不是6379
			redisPort = "16379"
			redisAddr = fmt.Sprintf("%s:%s", redisHost, redisPort) // 组合 host 和 port
		}

		log.Printf("使用 Redis 地址: %s", redisAddr) // 确认地址是正确的 host:port 格式

		redisPassword := os.Getenv("REDIS_PASSWORD")

		// 创建Redis客户端
		a.redisClient = redis.NewClient(&redis.Options{
			Addr:        redisAddr, // 使用修正后的 redisAddr
			Password:    redisPassword,
			DB:          0,
			DialTimeout: 5 * time.Second, // 增加超时时间
			ReadTimeout: 5 * time.Second, // 增加超时时间
			PoolSize:    20,              // 增加连接池大小
		})

		// 测试连接
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // 增加上下文超时
		defer cancel()

		if _, redisErr := a.redisClient.Ping(ctx).Result(); redisErr != nil {
			log.Printf("Redis连接失败: %v", redisErr)
			// 如果Redis也失败，设置为错误状态，避免进入内存模式
			a.initialized = false // 明确标记为未初始化
			err = fmt.Errorf("无法初始化Redis MQ: %v", redisErr)
		} else {
			// Redis连接成功，创建RedisMQ实例
			a.redisMQ = NewRedisMQ(a.redisClient)
			a.redisEnabled = true
			a.initialized = true // 标记为已初始化 (Redis 模式)
			log.Println("成功初始化Redis MQ")
		}
		// } // <-- 移除与 RocketMQ else 对应的括号

		// a.initialized = true // 移动到 Redis 初始化成功的分支
	})

	return err
}

// RegisterHandler 注册消息处理函数
func (a *MQAdapter) RegisterHandler(handler func(pollID string, optionID string) error) error {
	if !a.initialized {
		return fmt.Errorf("消息队列适配器未初始化")
	}

	a.processHandler = handler

	// 根据启用的队列类型，注册处理函数
	if a.rocketEnabled { // 这个分支现在不应该被执行
		log.Println("警告: RocketMQ 被禁用，但尝试注册其 Handler")
		return fmt.Errorf("RocketMQ 已被禁用")
	} else if a.redisEnabled {
		// 用Redis MQ处理函数
		if a.redisMQ == nil {
			return fmt.Errorf("错误: Redis MQ 实例为空，无法注册 Handler")
		}
		a.redisMQ.RegisterHandler(handler)
		err := a.redisMQ.Start()
		if err != nil {
			return fmt.Errorf("启动Redis MQ消费者失败: %v", err)
		}
		log.Println("已注册并启动 Redis MQ 消费者")
	} else {
		log.Println("错误: 适配器未初始化成功，无法注册 Handler")
		return fmt.Errorf("适配器未初始化成功")
	}

	return nil
}

// SendVoteMessage 发送投票消息
func (a *MQAdapter) SendVoteMessage(pollID string, optionID string) error {
	// 修复：简化初始化检查
	if !a.IsInitialized() {
		return fmt.Errorf("消息队列适配器未初始化，无法发送消息")
	}

	if a.redisEnabled {
		if a.redisMQ == nil {
			return fmt.Errorf("错误: Redis MQ 实例为空，无法发送消息")
		}
		// 委托给 RedisMQ 实现
		return a.redisMQ.SendVoteMessage(pollID, optionID)
	} else {
		// 现在不应该进入此分支
		return fmt.Errorf("消息队列未初始化为 Redis 模式")
	}
}

// SendOrderedVoteMessage 发送顺序投票消息
func (a *MQAdapter) SendOrderedVoteMessage(pollID string, optionID string) error {
	// 对于 Redis List，普通发送即保证顺序
	return a.SendVoteMessage(pollID, optionID)
}

// Close 关闭消息队列
func (a *MQAdapter) Close() {
	if a.redisEnabled && a.redisMQ != nil {
		// 关闭Redis MQ
		a.redisMQ.Stop()
		a.redisClient.Close()
	}
	log.Println("消息队列已关闭")
}

// GetQueueStats 获取队列统计信息
func (a *MQAdapter) GetQueueStats() map[string]interface{} {
	stats := make(map[string]interface{})

	if !a.IsInitialized() { // 使用 IsInitialized 判断
		stats["status"] = "未初始化"
		return stats
	}

	if a.redisEnabled {
		stats["type"] = "Redis MQ"
		if a.redisMQ != nil {
			stats["status"] = "正常运行"
			stats["queues"] = a.redisMQ.GetQueueStats()
		} else {
			stats["status"] = "实例为空"
		}
	} else {
		stats["type"] = "未知 (错误状态)"
		stats["status"] = "错误"
	}

	return stats
}

// RetryDeadLetters 重试死信队列中的消息（仅Redis MQ模式可用）
func (a *MQAdapter) RetryDeadLetters() error {
	if !a.IsInitialized() {
		return fmt.Errorf("消息队列适配器未初始化")
	}

	if a.redisEnabled {
		if a.redisMQ == nil {
			return fmt.Errorf("Redis MQ 实例为空")
		}
		return a.redisMQ.RetryDeadLetters()
	}

	return fmt.Errorf("当前消息队列模式不支持死信队列操作")
}

// IsInitialized 检查适配器是否已初始化
func (a *MQAdapter) IsInitialized() bool {
	// 现在只关心 Redis 是否启用并初始化
	return a.initialized && a.redisEnabled
}

// SendVoteMessageWithID 发送投票消息，使用指定的messageID
func (a *MQAdapter) SendVoteMessageWithID(pollID string, optionID string, messageID string) error {
	if !a.IsInitialized() {
		return fmt.Errorf("消息队列适配器未初始化")
	}

	if a.redisEnabled {
		if a.redisMQ == nil {
			return fmt.Errorf("错误: Redis MQ 实例为空，无法发送消息")
		}

		// 构造消息
		msg := VoteMessage{
			PollID:    pollID,
			OptionID:  optionID,
			Timestamp: time.Now().Unix(),
			MessageID: messageID, // 使用传入的 messageID
		}
		jsonData, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("序列化消息失败: %v", err)
		}

		// 幂等性检查
		processedSetKey := "processed_vote_ids"
		exists, err := a.redisClient.SIsMember(context.Background(), processedSetKey, messageID).Result()
		if err != nil {
			log.Printf("检查消息幂等性出错: %v", err) // 记录错误但继续
		} else if exists {
			log.Printf("检测到重复消息ID %s，跳过发送", messageID)
			return nil // 幂等成功
		}

		// 发送消息到Redis List - 修复：使用 MainQueueName 常量
		// 注意：MainQueueName 需要在包内可访问，或者直接用 "vote_queue"
		err = a.redisClient.LPush(context.Background(), MainQueueName, string(jsonData)).Err()
		if err != nil {
			return fmt.Errorf("发送消息到 Redis 失败: %v", err)
		}

		// 添加到已处理集合
		err = a.redisClient.SAdd(context.Background(), processedSetKey, messageID).Err()
		if err != nil {
			log.Printf("添加消息ID %s 到幂等性集合失败: %v", messageID, err) // 记录错误
		}

		return nil
	} else {
		return fmt.Errorf("消息队列未初始化为 Redis 模式")
	}
}
