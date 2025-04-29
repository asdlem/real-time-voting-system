package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
)

// VoteMessage 表示投票消息结构
type VoteMessage struct {
	PollID    string `json:"poll_id"`
	OptionID  string `json:"option_id"`
	Timestamp int64  `json:"timestamp"`
	MessageID string `json:"message_id"` // 用于幂等性处理
}

// 全局RocketMQ生产者和消费者
var (
	rocketProducer rocketmq.Producer
	initOnce       sync.Once
	isInitialized  bool
	mockMode       bool                                       // 模拟模式标志
	mockMessages   = make([]VoteMessage, 0)                   // 模拟消息存储
	mockMutex      sync.Mutex                                 // 模拟操作的互斥锁
	processHandler func(pollID string, optionID string) error // 存储消息处理函数

	// 幂等性处理相关
	processedMessages      = make(map[string]bool) // 已处理消息的ID映射
	processedMessagesMutex sync.RWMutex            // 保护已处理消息映射的互斥锁
)

// 主题常量
const (
	TopicVoteEvents = "vote_events"
)

// InitRocketMQ 初始化RocketMQ生产者
func InitRocketMQ() error {
	var initErr error

	initOnce.Do(func() {
		// 检查是否强制使用模拟模式
		if os.Getenv("ROCKETMQ_MOCK") == "true" {
			log.Println("强制使用RocketMQ模拟模式")
			mockMode = true
			mockMessages = make([]VoteMessage, 0)
			isInitialized = true
			return
		}

		// 从环境变量获取RocketMQ地址
		nameServerAddr := os.Getenv("ROCKETMQ_NAMESRV_ADDR")
		if nameServerAddr == "" {
			nameServerAddr = "localhost:9876" // 默认地址，与docker-compose一致
		}

		log.Printf("初始化RocketMQ连接, 地址: %s", nameServerAddr)

		// 打印更多诊断信息
		log.Printf("RocketMQ连接环境：%v", os.Environ())

		// 准备多个可能的地址，提高连接成功率
		addrList := []string{
			nameServerAddr,             // 配置的地址
			"localhost:9876",           // 默认本地地址
			"127.0.0.1:9876",           // 本地回环地址
			"172.21.0.4:9876",          // Docker容器内部地址
			"rocketmq:9876",            // Docker容器服务名
			"89t3v2-rocketmq-dev:9876", // Docker容器名
		}

		// 尝试每个地址，直到成功或都失败
		var p rocketmq.Producer
		var err error
		var connected bool

		for _, addr := range addrList {
			log.Printf("尝试连接RocketMQ: %s", addr)

			// 尝试创建生产者
			p, err = rocketmq.NewProducer(
				producer.WithNameServer([]string{addr}),
				producer.WithGroupName("vote_producer"),
				producer.WithRetry(2),
				producer.WithSendMsgTimeout(time.Second*10), // 增加超时时间
				producer.WithVIPChannel(false),
			)

			if err != nil {
				log.Printf("使用地址 %s 创建RocketMQ生产者失败: %v", addr, err)
				continue
			}

			// 尝试启动生产者
			if err := p.Start(); err != nil {
				log.Printf("使用地址 %s 启动RocketMQ生产者失败: %v", addr, err)
				continue
			}

			// 启动成功
			connected = true
			log.Printf("使用地址 %s 连接RocketMQ成功", addr)
			break
		}

		if !connected {
			log.Printf("所有RocketMQ连接尝试均失败，将使用模拟模式")
			mockMode = true
			isInitialized = true
			return
		}

		rocketProducer = p
		isInitialized = true
		mockMode = false
		log.Println("RocketMQ生产者初始化成功")
	})

	return initErr
}

// IsMockMode 检查是否处于模拟模式
func IsMockMode() bool {
	return mockMode
}

// IsInitialized 检查RocketMQ是否已初始化
func IsInitialized() bool {
	return isInitialized && rocketProducer != nil
}

