package cache

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// 全局Redis客户端
var (
	redisClient *redis.Client
	redisCtx    = context.Background()
	initOnce    sync.Once
	initialized bool

	// 模拟相关
	PollIDBloomFilter     bool // 简化的布隆过滤器
	VoteMessageIdempotent bool // 简化的幂等控制
	GlobalDCL             bool // 简化的双重检查锁

	// 缓存默认过期时间
	defaultExpiration = 1 * time.Hour
	// 空值缓存过期时间（用于缓存穿透）
	nullExpiration = 5 * time.Minute
	// 缓存时间抖动系数
	jitterFactor = 0.2
	// 布隆过滤器误判率
	bloomFilterErrorRate = 0.01
	// 锁超时时间
	lockTimeout = 5 * time.Second
)

// InitRedis 初始化Redis连接
func InitRedis() error {
	var initErr error

	initOnce.Do(func() {
		// 检查是否强制使用模拟模式
		if os.Getenv("REDIS_MOCK") == "true" {
			log.Println("强制使用Redis模拟模式")
			mockMode = true
			initialized = true
			return
		}

		// 从环境变量获取Redis连接信息
		redisAddr := os.Getenv("REDIS_ADDR")
		redisPassword := os.Getenv("REDIS_PASSWORD")
		redisDb := 0

		// 尝试从环境变量解析Redis数据库编号
		if dbStr := os.Getenv("REDIS_DB"); dbStr != "" {
			if db, err := strconv.Atoi(dbStr); err == nil {
				redisDb = db
			}
		}

		if redisAddr == "" {
			redisAddr = "localhost:16379"
		}

		log.Printf("初始化Redis连接, 地址: %s", redisAddr)

		// 创建Redis客户端
		options := &redis.Options{
			Addr:        redisAddr,
			Password:    redisPassword,
			DB:          redisDb,
			DialTimeout: 3 * time.Second, // 适当增加超时时间
			ReadTimeout: 3 * time.Second,
			PoolSize:    10,
		}

		client := redis.NewClient(options)

		// 测试连接
		if _, err := client.Ping(redisCtx).Result(); err != nil {
			log.Printf("Redis连接失败: %v，将使用模拟模式", err)
			mockMode = true
			initialized = true
			return
		}

		redisClient = client
		initialized = true
		mockMode = false
		log.Println("Redis连接初始化成功")
	})

	return initErr
}

// GetClient 获取Redis客户端实例
func GetClient() (*redis.Client, error) {
	if !initialized {
		return nil, fmt.Errorf("Redis客户端未初始化")
	}
	if mockMode {
		return nil, fmt.Errorf("处于模拟模式，无法获取真实客户端")
	}
	return redisClient, nil
}

// IncrementVoteCount 原子增加投票计数
func IncrementVoteCount(pollID string, optionID string, increment int64) (int64, error) {
	if !initialized {
		return 0, fmt.Errorf("Redis客户端未初始化")
	}

	// 使用标准的票数存储格式
	key := fmt.Sprintf("poll:%s:votes:%s", pollID, optionID)

	if mockMode {
		// 模拟模式下的计数器实现
		mockMutex.Lock()
		defer mockMutex.Unlock()

		currentVal := int64(0)
		if val, ok := mockData[key]; ok {
			if parsed, err := strconv.ParseInt(val, 10, 64); err == nil {
				currentVal = parsed
			}
		}

		currentVal += increment
		mockData[key] = strconv.FormatInt(currentVal, 10)

		return currentVal, nil
	}

	// 真实Redis模式
	ctx := context.Background()
	currentCount, err := redisClient.IncrBy(ctx, key, increment).Result()
	if err != nil {
		return 0, fmt.Errorf("增加投票计数失败: %v", err)
	}

	// 设置键的过期时间（24小时）
	redisClient.Expire(ctx, key, 24*time.Hour)

	return currentCount, nil
}

