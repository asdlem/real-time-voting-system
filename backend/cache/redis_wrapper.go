package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisWrapper 是一个包装redis.Client的实现，符合RedisClient接口
type RedisWrapper struct {
	client *redis.Client
}

// NewRedisWrapper 创建Redis客户端包装器
func NewRedisWrapper(client *redis.Client) *RedisWrapper {
	return &RedisWrapper{
		client: client,
	}
}

// Set 实现Set方法
func (r *RedisWrapper) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	return r.client.Set(ctx, key, value, expiration)
}

// Get 实现Get方法
func (r *RedisWrapper) Get(ctx context.Context, key string) *redis.StringCmd {
	return r.client.Get(ctx, key)
}

// Del 实现Del方法
func (r *RedisWrapper) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	return r.client.Del(ctx, keys...)
}

// Exists 实现Exists方法
func (r *RedisWrapper) Exists(ctx context.Context, keys ...string) *redis.IntCmd {
	return r.client.Exists(ctx, keys...)
}

// Incr 实现Incr方法
func (r *RedisWrapper) Incr(ctx context.Context, key string) *redis.IntCmd {
	return r.client.Incr(ctx, key)
}

// IncrBy 实现IncrBy方法
func (r *RedisWrapper) IncrBy(ctx context.Context, key string, value int64) *redis.IntCmd {
	return r.client.IncrBy(ctx, key, value)
}

// Expire 实现Expire方法
func (r *RedisWrapper) Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	return r.client.Expire(ctx, key, expiration)
}

// TTL 实现TTL方法
func (r *RedisWrapper) TTL(ctx context.Context, key string) *redis.DurationCmd {
	return r.client.TTL(ctx, key)
}

// Pipeline 实现Pipeline方法
func (r *RedisWrapper) Pipeline() redis.Pipeliner {
	return r.client.Pipeline()
}

// TxPipeline 实现TxPipeline方法
func (r *RedisWrapper) TxPipeline() redis.Pipeliner {
	return r.client.TxPipeline()
}

// SetBit 实现SetBit方法
func (r *RedisWrapper) SetBit(ctx context.Context, key string, offset int64, value int) *redis.IntCmd {
	return r.client.SetBit(ctx, key, offset, value)
}

// GetBit 实现GetBit方法
func (r *RedisWrapper) GetBit(ctx context.Context, key string, offset int64) *redis.IntCmd {
	return r.client.GetBit(ctx, key, offset)
}

// HSet 实现HSet方法
func (r *RedisWrapper) HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	return r.client.HSet(ctx, key, values...)
}

// HGet 实现HGet方法
func (r *RedisWrapper) HGet(ctx context.Context, key, field string) *redis.StringCmd {
	return r.client.HGet(ctx, key, field)
}

// HGetAll 实现HGetAll方法
func (r *RedisWrapper) HGetAll(ctx context.Context, key string) *redis.MapStringStringCmd {
	return r.client.HGetAll(ctx, key)
}

// HDel 实现HDel方法
func (r *RedisWrapper) HDel(ctx context.Context, key string, fields ...string) *redis.IntCmd {
	return r.client.HDel(ctx, key, fields...)
}

// ZAdd 实现ZAdd方法
func (r *RedisWrapper) ZAdd(ctx context.Context, key string, members ...redis.Z) *redis.IntCmd {
	return r.client.ZAdd(ctx, key, members...)
}

// ZRemRangeByScore 实现ZRemRangeByScore方法
func (r *RedisWrapper) ZRemRangeByScore(ctx context.Context, key, min, max string) *redis.IntCmd {
	return r.client.ZRemRangeByScore(ctx, key, min, max)
}

// ZCard 实现ZCard方法
func (r *RedisWrapper) ZCard(ctx context.Context, key string) *redis.IntCmd {
	return r.client.ZCard(ctx, key)
}

// ZRem 实现ZRem方法
func (r *RedisWrapper) ZRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	return r.client.ZRem(ctx, key, members...)
}

// Eval 实现Eval方法
func (r *RedisWrapper) Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd {
	return r.client.Eval(ctx, script, keys, args...)
}

// EvalSha 实现EvalSha方法
func (r *RedisWrapper) EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) *redis.Cmd {
	return r.client.EvalSha(ctx, sha1, keys, args...)
}

// ScriptLoad 实现ScriptLoad方法
func (r *RedisWrapper) ScriptLoad(ctx context.Context, script string) *redis.StringCmd {
	return r.client.ScriptLoad(ctx, script)
}

// HIncrBy 实现HIncrBy方法
func (r *RedisWrapper) HIncrBy(ctx context.Context, key string, field string, incr int64) *redis.IntCmd {
	return r.client.HIncrBy(ctx, key, field, incr)
}

// SAdd 实现SAdd方法
func (r *RedisWrapper) SAdd(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	return r.client.SAdd(ctx, key, members...)
}

// SIsMember 实现SIsMember方法
func (r *RedisWrapper) SIsMember(ctx context.Context, key string, member interface{}) *redis.BoolCmd {
	return r.client.SIsMember(ctx, key, member)
}

// SMembers 实现SMembers方法
func (r *RedisWrapper) SMembers(ctx context.Context, key string) *redis.StringSliceCmd {
	return r.client.SMembers(ctx, key)
}

// SRem 实现SRem方法
func (r *RedisWrapper) SRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	return r.client.SRem(ctx, key, members...)
}

// GetRedisClient 获取Redis客户端包装器
func GetRedisClient() (RedisClient, error) {
	client, err := GetClient()
	if err != nil {
		return nil, err
	}
	return NewRedisWrapper(client), nil
}