// 生成唯一的消息ID
func generateMessageID(pollID string, optionID string) string {
	return fmt.Sprintf("%s_%s_%d", pollID, optionID, time.Now().UnixNano())
}

// 检查消息是否已经处理过（幂等性检查）
func isMessageProcessed(messageID string) bool {
	processedMessagesMutex.RLock()
	defer processedMessagesMutex.RUnlock()
	return processedMessages[messageID]
}

// 标记消息为已处理
func markMessageAsProcessed(messageID string) {
	processedMessagesMutex.Lock()
	defer processedMessagesMutex.Unlock()
	processedMessages[messageID] = true

	// 设置过期时间，避免无限增长
	// 在实际生产环境中，应使用Redis等持久化存储
	go func(id string) {
		time.Sleep(24 * time.Hour) // 24小时后过期
		processedMessagesMutex.Lock()
		delete(processedMessages, id)
		processedMessagesMutex.Unlock()
	}(messageID)
}

// SendVoteMessage 发送投票消息到RocketMQ
func SendVoteMessage(pollID string, optionID string) error {
	if !isInitialized {
		return fmt.Errorf("RocketMQ生产者未初始化")
	}

	// 生成唯一消息ID
	messageID := generateMessageID(pollID, optionID)

	// 构造消息
	msg := VoteMessage{
		PollID:    pollID,
		OptionID:  optionID,
		Timestamp: time.Now().Unix(),
		MessageID: messageID,
	}

	// 模拟模式下的消息处理
	if mockMode {
		mockMutex.Lock()
		defer mockMutex.Unlock()

		// 保存消息到模拟存储
		mockMessages = append(mockMessages, msg)
		log.Printf("模拟模式: 发送消息成功, Poll: %s, Option: %s, MessageID: %s", pollID, optionID, messageID)

		// 如果有处理函数，直接处理消息
		if processHandler != nil {
			go func() {
				// 幂等性检查
				if isMessageProcessed(messageID) {
					log.Printf("模拟模式: 消息已处理过，跳过: %s", messageID)
					return
				}

				if err := processHandler(pollID, optionID); err != nil {
					log.Printf("模拟模式: 处理消息失败: %v", err)
				} else {
					log.Printf("模拟模式: 处理消息成功, Poll: %s, Option: %s", pollID, optionID)
					// 标记为已处理
					markMessageAsProcessed(messageID)
				}
			}()
		}

		return nil
	}

	// 真实RocketMQ模式
	// 序列化为JSON
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %v", err)
	}

	// 创建RocketMQ消息
	message := primitive.NewMessage(TopicVoteEvents, body)

	// 添加标签
	message.WithTag("vote")

	// 添加键 (用于消息去重和顺序性)
	message.WithKeys([]string{messageID})

	// 设置分区键（确保同一投票的消息进入同一队列）
	// 这样可以保证同一投票的操作顺序性
	message.WithShardingKey(pollID)

	// 发送消息（顺序消息）
	res, err := rocketProducer.SendSync(context.Background(), message)

	if err != nil {
		log.Printf("发送消息失败: %v", err)
		return fmt.Errorf("发送消息失败: %v", err)
	}

	log.Printf("发送消息成功, MsgID: %s, MessageID: %s, 队列: %s",
		res.MsgID, messageID, res.MessageQueue.String())
	return nil
}