// GetVoteCounts 获取投票所有选项的计数
func GetVoteCounts(pollID string) (map[string]int64, error) {
	client, err := GetClient()
	if err != nil {
		return nil, err
	}

	// 使用正确的格式构建选项计数的键
	pattern := fmt.Sprintf("poll:%s:votes:*", pollID)

	// 创建context
	ctx := context.Background()

	// 获取所有匹配的键
	keys, err := client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}

	if len(keys) == 0 {
		log.Printf("未找到投票 %s 的选项计数键", pollID)
		return map[string]int64{}, nil
	}

	// 使用管道批量获取所有键的值
	pipe := client.Pipeline()
	cmds := make(map[string]*redis.StringCmd)
	for _, key := range keys {
		cmds[key] = pipe.Get(ctx, key)
	}

	// 执行管道命令
	_, err = pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, err
	}

	// 处理结果并构建返回值
	results := make(map[string]int64)
	for key, cmd := range cmds {
		// 从键名中提取选项ID
		parts := strings.Split(key, ":")
		if len(parts) < 4 {
			continue
		}
		optionID := parts[3]

		// 获取计数值并转换为整数
		stringVal, err := cmd.Result()
		if err == nil {
			count, err := strconv.ParseInt(stringVal, 10, 64)
			if err == nil {
				results[optionID] = count
			}
		}
	}

	log.Printf("获取到投票 %s 的计数: %v", pollID, results)
	return results, nil
}

// AcquireLock 获取分布式锁
func AcquireLock(lockKey string, expiration time.Duration) (bool, error) {
	if !initialized {
		return false, fmt.Errorf("Redis客户端未初始化")
	}

	key := "poll:lock:" + lockKey

	if mockMode {
		// 模拟模式下的锁实现
		mockMutex.Lock()
		defer mockMutex.Unlock()

		// 检查锁是否存在
		if locked, exists := mockLocks[key]; exists && locked {
			return false, nil // 锁已被占用
		}

		// 设置锁
		mockLocks[key] = true
		return true, nil
	}

	// 真实Redis模式
	success, err := redisClient.SetNX(redisCtx, key, "1", expiration).Result()
	if err != nil {
		return false, fmt.Errorf("获取锁失败: %v", err)
	}

	return success, nil
}

// ReleaseLock 释放分布式锁
func ReleaseLock(lockKey string) error {
	if !initialized {
		return fmt.Errorf("Redis客户端未初始化")
	}

	key := "poll:lock:" + lockKey

	if mockMode {
		// 模拟模式下的释放锁实现
		mockMutex.Lock()
		defer mockMutex.Unlock()

		delete(mockLocks, key)
		return nil
	}

	// 真实Redis模式
	_, err := redisClient.Del(redisCtx, key).Result()
	if err != nil {
		return fmt.Errorf("释放锁失败: %v", err)
	}

	return nil
}

// CloseRedis 关闭Redis连接
func CloseRedis() {
	if initialized && !mockMode && redisClient != nil {
		err := redisClient.Close()
		if err != nil {
			log.Printf("关闭Redis连接错误: %v", err)
		}
		log.Println("Redis连接已关闭")
	}
}

// ----------------- 缓存三大问题解决方案 -----------------

// CheckBloomFilter 检查布隆过滤器，防止缓存穿透
// 使用布隆过滤器快速判断一个key是否可能存在，降低对数据库的无效查询
func CheckBloomFilter(key string) bool {
	if !initialized {
		log.Println("Redis未初始化，无法检查布隆过滤器")
		return true // 保守策略，未初始化时假设key可能存在
	}

	if mockMode {
		// 模拟模式下总是返回存在
		return PollIDBloomFilter
	}

	// 实际布隆过滤器检查逻辑
	bloomKey := "bloom:filter:polls" // 布隆过滤器的key
	exists, err := redisClient.BFExists(redisCtx, bloomKey, key).Result()
	if err != nil {
		log.Printf("检查布隆过滤器失败: %v", err)
		return true // 出错时保守处理，认为可能存在
	}

	return exists
}

// AddToBloomFilter 添加值到布隆过滤器
func AddToBloomFilter(key string) error {
	if !initialized {
		return fmt.Errorf("Redis未初始化，无法添加到布隆过滤器")
	}

	if mockMode {
		// 模拟模式下直接设置标志
		PollIDBloomFilter = true
		return nil
	}

	// 实际添加到布隆过滤器的逻辑
	bloomKey := "bloom:filter:polls" // 布隆过滤器的key
	_, err := redisClient.BFAdd(redisCtx, bloomKey, key).Result()
	if err != nil {
		return fmt.Errorf("添加到布隆过滤器失败: %v", err)
	}
	return nil
}

