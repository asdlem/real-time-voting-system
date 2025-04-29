package cache

import "errors"

var (
	// ErrRedisNotAvailable Redis不可用错误
	ErrRedisNotAvailable = errors.New("Redis不可用")

	// ErrLockNotAcquired 获取锁失败错误
	ErrLockNotAcquired = errors.New("无法获取分布式锁")

	// ErrKeyNotFound 键不存在错误
	ErrKeyNotFound = errors.New("键不存在")
)
