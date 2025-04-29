package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisMQ是基于Redis实现的消息队列
type RedisMQ struct {
	client            *redis.Client
	ctx               context.Context
	processHandler    func(pollID string, optionID string) error
	isRunning         bool
	stopChan          chan struct{}
	wg                sync.WaitGroup
	processingTimeout time.Duration // 消息处理超时时间
	retryDelay        time.Duration // 重试延迟
	maxRetries        int           // 最大重试次数
}

// 消息队列的队列名称常量
const (
	MainQueueName       = "vote_queue"       // 主队列
	ProcessingQueueName = "vote_processing"  // 处理中队列
	DeadLetterQueueName = "vote_dead_letter" // 死信队列
	RetriesHashName     = "vote_retries"     // 重试次数记录
)

// 创建新的基于Redis的消息队列
func NewRedisMQ(redisClient *redis.Client) *RedisMQ {
	return &RedisMQ{
		client:            redisClient,
		ctx:               context.Background(),
		isRunning:         false,
		stopChan:          make(chan struct{}),
		processingTimeout: 5 * time.Minute,  // 默认5分钟超时
		retryDelay:        30 * time.Second, // 默认30秒重试延迟
		maxRetries:        3,                // 默认最大重试3次
	}
}

// 注册消息处理函数
func (r *RedisMQ) RegisterHandler(handler func(pollID string, optionID string) error) {
	r.processHandler = handler
}

// 发送投票消息
func (r *RedisMQ) SendVoteMessage(pollID string, optionID string) error {
	// 构造消息
	messageID := generateRedisMessageID(pollID, optionID)
	msg := VoteMessage{
		PollID:    pollID,
		OptionID:  optionID,
		Timestamp: time.Now().Unix(),
		MessageID: messageID,
	}

	// 序列化消息
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %v", err)
	}

	// 幂等性检查 - 检查该消息是否已在处理队列中
	exists, err := r.client.SIsMember(r.ctx, "vote_message_ids", messageID).Result()
	if err != nil {
		// 继续处理，不因此阻止业务 - 但记录错误
		log.Printf("检查消息幂等性出错: %v", err)
	} else if exists {
		// 消息已存在，已处理过
		log.Printf("消息已处理过，跳过: %s", messageID)
		return nil
	}

	// 添加消息ID到集合，用于幂等性检查
	err = r.client.SAdd(r.ctx, "vote_message_ids", messageID).Err()
	if err != nil {
		// 继续处理，但记录错误
		log.Printf("添加消息ID到幂等性集合出错: %v", err)
	}
	// 设置过期时间，避免集合无限增长
	r.client.Expire(r.ctx, "vote_message_ids", 48*time.Hour)

	// 发送消息到主队列
	err = r.client.LPush(r.ctx, MainQueueName, jsonData).Err()
	if err != nil {
		return fmt.Errorf("发送消息到队列失败: %v", err)
	}

	log.Printf("消息成功发送到Redis队列: %s, 消息ID: %s", MainQueueName, messageID)
	return nil
}

// 启动消费者
func (r *RedisMQ) Start() error {
	if r.processHandler == nil {
		return fmt.Errorf("处理函数未注册")
	}

	if r.isRunning {
		return nil // 已经在运行中
	}

	r.isRunning = true
	log.Println("Redis消息队列消费者启动中...")

	// 启动主消费循环
	r.wg.Add(1)
	go r.consumeLoop()

	// 启动处理中消息的超时检查
	r.wg.Add(1)
	go r.timeoutCheckLoop()

	log.Println("Redis消息队列消费者已启动")
	return nil
}

// 关闭消费者
func (r *RedisMQ) Stop() {
	if !r.isRunning {
		return
	}

	log.Println("正在关闭Redis消息队列消费者...")
	close(r.stopChan)
	r.wg.Wait()
	r.isRunning = false
	log.Println("Redis消息队列消费者已关闭")
}

// 主消费循环
func (r *RedisMQ) consumeLoop() {
	defer r.wg.Done()

	for {
		select {
		case <-r.stopChan:
			return
		default:
			// 使用BRPOPLPUSH原子操作从主队列获取并移动到处理中队列
			result, err := r.client.BRPopLPush(r.ctx, MainQueueName, ProcessingQueueName, 1*time.Second).Result()

			if err != nil {
				if err != redis.Nil { // 忽略超时错误
					log.Printf("从队列获取消息失败: %v", err)
				}
				continue
			}

			// 异步处理消息
			go r.processMessage(result)
		}
	}
}

// 超时检查循环
func (r *RedisMQ) timeoutCheckLoop() {
	defer r.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-r.stopChan:
			return
		case <-ticker.C:
			r.checkTimeouts()
		}
	}
}

