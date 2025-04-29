package repository

import (
	"context"
	"errors"

	"backend/model"
)

// PollRepository 定义投票数据访问接口
type PollRepository interface {
	// 投票活动相关方法
	CreatePoll(ctx context.Context, poll *model.Poll) (int64, error)
	GetPollByID(ctx context.Context, id int64) (*model.Poll, error)
	ListPolls(ctx context.Context, offset, limit int) ([]*model.Poll, error)
	UpdatePoll(ctx context.Context, poll *model.Poll) error
	DeletePoll(ctx context.Context, id int64) error

	// 投票选项相关方法
	CreateOption(ctx context.Context, option *model.Option) (int64, error)
	GetOptionsByPollID(ctx context.Context, pollID int64) ([]model.Option, error)
	UpdateOption(ctx context.Context, option *model.Option) error

	// 投票记录相关方法
	CreateVote(ctx context.Context, vote *model.Vote) (int64, error)
	GetVotesByPollID(ctx context.Context, pollID int64) ([]*model.Vote, error)
	HasUserVoted(ctx context.Context, pollID int64, userID string) (bool, error)

	// 投票统计相关方法
	GetPollStatistics(ctx context.Context, pollID int64) (*model.PollStatistics, error)
	IncrementOptionCount(ctx context.Context, optionID int64) error
}

// PollCacheRepository 定义投票缓存接口
type PollCacheRepository interface {
	// 投票活动缓存方法
	SetPoll(ctx context.Context, poll *model.Poll) error
	GetPoll(ctx context.Context, id int64) (*model.Poll, error)
	DeletePoll(ctx context.Context, id int64) error

	// 投票统计缓存方法
	SetPollStatistics(ctx context.Context, stats *model.PollStatistics) error
	GetPollStatistics(ctx context.Context, pollID int64) (*model.PollStatistics, error)
	IncrementOptionCount(ctx context.Context, pollID, optionID int64) error

	// 用户投票记录缓存方法
	SetUserVoted(ctx context.Context, pollID int64, userID string, optionID int64) error
	HasUserVoted(ctx context.Context, pollID int64, userID string) (bool, int64, error)
}

// BloomFilterRepository 定义布隆过滤器接口
type BloomFilterRepository interface {
	// 投票ID相关方法
	AddPollID(ctx context.Context, id int64) error
	ExistsPollID(ctx context.Context, id int64) (bool, error)
}

// CachedPollRepository 实现带缓存的投票数据仓库
type CachedPollRepository struct {
	// 实际数据库操作实现
	db PollRepository
	// 缓存实现
	cache PollCacheRepository
	// 布隆过滤器
	bloomFilter BloomFilterRepository
}

// NewCachedPollRepository 创建带缓存的投票数据仓库
func NewCachedPollRepository(db PollRepository, cache PollCacheRepository, bloom BloomFilterRepository) *CachedPollRepository {
	return &CachedPollRepository{
		db:          db,
		cache:       cache,
		bloomFilter: bloom,
	}
}

// CreatePoll 创建投票活动
func (r *CachedPollRepository) CreatePoll(ctx context.Context, poll *model.Poll) (int64, error) {
	// 1. 写入数据库
	id, err := r.db.CreatePoll(ctx, poll)
	if err != nil {
		return 0, err
	}

	// 2. 设置缓存
	poll.ID = id
	err = r.cache.SetPoll(ctx, poll)
	if err != nil {
		// 缓存错误只记录日志，不影响返回结果
		// logger.Error("Failed to cache poll", zap.Error(err))
	}

	// 3. 更新布隆过滤器
	err = r.bloomFilter.AddPollID(ctx, id)
	if err != nil {
		// logger.Error("Failed to update bloom filter", zap.Error(err))
	}

	return id, nil
}

// GetPollByID 获取投票活动详情
func (r *CachedPollRepository) GetPollByID(ctx context.Context, id int64) (*model.Poll, error) {
	// 1. 检查布隆过滤器
	exists, err := r.bloomFilter.ExistsPollID(ctx, id)
	if err == nil && !exists {
		return nil, ErrPollNotFound
	}

	// 2. 尝试从缓存获取
	poll, err := r.cache.GetPoll(ctx, id)
	if err == nil && poll != nil {
		// 缓存命中
		return poll, nil
	}

	// 3. 缓存未命中，从数据库获取
	poll, err = r.db.GetPollByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 4. 设置缓存
	err = r.cache.SetPoll(ctx, poll)
	if err != nil {
		// logger.Error("Failed to cache poll", zap.Error(err))
	}

	return poll, nil
}

// 省略其他方法实现，逻辑类似...

// ErrPollNotFound 投票活动不存在错误
var ErrPollNotFound = errors.New("poll not found")
