package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisClient 定义Redis客户端接口
type RedisClient interface {
	// 基本操作
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Get(ctx context.Context, key string) *redis.StringCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Exists(ctx context.Context, keys ...string) *redis.IntCmd
	Incr(ctx context.Context, key string) *redis.IntCmd
	IncrBy(ctx context.Context, key string, value int64) *redis.IntCmd
	Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd
	TTL(ctx context.Context, key string) *redis.DurationCmd

	// 管道操作
	Pipeline() redis.Pipeliner
	TxPipeline() redis.Pipeliner

	// 位操作
	SetBit(ctx context.Context, key string, offset int64, value int) *redis.IntCmd
	GetBit(ctx context.Context, key string, offset int64) *redis.IntCmd

	// 哈希操作
	HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd
	HGet(ctx context.Context, key, field string) *redis.StringCmd
	HGetAll(ctx context.Context, key string) *redis.MapStringStringCmd
	HDel(ctx context.Context, key string, fields ...string) *redis.IntCmd
	HIncrBy(ctx context.Context, key string, field string, incr int64) *redis.IntCmd

	// 集合操作
	SAdd(ctx context.Context, key string, members ...interface{}) *redis.IntCmd
	SIsMember(ctx context.Context, key string, member interface{}) *redis.BoolCmd
	SMembers(ctx context.Context, key string) *redis.StringSliceCmd
	SRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd

	// 有序集合操作
	ZAdd(ctx context.Context, key string, members ...redis.Z) *redis.IntCmd
	ZRemRangeByScore(ctx context.Context, key, min, max string) *redis.IntCmd
	ZCard(ctx context.Context, key string) *redis.IntCmd
	ZRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd

	// Lua脚本
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd
	EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) *redis.Cmd
	ScriptLoad(ctx context.Context, script string) *redis.StringCmd
}
