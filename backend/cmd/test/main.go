package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"realtime-voting-backend/cache"
	"sync"
	"time"
)

// 测试布隆过滤器
func testBloomFilter() {
	fmt.Println("=== 测试布隆过滤器 ===")

	// 初始化Redis
	err := cache.InitRedis()
	if err != nil {
		log.Fatalf("初始化Redis失败: %v", err)
	}

	// 初始化布隆过滤器
	filter := cache.InitBloomFilter()
	if filter == nil {
		log.Fatalf("初始化布隆过滤器失败")
	}

	// 测试添加和检查
	ctx := context.Background()

	// 添加测试数据
	pollIDs := []string{"1", "2", "3", "test-poll"}
	for _, id := range pollIDs {
		err := filter.Add(ctx, "poll:"+id)
		if err != nil {
			log.Printf("添加到布隆过滤器失败 - poll:%s: %v", id, err)
		} else {
			log.Printf("添加到布隆过滤器成功 - poll:%s", id)
		}
	}

	// 检查存在的数据
	for _, id := range pollIDs {
		exists, err := filter.Contains(ctx, "poll:"+id)
		if err != nil {
			log.Printf("检查布隆过滤器失败 - poll:%s: %v", id, err)
		} else if exists {
			log.Printf("布隆过滤器检查成功 - poll:%s 存在", id)
		} else {
			log.Printf("布隆过滤器异常 - poll:%s 应该存在但未检测到", id)
		}
	}

	// 检查不存在的数据
	nonExistIDs := []string{"999", "non-exist"}
	for _, id := range nonExistIDs {
		exists, err := filter.Contains(ctx, "poll:"+id)
		if err != nil {
			log.Printf("检查布隆过滤器失败 - poll:%s: %v", id, err)
		} else if !exists {
			log.Printf("布隆过滤器检查成功 - poll:%s 不存在", id)
		} else {
			log.Printf("布隆过滤器误判 - poll:%s 不应该存在但检测到存在", id)
		}
	}
}

// 测试限流器
func testRateLimiter() {
	fmt.Println("\n=== 测试限流器 ===")

	// 获取Redis客户端
	redisClient, err := cache.GetRedisClient()
	if err != nil {
		log.Fatalf("获取Redis客户端失败: %v", err)
	}

	// 创建令牌桶限流器：每秒3个请求，最大突发5个
	limiter := cache.NewTokenBucketRateLimiter(redisClient, "test_limiter", 3, 5)

	// 发送10个请求，应该只有前5个允许通过
	ctx := context.Background()

	// 快速连续请求
	log.Println("快速连续发送10个请求：")
	allowedCount := 0
	for i := 0; i < 10; i++ {
		allowed, err := limiter.Allow(ctx)
		if err != nil {
			log.Printf("请求 %d: 限流检查错误: %v", i+1, err)
		} else if allowed {
			log.Printf("请求 %d: 允许通过", i+1)
			allowedCount++
		} else {
			log.Printf("请求 %d: 被限流", i+1)
		}
	}
	log.Printf("总共允许 %d 个请求通过", allowedCount)

	// 等待3秒，应该又可以通过3个请求
	time.Sleep(3 * time.Second)
	log.Println("等待3秒后再发送5个请求：")
	allowedCount = 0
	for i := 0; i < 5; i++ {
		allowed, err := limiter.Allow(ctx)
		if err != nil {
			log.Printf("请求 %d: 限流检查错误: %v", i+1, err)
		} else if allowed {
			log.Printf("请求 %d: 允许通过", i+1)
			allowedCount++
		} else {
			log.Printf("请求 %d: 被限流", i+1)
		}
	}
	log.Printf("总共允许 %d 个请求通过", allowedCount)
}

