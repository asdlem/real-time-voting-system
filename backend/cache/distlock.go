package cache

import (
	"context"
	"log"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
)

var (
	// rs 全局的Redsync实例
	rs *redsync.Redsync
)

// DistributedLockService 分布式锁服务
type DistributedLockService struct {
	rs *redsync.Redsync
}

// InitDistLock 初始化分布式锁
func InitDistLock() {
	// 使用现有的Redis客户端
	client, err := GetClient()
	if err != nil {
		log.Printf("初始化分布式锁失败: %v", err)
		return
	}

	// 创建Redis连接池
	pool := goredis.NewPool(client)

	// 创建Redsync实例
	rs = redsync.New(pool)
	log.Println("分布式锁初始化成功")
}

// GetLockService 获取分布式锁服务实例
func GetLockService() *DistributedLockService {
	if rs == nil {
		InitDistLock()
	}
	return &DistributedLockService{rs: rs}
}

// AcquireLock 尝试获取锁，带有超时时间
func (s *DistributedLockService) AcquireLock(lockName string, expiry time.Duration) (mutex *redsync.Mutex, acquired bool, err error) {
	mutex = s.rs.NewMutex(lockName,
		redsync.WithExpiry(expiry),
		redsync.WithTries(5),                        // 最大重试次数
		redsync.WithRetryDelay(50*time.Millisecond), // 重试延迟
		redsync.WithDriftFactor(0.01),               // 时钟漂移因子
	)

	// 尝试获取锁
	err = mutex.Lock()
	if err != nil {
		return nil, false, err
	}

	return mutex, true, nil
}

// ReleaseLock 释放锁
func (s *DistributedLockService) ReleaseLock(mutex *redsync.Mutex) (bool, error) {
	return mutex.Unlock()
}

// WithLock 在锁内执行操作
func (s *DistributedLockService) WithLock(lockName string, expiry time.Duration, action func() error) error {
	mutex, acquired, err := s.AcquireLock(lockName, expiry)
	if err != nil {
		return err
	}

	if !acquired {
		return ErrLockNotAcquired
	}

	// 确保解锁
	defer func() {
		_, _ = s.ReleaseLock(mutex)
	}()

	// 执行业务逻辑
	return action()
}

// TryWithLock 尝试在锁内执行操作，如果获取锁失败立即返回
func (s *DistributedLockService) TryWithLock(ctx context.Context, lockName string, expiry time.Duration, action func() error) error {
	// 尝试获取锁，不等待
	mutex, acquired, err := s.AcquireLock(lockName, expiry)
	if err != nil {
		return err
	}

	if !acquired {
		return ErrLockNotAcquired
	}

	// 确保解锁
	defer func() {
		_, _ = s.ReleaseLock(mutex)
	}()

	// 带上下文执行，支持超时或取消
	done := make(chan error, 1)
	go func() {
		done <- action()
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
