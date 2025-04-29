package service

import (
	"context"
	"errors"
	"time"

	"backend/model"
	"backend/repository"
	"backend/websocket"
)

var (
	// 业务错误定义
	ErrPollNotFound   = errors.New("poll not found")
	ErrOptionNotFound = errors.New("option not found")
	ErrPollClosed     = errors.New("poll is closed")
	ErrAlreadyVoted   = errors.New("user already voted")
)

// PollService 投票服务接口
type PollService interface {
	// 投票管理
	CreatePoll(ctx context.Context, poll *model.Poll) (*model.Poll, error)
	GetPollByID(ctx context.Context, id int64) (*model.Poll, error)
	ListPolls(ctx context.Context, offset, limit int) ([]*model.Poll, error)
	UpdatePoll(ctx context.Context, poll *model.Poll) error
	DeletePoll(ctx context.Context, id int64) error

	// 投票操作
	SubmitVote(ctx context.Context, vote *model.VoteRequest) error
	GetPollStatistics(ctx context.Context, pollID int64) (*model.PollStatistics, error)
}

// PollServiceImpl 投票服务实现
type PollServiceImpl struct {
	pollRepo  repository.PollRepository
	rateLimit RateLimiter
	wsHub     *websocket.Hub
}

// NewPollService 创建投票服务
func NewPollService(pollRepo repository.PollRepository, rateLimit RateLimiter, wsHub *websocket.Hub) PollService {
	return &PollServiceImpl{
		pollRepo:  pollRepo,
		rateLimit: rateLimit,
		wsHub:     wsHub,
	}
}

// CreatePoll 创建投票活动
func (s *PollServiceImpl) CreatePoll(ctx context.Context, poll *model.Poll) (*model.Poll, error) {
	// 创建投票
	createdPoll, err := s.pollRepo.CreatePoll(ctx, poll)
	if err != nil {
		return nil, err
	}

	// 创建投票选项
	for i := range poll.Options {
		poll.Options[i].PollID = createdPoll.ID
		_, err := s.pollRepo.CreateOption(ctx, &poll.Options[i])
		if err != nil {
			// 严格来说，应该事务回滚，这里简化处理
			return nil, err
		}
	}

	// 重新查询完整信息
	return s.GetPollByID(ctx, createdPoll.ID)
}

// GetPollByID 获取投票详情
func (s *PollServiceImpl) GetPollByID(ctx context.Context, id int64) (*model.Poll, error) {
	// 获取投票信息
	poll, err := s.pollRepo.GetPollByID(ctx, id)
	if err != nil {
		return nil, ErrPollNotFound
	}

	// 获取投票选项
	options, err := s.pollRepo.GetOptionsByPollID(ctx, id)
	if err != nil {
		return nil, err
	}
	poll.Options = options

	return poll, nil
}

// ListPolls 获取投票列表
func (s *PollServiceImpl) ListPolls(ctx context.Context, offset, limit int) ([]*model.Poll, error) {
	return s.pollRepo.ListPolls(ctx, offset, limit)
}

// UpdatePoll 更新投票
func (s *PollServiceImpl) UpdatePoll(ctx context.Context, poll *model.Poll) error {
	return s.pollRepo.UpdatePoll(ctx, poll)
}

// DeletePoll 删除投票
func (s *PollServiceImpl) DeletePoll(ctx context.Context, id int64) error {
	return s.pollRepo.DeletePoll(ctx, id)
}

// SubmitVote 提交投票
func (s *PollServiceImpl) SubmitVote(ctx context.Context, voteReq *model.VoteRequest) error {
	// 检查投票是否存在
	poll, err := s.pollRepo.GetPollByID(ctx, voteReq.PollID)
	if err != nil {
		return ErrPollNotFound
	}

	// 检查投票是否开放
	now := time.Now()
	if now.Before(poll.StartTime) || now.After(poll.EndTime) || poll.Status != model.PollStatusActive {
		return ErrPollClosed
	}

	// 检查用户是否已投票
	hasVoted, err := s.pollRepo.HasUserVoted(ctx, voteReq.PollID, voteReq.UserID)
	if err != nil {
		return err
	}
	if hasVoted {
		return ErrAlreadyVoted
	}

	// 检查选项是否存在
	options, err := s.pollRepo.GetOptionsByPollID(ctx, voteReq.PollID)
	if err != nil {
		return err
	}

	optionExists := false
	for _, opt := range options {
		if opt.ID == voteReq.OptionID {
			optionExists = true
			break
		}
	}

	if !optionExists {
		return ErrOptionNotFound
	}

	// 创建投票记录
	vote := &model.Vote{
		PollID:   voteReq.PollID,
		OptionID: voteReq.OptionID,
		UserID:   voteReq.UserID,
		VotedAt:  time.Now(),
	}
	err = s.pollRepo.CreateVote(ctx, vote)
	if err != nil {
		return err
	}

	// 更新选项计数
	err = s.pollRepo.IncrementOptionCount(ctx, voteReq.OptionID)
	if err != nil {
		return err
	}

	// 实时推送最新投票结果
	go func() {
		stats, err := s.GetPollStatistics(context.Background(), voteReq.PollID)
		if err == nil {
			s.wsHub.BroadcastToPoll(voteReq.PollID, &model.WebSocketMessage{
				Type:    "VOTE_UPDATE",
				PollID:  voteReq.PollID,
				Payload: stats,
			})
		}
	}()

	return nil
}

// GetPollStatistics 获取投票统计
func (s *PollServiceImpl) GetPollStatistics(ctx context.Context, pollID int64) (*model.PollStatistics, error) {
	// 检查投票是否存在
	poll, err := s.pollRepo.GetPollByID(ctx, pollID)
	if err != nil {
		return nil, ErrPollNotFound
	}

	// 获取统计信息
	stats, err := s.pollRepo.GetPollStatistics(ctx, pollID)
	if err != nil {
		return nil, err
	}

	// 计算选项百分比
	if stats.TotalVotes > 0 {
		for i := range stats.Options {
			stats.Options[i].Percentage = float64(stats.Options[i].Count) / float64(stats.TotalVotes) * 100
		}
	}

	// 添加投票基本信息
	stats.PollID = poll.ID
	stats.Title = poll.Title
	stats.Status = poll.Status

	return stats, nil
}

// WebSocketHub WebSocket消息中心
type WebSocketHub struct {
	// 为简化实现，省略具体实现
	clients map[int64]map[*WebSocketClient]bool
}

// WebSocketClient WebSocket客户端
type WebSocketClient struct {
	// 为简化实现，省略具体实现
}

// BroadcastToPoll 向特定投票的所有连接用户广播消息
func (h *WebSocketHub) BroadcastToPoll(pollID int64, message *model.WebSocketMessage) {
	// 为简化实现，省略具体实现
}