// SendOrderedVoteMessage 发送顺序投票消息到RocketMQ
// 保证同一投票ID的所有操作按顺序处理
func SendOrderedVoteMessage(pollID string, optionID string) error {
	if !isInitialized {
		return fmt.Errorf("RocketMQ生产者未初始化")
	}

	// 生成唯一消息ID
	messageID := generateMessageID(pollID, optionID)

	// 构造消息
	msg := VoteMessage{
		PollID:    pollID,
		OptionID:  optionID,
		Timestamp: time.Now().Unix(),
		MessageID: messageID,
	}

	// 模拟模式下的消息处理
	if mockMode {
		mockMutex.Lock()

		// 保存消息到模拟存储（按顺序存储）
		mockMessages = append(mockMessages, msg)
		mockMutex.Unlock()

		log.Printf("模拟模式: 发送顺序消息成功, Poll: %s, Option: %s, MessageID: %s",
			pollID, optionID, messageID)

		// 如果有处理函数，直接处理消息
		if processHandler != nil {
			// 幂等性检查
			if isMessageProcessed(messageID) {
				log.Printf("模拟模式: 顺序消息已处理过，跳过: %s", messageID)
				return nil
			}

			if err := processHandler(pollID, optionID); err != nil {
				log.Printf("模拟模式: 处理顺序消息失败: %v", err)
			} else {
				log.Printf("模拟模式: 处理顺序消息成功, Poll: %s, Option: %s", pollID, optionID)
				// 标记为已处理
				markMessageAsProcessed(messageID)
			}
		}

		return nil
	}

	// 真实RocketMQ模式
	// 序列化为JSON
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("序列化顺序消息失败: %v", err)
	}

	// 创建RocketMQ消息
	message := primitive.NewMessage(TopicVoteEvents, body)
	message.WithTag("ordered_vote")
	message.WithKeys([]string{messageID})
	message.WithShardingKey(pollID) // 确保相同投票ID的消息路由到相同队列

	// 通过选择相同队列来保证顺序性
	// 在我们的场景中，使用普通同步发送，但在后台确保投票ID相同的消息进入同一队列
	res, err := rocketProducer.SendSync(context.Background(), message)

	if err != nil {
		log.Printf("发送顺序消息失败: %v", err)
		return fmt.Errorf("发送顺序消息失败: %v", err)
	}

	log.Printf("发送顺序消息成功, MsgID: %s, MessageID: %s, 队列: %s",
		res.MsgID, messageID, res.MessageQueue.String())
	return nil
}

// StartVoteConsumer 启动投票消息消费者
func StartVoteConsumer(processFunc func(pollID string, optionID string) error) error {
	// 保存处理函数
	processHandler = processFunc

	// 模拟模式下不需要创建真实消费者
	if mockMode {
		log.Println("模拟模式: 消息消费者启动")
		return nil
	}

	// 从环境变量获取RocketMQ地址
	nameServerAddr := os.Getenv("ROCKETMQ_NAMESRV_ADDR")
	if nameServerAddr == "" {
		nameServerAddr = "localhost:9876" // 默认地址
	}

	// 创建普通消费者
	c, err := rocketmq.NewPushConsumer(
		consumer.WithNameServer([]string{nameServerAddr}),
		consumer.WithGroupName("vote_consumer"),
		consumer.WithConsumerModel(consumer.Clustering),
		consumer.WithConsumeFromWhere(consumer.ConsumeFromLastOffset),
	)

	if err != nil {
		return fmt.Errorf("创建消息消费者失败: %v", err)
	}

	// 订阅主题
	err = c.Subscribe(TopicVoteEvents, consumer.MessageSelector{
		Type:       consumer.TAG,
		Expression: "vote",
	}, func(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
		for _, msg := range msgs {
			// 解析消息
			var voteMsg VoteMessage
			if err := json.Unmarshal(msg.Body, &voteMsg); err != nil {
				log.Printf("解析消息失败: %v", err)
				continue
			}

			// 幂等性检查 - 检查是否已处理过该消息
			if isMessageProcessed(voteMsg.MessageID) {
				log.Printf("消息已处理过，跳过: %s", voteMsg.MessageID)
				continue
			}

			log.Printf("收到投票消息: Poll=%s, Option=%s, MessageID=%s",
				voteMsg.PollID, voteMsg.OptionID, voteMsg.MessageID)

			// 处理消息
			if err := processFunc(voteMsg.PollID, voteMsg.OptionID); err != nil {
				log.Printf("处理消息失败: %v", err)
				return consumer.ConsumeRetryLater, nil // 稍后重试
			}

			// 标记消息为已处理，确保幂等性
			markMessageAsProcessed(voteMsg.MessageID)
		}
		return consumer.ConsumeSuccess, nil
	})

	if err != nil {
		return fmt.Errorf("订阅主题失败: %v", err)
	}

	// 启动消费者
	if err = c.Start(); err != nil {
		return fmt.Errorf("启动消费者失败: %v", err)
	}

	log.Println("消息消费者启动成功")
	return nil
}

