package cache

import (
	"context"
	"hash/fnv"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// BloomFilter 布隆过滤器实现
type BloomFilter struct {
	redisClient RedisClient
	key         string
	hashCount   int
}

// NewBloomFilter 创建新的布隆过滤器
func NewBloomFilter(client RedisClient, key string, hashCount int) *BloomFilter {
	return &BloomFilter{
		redisClient: client,
		key:         "bloom:" + key,
		hashCount:   hashCount,
	}
}

// Add 添加元素到布隆过滤器
func (bf *BloomFilter) Add(ctx context.Context, item string) error {
	if bf.redisClient == nil {
		return ErrRedisNotAvailable
	}

	pipe := bf.redisClient.Pipeline()

	// 计算多个哈希值
	for i := 0; i < bf.hashCount; i++ {
		hashValue := bf.hash(item, i)
		pipe.SetBit(ctx, bf.key, hashValue, 1)
	}

	// 设置过期时间（可选，根据需要设置）
	pipe.Expire(ctx, bf.key, 24*time.Hour)

	_, err := pipe.Exec(ctx)
	return err
}

// Contains 检查元素是否可能存在于布隆过滤器中
func (bf *BloomFilter) Contains(ctx context.Context, item string) (bool, error) {
	if bf.redisClient == nil {
		return false, ErrRedisNotAvailable
	}

	pipe := bf.redisClient.Pipeline()

	// 获取所有哈希位置的值
	var cmds []*redis.IntCmd
	for i := 0; i < bf.hashCount; i++ {
		hashValue := bf.hash(item, i)
		cmds = append(cmds, pipe.GetBit(ctx, bf.key, hashValue))
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, err
	}

	// 如果任何一个位为0，则元素肯定不存在
	for _, cmd := range cmds {
		if cmd.Val() == 0 {
			return false, nil
		}
	}

	// 所有位都为1，元素可能存在（有一定的误判率）
	return true, nil
}

// hash 计算哈希值，使用不同的种子
func (bf *BloomFilter) hash(key string, seed int) int64 {
	h := fnv.New64a()
	h.Write([]byte(key))
	h.Write([]byte{byte(seed)})
	return int64(h.Sum64() % uint64(1<<30)) // 使用2^30位
}

// InitBloomFilter 初始化全局布隆过滤器
func InitBloomFilter() *BloomFilter {
	client, err := GetClient()
	if err != nil {
		log.Printf("初始化布隆过滤器失败: %v", err)
		return nil
	}

	return NewBloomFilter(client, "global_filter", 5)
}

// BoolCmd Redis布尔命令返回结果
type BoolCmd interface {
	Val() int64
}