// GetWithCache 带缓存的获取数据，处理缓存穿透和缓存击穿
// queryFunc是在缓存未命中时调用的函数，用于从数据库查询数据
func GetWithCache(key string, expiration time.Duration, queryFunc func() (interface{}, error)) (interface{}, error) {
	if !initialized {
		return nil, fmt.Errorf("Redis未初始化")
	}

	// 先查缓存
	if mockMode {
		// 模拟模式下的缓存查询
		mockMutex.Lock()
		if data, exists := mockData[key]; exists {
			mockMutex.Unlock()
			// 检查是否为空值（缓存穿透保护）
			if data == "nil" {
				return nil, nil
			}
			return data, nil
		}
		mockMutex.Unlock()
	} else {
		// 真实Redis模式
		data, err := redisClient.Get(redisCtx, key).Result()
		if err == nil {
			// 检查是否为空值（缓存穿透保护）
			if data == "nil" {
				return nil, nil
			}
			return data, nil
		} else if err != redis.Nil {
			return nil, fmt.Errorf("查询缓存失败: %v", err)
		}
	}

	// 缓存未命中，检查布隆过滤器（缓存穿透保护）
	if !CheckBloomFilter(key) {
		return nil, nil // 布隆过滤器显示不存在，直接返回空
	}

	// 尝试获取分布式锁来防止缓存击穿
	lockKey := "lock:" + key
	acquired, err := AcquireLock(lockKey, lockTimeout)
	if err != nil {
		log.Printf("获取缓存锁失败: %v", err)
		// 即使获取锁失败也继续执行，但可能会有短时间的缓存击穿
	}

	// 如果获取锁成功，我们负责重建缓存
	// 如果没获取到锁，再次检查缓存，可能已被其他进程重建
	if !acquired {
		// 再次检查缓存
		if mockMode {
			mockMutex.Lock()
			if data, exists := mockData[key]; exists {
				mockMutex.Unlock()
				if data == "nil" {
					return nil, nil
				}
				return data, nil
			}
			mockMutex.Unlock()
		} else {
			data, err := redisClient.Get(redisCtx, key).Result()
			if err == nil {
				if data == "nil" {
					return nil, nil
				}
				return data, nil
			} else if err != redis.Nil {
				return nil, fmt.Errorf("二次查询缓存失败: %v", err)
			}
		}
	}

	// 从数据库获取数据
	data, err := queryFunc()

	// 无论成功失败，尝试释放锁
	if acquired {
		if releaseErr := ReleaseLock(lockKey); releaseErr != nil {
			log.Printf("释放缓存锁失败: %v", releaseErr)
		}
	}

	// 处理数据库查询结果
	if err != nil {
		return nil, fmt.Errorf("从数据库获取数据失败: %v", err)
	}

	// 如果结果为空，缓存一个空值以防止缓存穿透
	var cacheData string
	if data == nil {
		cacheData = "nil"
		expiration = nullExpiration // 空值使用较短的过期时间
	} else {
		// 转换数据为字符串
		switch v := data.(type) {
		case string:
			cacheData = v
		case []byte:
			cacheData = string(v)
		default:
			// 尝试将其他类型转为字符串
			cacheData = fmt.Sprintf("%v", data)
		}
	}

	// 添加随机抖动，防止缓存雪崩
	jitter := time.Duration(float64(expiration) * (1 + jitterFactor*(0.5-rand.Float64())))

	// 将数据放入缓存
	if mockMode {
		mockMutex.Lock()
		mockData[key] = cacheData
		mockMutex.Unlock()
	} else {
		if err := redisClient.Set(redisCtx, key, cacheData, jitter).Err(); err != nil {
			log.Printf("设置缓存失败: %v", err)
		}
	}

	if data == nil {
		return nil, nil
	}
	return data, nil
}

// PrewarmCache 预热缓存，防止缓存雪崩
// keys是需要预热的键列表，queryFunc是用于从数据库查询数据的函数
func PrewarmCache(keys []string, queryFunc func(key string) (interface{}, error)) error {
	if !initialized {
		return fmt.Errorf("Redis未初始化")
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(keys))

	for _, key := range keys {
		wg.Add(1)

		go func(k string) {
			defer wg.Done()

			// 使用不同的过期时间，防止同时失效
			expiration := defaultExpiration + time.Duration(rand.Int63n(int64(30*time.Minute)))

			_, err := GetWithCache(k, expiration, func() (interface{}, error) {
				return queryFunc(k)
			})

			if err != nil {
				errChan <- fmt.Errorf("预热缓存 %s 失败: %v", k, err)
			}
		}(key)
	}

	wg.Wait()
	close(errChan)

	// 收集所有错误
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("预热缓存过程中发生 %d 个错误", len(errors))
	}

	return nil
}