// StartOrderedVoteConsumer 启动顺序投票消息消费者
func StartOrderedVoteConsumer(processFunc func(pollID string, optionID string) error) error {
	// 保存处理函数
	processHandler = processFunc

	// 模拟模式下不需要创建真实消费者
	if mockMode {
		log.Println("模拟模式: 顺序消息消费者启动")
		return nil
	}

	// 从环境变量获取RocketMQ地址
	nameServerAddr := os.Getenv("ROCKETMQ_NAMESRV_ADDR")
	if nameServerAddr == "" {
		nameServerAddr = "localhost:9876" // 默认地址
	}

	// 创建顺序消费者
	c, err := rocketmq.NewPushConsumer(
		consumer.WithNameServer([]string{nameServerAddr}),
		consumer.WithGroupName("ordered_vote_consumer"),
		consumer.WithConsumerModel(consumer.Clustering),
		consumer.WithConsumeFromWhere(consumer.ConsumeFromLastOffset),
		consumer.WithConsumerOrder(true), // 顺序消费
	)

	if err != nil {
		return fmt.Errorf("创建顺序消息消费者失败: %v", err)
	}

	// 订阅主题
	err = c.Subscribe(TopicVoteEvents, consumer.MessageSelector{
		Type:       consumer.TAG,
		Expression: "ordered_vote",
	}, func(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
		for _, msg := range msgs {
			// 解析消息
			var voteMsg VoteMessage
			if err := json.Unmarshal(msg.Body, &voteMsg); err != nil {
				log.Printf("解析顺序消息失败: %v", err)
				continue
			}

			// 幂等性检查 - 检查是否已处理过该消息
			if isMessageProcessed(voteMsg.MessageID) {
				log.Printf("顺序消息已处理过，跳过: %s", voteMsg.MessageID)
				continue
			}

			log.Printf("收到顺序投票消息: Poll=%s, Option=%s, MessageID=%s",
				voteMsg.PollID, voteMsg.OptionID, voteMsg.MessageID)

			// 处理消息
			if err := processFunc(voteMsg.PollID, voteMsg.OptionID); err != nil {
				log.Printf("处理顺序消息失败: %v", err)
				// 对于顺序消息，处理失败会阻塞同一队列的后续消息
				return consumer.ConsumeRetryLater, nil
			}

			// 标记消息为已处理，确保幂等性
			markMessageAsProcessed(voteMsg.MessageID)
		}
		return consumer.ConsumeSuccess, nil
	})

	if err != nil {
		return fmt.Errorf("订阅顺序主题失败: %v", err)
	}

	// 启动消费者
	if err = c.Start(); err != nil {
		return fmt.Errorf("启动顺序消费者失败: %v", err)
	}

	log.Println("顺序消息消费者启动成功")
	return nil
}

// CloseRocketMQ 关闭RocketMQ连接
func CloseRocketMQ() {
	if mockMode {
		log.Println("模拟模式: 关闭RocketMQ连接")
		return
	}

	if rocketProducer != nil {
		err := rocketProducer.Shutdown()
		if err != nil {
			log.Printf("关闭RocketMQ生产者失败: %v", err)
		} else {
			log.Println("RocketMQ生产者已关闭")
		}
	}
}

// GetQueuedMessageCount 获取队列中的消息数量
func GetQueuedMessageCount() int {
	if !mockMode {
		return -1 // 非模拟模式不支持此操作
	}

	mockMutex.Lock()
	defer mockMutex.Unlock()
	return len(mockMessages)
}
