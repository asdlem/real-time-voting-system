package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"realtime-voting-backend/cache"
	"realtime-voting-backend/database"
	"realtime-voting-backend/mq"
)

// VoteRequest 投票请求结构
type VoteRequest struct {
	PollID   string `json:"poll_id"`
	OptionID string `json:"option_id"`
}

// HandleVote 处理投票请求
func HandleVote(w http.ResponseWriter, r *http.Request) {
	// 1. 解析请求
	var req VoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "无效的请求格式", http.StatusBadRequest)
		return
	}

	// 验证投票数据
	if req.PollID == "" || req.OptionID == "" {
		http.Error(w, "投票ID和选项ID不能为空", http.StatusBadRequest)
		return
	}

	// 2. 获取poll信息，验证poll是否存在和是否过期
	poll, err := database.GetPoll(req.PollID)
	if err != nil {
		http.Error(w, fmt.Sprintf("获取投票失败: %v", err), http.StatusInternalServerError)
		return
	}

	if poll == nil {
		http.Error(w, "投票不存在", http.StatusNotFound)
		return
	}

	// 检查投票是否已结束
	if poll.EndTime.Valid && poll.EndTime.Time.Before(time.Now()) {
		http.Error(w, "投票已结束", http.StatusForbidden)
		return
	}

	// 3. 验证选项是否存在
	optionExists := false
	for _, option := range poll.Options {
		if option.ID == req.OptionID {
			optionExists = true
			break
		}
	}

	if !optionExists {
		http.Error(w, "选项不存在", http.StatusBadRequest)
		return
	}

	// 4. 发送消息到消息队列
	err = mq.SendVoteMessage(req.PollID, req.OptionID)
	if err != nil {
		log.Printf("发送投票消息失败: %v", err)
		http.Error(w, "处理投票失败", http.StatusInternalServerError)
		return
	}

	// 5. 返回成功响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "投票已提交",
	})
}

// ProcessVoteMessage 处理投票消息
func ProcessVoteMessage(pollID string, optionID string) error {
	// 1. 获取投票详情
	poll, err := database.GetPoll(pollID)
	if err != nil {
		return fmt.Errorf("获取投票详情失败: %v", err)
	}

	if poll == nil {
		return fmt.Errorf("投票不存在")
	}

	// 2. 验证投票是否已结束
	if poll.EndTime.Valid && poll.EndTime.Time.Before(time.Now()) {
		return fmt.Errorf("投票已结束")
	}

	// 3. 更新缓存中的投票计数
	count, err := cache.IncrementVoteCount(pollID, optionID, 1)
	if err != nil {
		log.Printf("增加缓存计数失败: %v", err)
	} else {
		log.Printf("选项 %s 当前票数: %d", optionID, count)
	}

	// 4. 更新数据库
	err = database.AddVote(context.Background(), pollID, optionID)
	if err != nil {
		return fmt.Errorf("更新数据库投票计数失败: %v", err)
	}

	// 5. 广播投票更新
	BroadcastPollUpdateStr(pollID, nil)

	return nil
}
