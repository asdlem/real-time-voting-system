package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"
)

// HotCache 热点缓存管理器
type HotCache struct {
	redisClient  RedisClient
	lockService  *DistributedLockService
	bloomFilter  *BloomFilter
	refreshTimer *time.Ticker
	done         chan struct{}
}

// NewHotCache 创建新的热点缓存管理器
func NewHotCache(client RedisClient, lockService *DistributedLockService, bloomFilter *BloomFilter) *HotCache {
	return &HotCache{
		redisClient: client,
		lockService: lockService,
		bloomFilter: bloomFilter,
		done:        make(chan struct{}),
	}
}

// GetWithCache 获取缓存，带击穿保护（互斥锁）
func (c *HotCache) GetWithCache(ctx context.Context, key string, ttl time.Duration, loader func() (interface{}, error)) (interface{}, error) {
	if c.redisClient == nil {
		return nil, ErrRedisNotAvailable
	}

	// 1. 尝试从缓存获取
	data, err := c.redisClient.Get(ctx, key).Result()
	if err == nil {
		// 缓存命中
		if data == "NULL" {
			// 空值缓存
			return nil, nil
		}

		// 解析缓存数据
		var result interface{}
		if err := json.Unmarshal([]byte(data), &result); err != nil {
			log.Printf("解析缓存数据失败: %v", err)
			// 继续执行，从数据源重新加载
		} else {
			return result, nil
		}
	}

	// 2. 使用分布式锁防止缓存击穿
	lockKey := fmt.Sprintf("cache_lock:%s", key)
	var result interface{}

	err = c.lockService.WithLock(lockKey, 5*time.Second, func() error {
		// 双重检查，可能其他线程已经填充了缓存
		data, err := c.redisClient.Get(ctx, key).Result()
		if err == nil {
			if data == "NULL" {
				result = nil
				return nil
			}

			return json.Unmarshal([]byte(data), &result)
		}

		// 从数据源加载
		loadedData, err := loader()
		if err != nil {
			return err
		}

		result = loadedData

		// 缓存结果
		if loadedData == nil {
			// 缓存空值，较短的过期时间
			c.redisClient.Set(ctx, key, "NULL", ttl/4)
		} else {
			// 使用随机过期时间，避免缓存雪崩
			jsonData, _ := json.Marshal(loadedData)
			expiration := ttl + time.Duration(rand.Intn(int(ttl/10)))
			c.redisClient.Set(ctx, key, string(jsonData), expiration)

			// 如果有布隆过滤器，添加到过滤器
			if c.bloomFilter != nil {
				c.bloomFilter.Add(ctx, key)
			}
		}

		return nil
	})

	return result, err
}

// StartHotDataRefresher 启动热点数据定时刷新
func (c *HotCache) StartHotDataRefresher(refreshInterval time.Duration, getHotKeysFunc func() ([]string, error), refreshFunc func(key string) (interface{}, error)) {
	c.refreshTimer = time.NewTicker(refreshInterval)

	go func() {
		for {
			select {
			case <-c.refreshTimer.C:
				// 获取热点key列表
				hotKeys, err := getHotKeysFunc()
				if err != nil {
					log.Printf("获取热点key列表失败: %v", err)
					continue
				}

				// 异步刷新每个热点key
				for _, key := range hotKeys {
					go func(k string) {
						// 使用短暂的锁避免并发刷新
						lockKey := fmt.Sprintf("refresh_lock:%s", k)
						c.lockService.TryWithLock(context.Background(), lockKey, 1*time.Second, func() error {
							// 从数据源加载最新数据
							data, err := refreshFunc(k)
							if err != nil {
								log.Printf("刷新热点数据失败: %v", err)
								return err
							}

							// 更新缓存，不过期（由后台刷新保证最新）
							jsonData, _ := json.Marshal(data)
							c.redisClient.Set(context.Background(), k, string(jsonData), 0)
							log.Printf("已刷新热点数据: %s", k)
							return nil
						})
					}(key)
				}
			case <-c.done:
				return
			}
		}
	}()
}

// StopRefresher 停止刷新器
func (c *HotCache) StopRefresher() {
	if c.refreshTimer != nil {
		c.refreshTimer.Stop()
		close(c.done)
	}
}

// PrewarmCache 预热缓存
func (c *HotCache) PrewarmCache(ctx context.Context, keys []string, loader func(key string) (interface{}, error), ttl time.Duration) {
	log.Println("开始预热缓存...")

	for _, key := range keys {
		go func(k string) {
			data, err := loader(k)
			if err != nil {
				log.Printf("预热缓存失败 %s: %v", k, err)
				return
			}

			if data != nil {
				jsonData, _ := json.Marshal(data)
				// 使用随机过期时间
				expiration := ttl + time.Duration(rand.Intn(int(ttl/10)))
				c.redisClient.Set(ctx, k, string(jsonData), expiration)

				// 添加到布隆过滤器
				if c.bloomFilter != nil {
					c.bloomFilter.Add(ctx, k)
				}

				log.Printf("已预热缓存: %s", k)
			}
		}(key)
	}

	log.Println("缓存预热队列已启动")
}
