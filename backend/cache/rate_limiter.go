package cache

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// RateLimiter 限流器接口
type RateLimiter interface {
	// Allow 判断请求是否允许通过
	Allow(ctx context.Context) (bool, error)
}

// TokenBucketRateLimiter 令牌桶限流器实现
type TokenBucketRateLimiter struct {
	redisClient RedisClient
	key         string
	rate        int // 每秒生成的令牌数量
	burst       int // 令牌桶最大容量
}

// NewTokenBucketRateLimiter 创建新的令牌桶限流器
func NewTokenBucketRateLimiter(client RedisClient, key string, rate, burst int) *TokenBucketRateLimiter {
	return &TokenBucketRateLimiter{
		redisClient: client,
		key:         fmt.Sprintf("rate_limit:%s", key),
		rate:        rate,
		burst:       burst,
	}
}

// Allow 判断请求是否允许通过
func (l *TokenBucketRateLimiter) Allow(ctx context.Context) (bool, error) {
	if l.redisClient == nil {
		return false, ErrRedisNotAvailable
	}

	// 令牌桶算法的Lua脚本
	script := `
	local key = KEYS[1]
	local now = tonumber(ARGV[1])
	local rate = tonumber(ARGV[2])
	local burst = tonumber(ARGV[3])
	local period = 1 -- 1秒为单位
	
	-- 获取当前桶中的令牌数和上次更新时间
	local tokens_key = key .. ":tokens"
	local timestamp_key = key .. ":ts"
	
	local tokens = tonumber(redis.call("get", tokens_key) or burst)
	local last_update = tonumber(redis.call("get", timestamp_key) or 0)
	
	-- 计算距离上次更新经过的时间，添加相应的令牌
	local elapsed = math.max(0, now - last_update)
	local new_tokens = math.min(burst, tokens + elapsed * rate)
	
	-- 判断是否有足够的令牌
	if new_tokens < 1 then
		return 0
	end
	
	-- 消耗一个令牌
	new_tokens = new_tokens - 1
	
	-- 更新令牌数和时间戳
	redis.call("setex", tokens_key, period * 2, new_tokens)
	redis.call("setex", timestamp_key, period * 2, now)
	
	return 1
	`

	// 执行Lua脚本
	now := time.Now().Unix()
	keys := []string{l.key}
	args := []interface{}{now, l.rate, l.burst}

	// 执行脚本
	result, err := l.redisClient.Eval(ctx, script, keys, args...).Result()
	if err != nil {
		return false, err
	}

	return result.(int64) == 1, nil
}

// SlidingWindowRateLimiter 滑动窗口限流器
type SlidingWindowRateLimiter struct {
	redisClient RedisClient
	key         string
	windowSize  time.Duration // 窗口大小
	limit       int           // 窗口内允许的最大请求数
}

// NewSlidingWindowRateLimiter 创建新的滑动窗口限流器
func NewSlidingWindowRateLimiter(client RedisClient, key string, windowSize time.Duration, limit int) *SlidingWindowRateLimiter {
	return &SlidingWindowRateLimiter{
		redisClient: client,
		key:         fmt.Sprintf("sliding_window:%s", key),
		windowSize:  windowSize,
		limit:       limit,
	}
}

// Allow 判断请求是否允许通过
func (l *SlidingWindowRateLimiter) Allow(ctx context.Context) (bool, error) {
	if l.redisClient == nil {
		return false, ErrRedisNotAvailable
	}

	now := time.Now().UnixNano() / int64(time.Millisecond)
	windowStart := now - int64(l.windowSize/time.Millisecond)
	requestID := uuid.New().String()

	// 使用有序集合记录请求
	pipe := l.redisClient.Pipeline()
	pipe.ZAdd(ctx, l.key, redis.Z{Score: float64(now), Member: requestID})
	pipe.ZRemRangeByScore(ctx, l.key, "0", strconv.FormatInt(windowStart, 10))
	pipe.ZCard(ctx, l.key)
	pipe.Expire(ctx, l.key, l.windowSize*2) // 设置过期时间，避免集合无限增长

	cmds, err := pipe.Exec(ctx)
	if err != nil {
		return false, err
	}

	// 获取当前窗口内的请求数量
	count := cmds[2].(*redis.IntCmd).Val()

	// 如果超过限制，移除当前请求
	if count > int64(l.limit) {
		l.redisClient.ZRem(ctx, l.key, requestID)
		return false, nil
	}

	return true, nil
}

// UserRateLimiter 用户级别限流器，每个用户有自己的限流器
type UserRateLimiter struct {
	redisClient   RedisClient
	globalLimiter RateLimiter
	keyPrefix     string
	rate          int
	burst         int
	limiters      map[string]RateLimiter
}

// NewUserRateLimiter 创建新的用户级别限流器
func NewUserRateLimiter(client RedisClient, keyPrefix string, globalRate, globalBurst, userRate, userBurst int) *UserRateLimiter {
	return &UserRateLimiter{
		redisClient:   client,
		globalLimiter: NewTokenBucketRateLimiter(client, keyPrefix+":global", globalRate, globalBurst),
		keyPrefix:     keyPrefix,
		rate:          userRate,
		burst:         userBurst,
		limiters:      make(map[string]RateLimiter),
	}
}

// GetUserLimiter 获取用户的限流器
func (l *UserRateLimiter) GetUserLimiter(userID string) RateLimiter {
	if limiter, ok := l.limiters[userID]; ok {
		return limiter
	}

	limiter := NewTokenBucketRateLimiter(l.redisClient, l.keyPrefix+":user:"+userID, l.rate, l.burst)
	l.limiters[userID] = limiter
	return limiter
}

// AllowUser 判断用户请求是否允许通过
func (l *UserRateLimiter) AllowUser(ctx context.Context, userID string) (bool, error) {
	// 先检查全局限流
	allowed, err := l.globalLimiter.Allow(ctx)
	if err != nil || !allowed {
		if err != nil {
			log.Printf("全局限流检查失败: %v", err)
		}
		return allowed, err
	}

	// 再检查用户级别限流
	userLimiter := l.GetUserLimiter(userID)
	return userLimiter.Allow(ctx)
}