// 检查消息处理超时
func (r *RedisMQ) checkTimeouts() {
	// 获取处理中队列的所有消息
	messages, err := r.client.LRange(r.ctx, ProcessingQueueName, 0, -1).Result()
	if err != nil {
		log.Printf("获取处理中队列消息失败: %v", err)
		return
	}

	now := time.Now().Unix()

	for _, msgData := range messages {
		var msg VoteMessage
		if err := json.Unmarshal([]byte(msgData), &msg); err != nil {
			log.Printf("解析消息数据失败: %v", err)
			continue
		}

		// 如果消息处理时间超过timeout，重新入队
		if now-msg.Timestamp > int64(r.processingTimeout.Seconds()) {
			retries, _ := r.client.HGet(r.ctx, RetriesHashName, msg.MessageID).Int()

			if retries >= r.maxRetries {
				// 超过最大重试次数，移至死信队列
				log.Printf("消息 %s 超过最大重试次数，移至死信队列", msg.MessageID)
				r.moveToDeadLetter(msgData)
			} else {
				// 增加重试计数
				r.client.HIncrBy(r.ctx, RetriesHashName, msg.MessageID, 1)

				// 更新时间戳
				msg.Timestamp = now
				updatedData, _ := json.Marshal(msg)

				// 从处理中队列删除
				r.client.LRem(r.ctx, ProcessingQueueName, 1, msgData)

				// 延迟一段时间后重新入队
				time.AfterFunc(r.retryDelay, func() {
					r.client.LPush(r.ctx, MainQueueName, updatedData)
					log.Printf("消息 %s 重新入队，重试次数: %d", msg.MessageID, retries+1)
				})
			}
		}
	}
}

// 处理单个消息
func (r *RedisMQ) processMessage(msgData string) {
	var msg VoteMessage
	if err := json.Unmarshal([]byte(msgData), &msg); err != nil {
		log.Printf("解析消息失败: %v", err)
		r.moveToDeadLetter(msgData)
		return
	}

	log.Printf("处理消息: PollID=%s, OptionID=%s, MessageID=%s",
		msg.PollID, msg.OptionID, msg.MessageID)

	// 调用处理函数
	if err := r.processHandler(msg.PollID, msg.OptionID); err != nil {
		log.Printf("处理消息失败: %v", err)

		// 获取当前重试次数
		retries, _ := r.client.HGet(r.ctx, RetriesHashName, msg.MessageID).Int()

		if retries >= r.maxRetries {
			// 超过最大重试次数，移至死信队列
			log.Printf("消息 %s 超过最大重试次数，移至死信队列", msg.MessageID)
			r.moveToDeadLetter(msgData)
		} else {
			// 增加重试计数
			r.client.HIncrBy(r.ctx, RetriesHashName, msg.MessageID, 1)

			// 更新时间戳
			msg.Timestamp = time.Now().Unix()
			updatedData, _ := json.Marshal(msg)

			// 延迟重试
			time.AfterFunc(r.retryDelay, func() {
				r.client.LPush(r.ctx, MainQueueName, updatedData)
				log.Printf("消息 %s 重新入队，重试次数: %d", msg.MessageID, retries+1)
			})
		}
	} else {
		// 处理成功，从处理中队列移除
		log.Printf("消息处理成功: %s", msg.MessageID)
	}

	// 无论成功失败，都从处理中队列移除
	r.client.LRem(r.ctx, ProcessingQueueName, 1, msgData)
}

// 将消息移动到死信队列
func (r *RedisMQ) moveToDeadLetter(msgData string) {
	r.client.LPush(r.ctx, DeadLetterQueueName, msgData)
	r.client.LRem(r.ctx, ProcessingQueueName, 1, msgData)
}

// 生成唯一的Redis消息ID
func generateRedisMessageID(pollID string, optionID string) string {
	return fmt.Sprintf("redis_mq_%s_%s_%d", pollID, optionID, time.Now().UnixNano())
}

// 重新处理死信队列中的消息
func (r *RedisMQ) RetryDeadLetters() error {
	// 获取死信队列中的所有消息
	messages, err := r.client.LRange(r.ctx, DeadLetterQueueName, 0, -1).Result()
	if err != nil {
		return fmt.Errorf("获取死信队列消息失败: %v", err)
	}

	count := 0
	for _, msgData := range messages {
		// 重新入队到主队列
		err := r.client.LPush(r.ctx, MainQueueName, msgData).Err()
		if err != nil {
			log.Printf("重新入队消息失败: %v", err)
			continue
		}

		// 从死信队列移除
		r.client.LRem(r.ctx, DeadLetterQueueName, 1, msgData)

		// 重置重试计数
		var msg VoteMessage
		if json.Unmarshal([]byte(msgData), &msg) == nil {
			r.client.HDel(r.ctx, RetriesHashName, msg.MessageID)
		}

		count++
	}

	log.Printf("成功将 %d 条消息从死信队列移回主队列", count)
	return nil
}

// 获取各队列的消息数量统计
func (r *RedisMQ) GetQueueStats() map[string]int64 {
	stats := make(map[string]int64)

	mainLen, _ := r.client.LLen(r.ctx, MainQueueName).Result()
	procLen, _ := r.client.LLen(r.ctx, ProcessingQueueName).Result()
	deadLen, _ := r.client.LLen(r.ctx, DeadLetterQueueName).Result()

	stats["main_queue"] = mainLen
	stats["processing_queue"] = procLen
	stats["dead_letter_queue"] = deadLen

	return stats
}

// 清空所有队列（仅用于测试）
func (r *RedisMQ) ClearAllQueues() error {
	err := r.client.Del(r.ctx, MainQueueName, ProcessingQueueName, DeadLetterQueueName, RetriesHashName).Err()
	if err != nil {
		return fmt.Errorf("清空队列失败: %v", err)
	}
	return nil
}