// 测试热点缓存
func testHotCache() {
	fmt.Println("\n=== 测试热点缓存 ===")

	// 获取Redis客户端
	redisClient, err := cache.GetRedisClient()
	if err != nil {
		log.Fatalf("获取Redis客户端失败: %v", err)
	}

	// 获取布隆过滤器和分布式锁
	bloomFilter := cache.InitBloomFilter()
	lockService := cache.GetLockService()

	// 创建热点缓存管理器
	hotCache := cache.NewHotCache(redisClient, lockService, bloomFilter)

	// 测试缓存获取和缓存击穿保护
	ctx := context.Background()
	cacheKey := "test:hotcache:key"

	// 模拟慢速数据加载函数
	dataLoader := func() (interface{}, error) {
		log.Println("从数据源加载数据...")
		time.Sleep(2 * time.Second) // 模拟访问慢速数据源
		return map[string]interface{}{
			"id":   "1",
			"name": "测试数据",
			"time": time.Now().String(),
		}, nil
	}

	// 并发请求，测试缓存击穿保护
	var wg sync.WaitGroup
	concurrentRequests := 10

	log.Printf("发起 %d 个并发请求", concurrentRequests)
	dataSourceCallCount := 0
	var mu sync.Mutex

	// 记录开始时间
	startTime := time.Now()

	for i := 0; i < concurrentRequests; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			log.Printf("请求 %d: 开始", idx+1)
			data, err := hotCache.GetWithCache(ctx, cacheKey, 30, func() (interface{}, error) {
				mu.Lock()
				dataSourceCallCount++
				mu.Unlock()
				return dataLoader()
			})

			if err != nil {
				log.Printf("请求 %d: 获取数据失败: %v", idx+1, err)
			} else {
				log.Printf("请求 %d: 获取数据成功", idx+1)
				// 避免打印太多数据
				if idx == 0 {
					log.Printf("数据示例: %v", data)
				}
			}
		}(i)
	}

	wg.Wait()

	// 记录结束时间
	duration := time.Since(startTime)
	log.Printf("所有请求完成，耗时: %v", duration)
	log.Printf("数据源被调用次数: %d", dataSourceCallCount)

	// 如果分布式锁工作正常，数据源应该只被调用1次
	if dataSourceCallCount == 1 {
		log.Println("缓存击穿保护正常工作！")
	} else {
		log.Printf("缓存击穿保护异常！数据源被调用了 %d 次", dataSourceCallCount)
	}
}

// 测试分布式锁
func testDistributedLock() {
	fmt.Println("\n=== 测试分布式锁 ===")

	// 获取分布式锁服务
	lockService := cache.GetLockService()
	if lockService == nil {
		log.Fatalf("获取分布式锁服务失败")
	}

	// 测试锁
	lockKey := "test:distributed:lock"
	concurrentRequests := 10

	var wg sync.WaitGroup
	lockAcquiredCount := 0
	var mu sync.Mutex

	log.Printf("发起 %d 个并发请求尝试获取同一个锁", concurrentRequests)

	for i := 0; i < concurrentRequests; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			// 尝试获取锁并执行操作
			err := lockService.WithLock(lockKey, 5, func() error {
				mu.Lock()
				lockAcquiredCount++
				mu.Unlock()

				log.Printf("请求 %d: 成功获取锁并执行操作", idx+1)
				time.Sleep(1 * time.Second) // 模拟执行一些工作
				return nil
			})

			if err != nil {
				if err == cache.ErrLockNotAcquired {
					log.Printf("请求 %d: 未能获取锁", idx+1)
				} else {
					log.Printf("请求 %d: 锁操作错误: %v", idx+1, err)
				}
			}
		}(i)
	}

	wg.Wait()

	log.Printf("所有请求完成，成功获取锁的次数: %d", lockAcquiredCount)

	// 如果分布式锁工作正常，应该只有一个请求成功获取锁
	if lockAcquiredCount == 1 {
		log.Println("分布式锁正常工作！")
	} else {
		log.Printf("分布式锁异常！%d 个请求都获取了锁", lockAcquiredCount)
	}
}

func main() {
	// 确保清理所有输出
	defer fmt.Println("所有测试完成！")

	// 如果有参数，选择性测试
	args := os.Args[1:]
	if len(args) > 0 {
		for _, arg := range args {
			switch arg {
			case "bloom":
				testBloomFilter()
			case "rate":
				testRateLimiter()
			case "cache":
				testHotCache()
			case "lock":
				testDistributedLock()
			default:
				log.Printf("未知测试: %s", arg)
			}
		}
		return
	}

	// 否则运行所有测试
	testBloomFilter()
	testRateLimiter()
	testHotCache()
	testDistributedLock()
}
