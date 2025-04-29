package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"realtime-voting-backend/cache"
	"realtime-voting-backend/database"
	"realtime-voting-backend/models"
	"realtime-voting-backend/mq"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 全局MQAdapter引用
var mqAdapter *mq.MQAdapter

// InitHandler 初始化处理程序，设置MQAdapter引用
func InitHandler(adapter *mq.MQAdapter) {
	mqAdapter = adapter
	log.Println("消息队列适配器已设置到处理程序")
}

// CreatePollInput defines the expected input structure for creating a poll
type CreatePollInput struct {
	Question    string              `json:"question,Question" binding:"required"`
	Description string              `json:"description,Description,omitempty"`                // 添加Description字段
	PollType    models.PollType     `json:"poll_type,PollType" binding:"omitempty,oneof=0 1"` // 支持poll_type和PollType两种格式
	Options     []CreateOptionInput `json:"options,Options" binding:"required,min=2,dive"`
	EndTime     *time.Time          `json:"end_time,omitempty"`    // Optional end time
	MinOptions  *int                `json:"min_options,omitempty"` // For multiple choice polls
	MaxOptions  *int                `json:"max_options,omitempty"` // For multiple choice polls
}

// CreateOptionInput defines the structure for options when creating a poll
type CreateOptionInput struct {
	Text string `json:"text,Text" binding:"required"`
}

// CreatePoll handles the creation of a new poll
func CreatePoll(c *gin.Context) {
	var input CreatePollInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 记录请求数据
	log.Printf("收到创建投票请求: Question=%s, PollType=%d", input.Question, input.PollType)

	// Basic validation: Ensure at least two options are provided
	if len(input.Options) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "A poll must have at least two options"})
		return
	}

	// Validate end time if provided
	if input.EndTime != nil && input.EndTime.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "End time must be in the future"})
		return
	}

	poll := models.Poll{
		Question:    input.Question,
		Description: input.Description, // 添加Description字段
		PollType:    input.PollType,
		IsActive:    true, // Default to active
		EndTime:     input.EndTime,
	}

	log.Printf("准备创建投票: Question=%s, PollType=%d", poll.Question, poll.PollType)

	// Use a transaction to ensure atomicity
	tx := database.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	// Create the poll record
	if err := tx.Create(&poll).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create poll"})
		return
	}

	log.Printf("投票创建成功: ID=%d, Question=%s, PollType=%d", poll.ID, poll.Question, poll.PollType)

	// Create the poll options
	options := make([]models.PollOption, len(input.Options))
	for i, optInput := range input.Options {
		options[i] = models.PollOption{
			PollID: poll.ID,
			Text:   optInput.Text,
		}
	}

	if err := tx.Create(&options).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create poll options"})
		return
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	// Reload the poll with options to return the full object
	var createdPoll models.Poll
	if err := database.DB.Preload("Options").First(&createdPoll, poll.ID).Error; err != nil {
		// Log the error but return the basic poll info if reload fails
		log.Printf("Warning: Failed to reload poll with options after creation: %v", err)
		c.JSON(http.StatusCreated, poll)
		return
	}

	// 记录重新加载后的投票类型
	log.Printf("重新加载后的投票: ID=%d, Question=%s, PollType=%d", createdPoll.ID, createdPoll.Question, createdPoll.PollType)

	c.JSON(http.StatusCreated, createdPoll)
}

// GetPolls retrieves a list of all polls (consider pagination for large datasets)
func GetPolls(c *gin.Context) {
	var polls []models.Poll
	// Preload options to include them in the response
	if err := database.DB.Preload("Options").Order("created_at desc").Find(&polls).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve polls"})
		return
	}
	c.JSON(http.StatusOK, polls)
}

// GetPoll handles retrieving a single poll by ID
func GetPoll(c *gin.Context) {
	// Get ID from the URL parameter
	pollIDStr := c.Param("id")
	pollID, err := strconv.ParseUint(pollIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid poll ID format"})
		return
	}
	pollUintID := uint(pollID)

	// Find the poll in the database
	var poll models.Poll
	if err := database.DB.Preload("Options").First(&poll, pollUintID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Poll not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve poll"})
		}
		return
	}

	// 记录从数据库读取到的投票类型
	log.Printf("从数据库读取投票: ID=%d, Question=%s, PollType=%d", poll.ID, poll.Question, poll.PollType)

	// Calculate the total votes and percentages
	var totalVotes int64 = 0
	for _, option := range poll.Options {
		totalVotes += option.Votes
	}

	// Create response with options including vote percentages
	type OptionWithPercentage struct {
		ID         uint    `json:"id"`
		Text       string  `json:"text"`
		Votes      int64   `json:"votes"`
		Percentage float64 `json:"percentage"`
	}

	options := make([]OptionWithPercentage, len(poll.Options))
	for i, option := range poll.Options {
		percentage := 0.0
		if totalVotes > 0 {
			percentage = float64(option.Votes) / float64(totalVotes) * 100
		}
		options[i] = OptionWithPercentage{
			ID:         option.ID,
			Text:       option.Text,
			Votes:      option.Votes,
			Percentage: percentage,
		}
	}

	// Check if the poll is expired but still marked as active
	var isActive bool = poll.IsActive
	if poll.EndTime != nil && time.Now().After(*poll.EndTime) && poll.IsActive {
		isActive = false
		// 不自动更新数据库，只在响应中标记为非活动
	}

	// Return the poll with calculated percentages
	c.JSON(http.StatusOK, gin.H{
		"id":          poll.ID,
		"question":    poll.Question,
		"description": poll.Description, // 在GetPoll响应中添加Description字段
		"poll_type":   poll.PollType,
		"is_active":   isActive,
		"options":     options,
		"created_at":  poll.CreatedAt,
		"updated_at":  poll.UpdatedAt,
		"end_time":    poll.EndTime,
	})
}

// UpdatePollInput defines the expected input structure for updating a poll
// Note: We might want separate inputs/logic for updating options vs poll details
type UpdatePollInput struct {
	Question    *string             `json:"Question,question"` // Use pointers to distinguish between empty and not provided
	PollType    *models.PollType    `json:"PollType,poll_type" binding:"omitempty,oneof=0 1"`
	IsActive    *bool               `json:"is_active,IsActive"`
	Description *string             `json:"Description,description,omitempty"`
	EndTime     *time.Time          `json:"end_time,EndTime,omitempty"`
	Options     []UpdateOptionInput `json:"Options,options,omitempty"` // 支持更新选项
}

// UpdateOptionInput 定义选项更新的结构
type UpdateOptionInput struct {
	ID   uint   `json:"ID,id,omitempty"` // 选项ID，如果是新选项则为空
	Text string `json:"Text,text" binding:"required"`
}

// UpdatePoll handles updating an existing poll's details (not options)
func UpdatePoll(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid poll ID format"})
		return
	}

	var input UpdatePollInput
	if err := c.ShouldBindJSON(&input); err != nil {
		log.Printf("绑定JSON失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 记录收到的输入内容
	log.Printf("收到更新投票请求 ID:%d, 输入内容: %+v", id, input)
	if input.PollType != nil {
		log.Printf("请求中包含poll_type更新: %d", *input.PollType)
	}

	var poll models.Poll
	result := database.DB.First(&poll, uint(id))
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Poll not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve poll"})
		}
		return
	}

	// 记录当前投票信息
	log.Printf("当前投票信息: ID:%d, Question:%s, PollType:%d, IsActive:%v",
		poll.ID, poll.Question, poll.PollType, poll.IsActive)

	// 直接更新字段而非使用map
	needsUpdate := false

	if input.Question != nil {
		poll.Question = *input.Question
		needsUpdate = true
		log.Printf("更新问题: %s", *input.Question)
	}

	if input.PollType != nil {
		oldPollType := poll.PollType
		poll.PollType = *input.PollType
		needsUpdate = true
		log.Printf("更新poll_type: %d -> %d", oldPollType, *input.PollType)
	}

	if input.IsActive != nil {
		poll.IsActive = *input.IsActive
		needsUpdate = true
		log.Printf("更新活动状态: %v", *input.IsActive)
	}

	if input.Description != nil {
		poll.Description = *input.Description
		needsUpdate = true
		log.Printf("更新描述信息: %s", *input.Description)
	}

	if input.EndTime != nil {
		poll.EndTime = input.EndTime
		needsUpdate = true
		log.Printf("更新结束时间: %v", *input.EndTime)
	} else if input.EndTime == nil && c.Request.Method == "PUT" {
		// 在PUT请求中，如果前端未提供EndTime，可能需要清除已有的EndTime
		poll.EndTime = nil
		needsUpdate = true
		log.Printf("清除结束时间")
	}

	// 执行更新
	if needsUpdate {
		log.Printf("执行数据库更新操作...")
		if err := database.DB.Save(&poll).Error; err != nil {
			log.Printf("更新投票失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update poll"})
			return
		}
		log.Printf("投票更新成功")
	} else {
		log.Printf("未检测到字段变更，跳过数据库更新")
	}

	// 处理选项更新
	if input.Options != nil && len(input.Options) > 0 {
		log.Printf("检测到选项更新请求, 选项数量: %d", len(input.Options))

		// 获取现有选项
		var existingOptions []models.PollOption
		if err := database.DB.Where("poll_id = ?", poll.ID).Find(&existingOptions).Error; err != nil {
			log.Printf("获取现有选项失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve existing options"})
			return
		}

		// 映射现有选项ID到对象，方便查找
		existingOptionsMap := make(map[uint]models.PollOption)
		for _, opt := range existingOptions {
			existingOptionsMap[opt.ID] = opt
		}

		// 跟踪前端提交的选项ID
		submittedOptionIDs := make(map[uint]bool)

		// 处理每个提交的选项
		for _, optInput := range input.Options {
			if optInput.ID > 0 {
				// 更新现有选项
				submittedOptionIDs[optInput.ID] = true

				if existingOpt, ok := existingOptionsMap[optInput.ID]; ok {
					log.Printf("更新选项 ID:%d, 文本: %s", optInput.ID, optInput.Text)
					existingOpt.Text = optInput.Text
					if err := database.DB.Save(&existingOpt).Error; err != nil {
						log.Printf("更新选项失败: %v", err)
						// 继续处理其他选项，不终止整个请求
					}
				} else {
					log.Printf("选项ID %d 不存在，跳过更新", optInput.ID)
				}
			} else {
				// 添加新选项
				log.Printf("添加新选项: %s", optInput.Text)
				newOption := models.PollOption{
					PollID: poll.ID,
					Text:   optInput.Text,
				}
				if err := database.DB.Create(&newOption).Error; err != nil {
					log.Printf("添加新选项失败: %v", err)
					// 继续处理其他选项，不终止整个请求
				}
			}
		}

		// 识别并删除未在提交列表中的现有选项（已被删除的选项）
		for optID := range existingOptionsMap {
			if !submittedOptionIDs[optID] {
				// 检查选项是否有投票
				var voteCount int64
				if err := database.DB.Model(&models.PollOption{}).Where("id = ?", optID).Select("votes").Scan(&voteCount).Error; err != nil {
					log.Printf("检查选项投票数失败: %v", err)
					continue
				}

				// 只删除没有投票的选项
				if voteCount == 0 {
					log.Printf("删除选项 ID:%d, 该选项不在提交列表中且没有投票", optID)
					if err := database.DB.Delete(&models.PollOption{}, optID).Error; err != nil {
						log.Printf("删除选项失败: %v", err)
					}
				} else {
					log.Printf("选项 ID:%d 有 %d 票，不会被删除", optID, voteCount)
				}
			}
		}
	}

	// Reload the poll to return the updated object with all options
	var updatedPoll models.Poll
	if err := database.DB.Preload("Options").First(&updatedPoll, poll.ID).Error; err != nil {
		log.Printf("重新加载更新后的投票失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reload updated poll"})
		return
	}

	log.Printf("返回更新后的投票: ID:%d, Question:%s, PollType:%d, IsActive:%v, 选项数量: %d",
		updatedPoll.ID, updatedPoll.Question, updatedPoll.PollType, updatedPoll.IsActive, len(updatedPoll.Options))
	c.JSON(http.StatusOK, updatedPoll)
}

// DeletePoll handles deleting a poll and its associated options
func DeletePoll(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid poll ID format"})
		return
	}

	// Use a transaction to ensure atomicity
	tx := database.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	// Delete associated options first (or set up cascading delete in DB)
	if err := tx.Where("poll_id = ?", uint(id)).Delete(&models.PollOption{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete poll options"})
		return
	}

	// Delete the poll
	result := tx.Delete(&models.Poll{}, uint(id))
	if result.Error != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete poll"})
		return
	}

	// Check if the poll was actually found and deleted
	if result.RowsAffected == 0 {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "Poll not found"})
		return
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Poll deleted successfully"})
}

// --- Vote Submission ---

// VoteInput defines the expected input structure for submitting a vote
type VoteInput struct {
	OptionID  uint   `json:"option_id,OptionID"`   // 单选选项ID，支持大小写
	OptionIDs []uint `json:"option_ids,OptionIDs"` // 多选选项IDs，支持大小写
}

// SubmitVote handles the submission of a vote on a poll option.
func SubmitVote(c *gin.Context) {
	pollIDStr := c.Param("id")
	pollID, err := strconv.ParseUint(pollIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的投票ID格式"})
		return
	}
	pollUintID := uint(pollID) // Convert to uint for GORM

	var input VoteInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 处理投票选项 - 支持多选
	optionIDs := []uint{}

	// 如果提供了option_id，添加到选项列表
	if input.OptionID > 0 {
		optionIDs = append(optionIDs, input.OptionID)
	}

	// 如果提供了option_ids，添加到选项列表
	if len(input.OptionIDs) > 0 {
		optionIDs = append(optionIDs, input.OptionIDs...)
	}

	// 去重
	uniqueOptionIDs := make([]uint, 0)
	optionIDMap := make(map[uint]bool)
	for _, id := range optionIDs {
		if id > 0 && !optionIDMap[id] {
			uniqueOptionIDs = append(uniqueOptionIDs, id)
			optionIDMap[id] = true
		}
	}

	// 验证是否有有效的选项ID
	if len(uniqueOptionIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "必须提供至少一个有效的选项ID"})
		return
	}

	// 获取Redis客户端并检查是否可用
	redisClient, err := cache.GetClient()
	redisAvailable := redisClient != nil && err == nil

	// 获取客户端IP，用于简单的游客防重复提交
	clientIP := c.ClientIP()

	log.Printf("收到来自 %s 的投票: 投票ID=%d, 选项=%v", clientIP, pollUintID, uniqueOptionIDs)

	// 使用Redis进行简单的重复提交检查
	if redisAvailable {
		ctx := context.Background()

		// 检查这个IP是否已经对此投票投过票
		pollIPLockKey := fmt.Sprintf("vote_lock:poll:%d:ip:%s", pollUintID, clientIP)
		success, err := redisClient.SetNX(ctx, pollIPLockKey, time.Now().Unix(), 24*time.Hour).Result()
		if err != nil {
			log.Printf("检查投票IP锁失败: %v", err)
			// 继续执行，但记录错误
		} else if !success {
			// 已经对此投票投过票
			log.Printf("检测到重复投票: IP %s 已经对投票 %d 投过票", clientIP, pollUintID)
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "您已经对此投票投过票，每个IP只能对一个投票投一次票"})
			return
		}
	}

	// 1. 获取投票详情并验证
	var poll models.Poll
	if err := database.DB.Preload("Options").First(&poll, pollUintID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "投票未找到"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取投票数据失败: " + err.Error()})
		}
		return
	}

	// 2. 验证投票是否活跃
	if !poll.IsActive {
		c.JSON(http.StatusForbidden, gin.H{"error": "此投票已关闭"})
		return
	}
	if poll.EndTime != nil && time.Now().After(*poll.EndTime) {
		c.JSON(http.StatusForbidden, gin.H{"error": "投票期已结束"})
		return
	}

	// 3. 验证选项是否有效且属于当前投票
	validOptionMap := make(map[uint]bool)
	for _, opt := range poll.Options {
		validOptionMap[opt.ID] = true
	}

	// 验证所有选项是否有效
	validOptionIDs := make([]uint, 0)
	for _, optID := range uniqueOptionIDs {
		if validOptionMap[optID] {
			validOptionIDs = append(validOptionIDs, optID)
		} else {
			log.Printf("警告: 选项ID %d 无效或不属于投票 %d", optID, pollUintID)
		}
	}

	if len(validOptionIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "所有提供的选项ID都无效"})
		return
	}

	// 对单选投票进行检查，确保只选择了一个选项
	if poll.PollType == models.SingleChoice && len(validOptionIDs) > 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "单选投票只能选择一个选项"})
		return
	}

	// 4. 高效投票处理：为每个选项增加票数
	for _, optionID := range validOptionIDs {
		updateSQL := "UPDATE poll_options SET votes = votes + 1 WHERE id = ? AND poll_id = ?"
		result := database.DB.Exec(updateSQL, optionID, pollUintID)

		if result.Error != nil {
			log.Printf("更新选项 %d 的投票数据失败: %v", optionID, result.Error)
			// 继续处理其他选项
		} else if result.RowsAffected == 0 {
			log.Printf("选项 %d 的投票更新失败: 可能已被删除", optionID)
			// 继续处理其他选项
		} else {
			log.Printf("成功为选项 %d 投票", optionID)
		}
	}

	// 5. 清理缓存，确保下次读取能获取最新数据
	if redisAvailable {
		ctx := context.Background()
		// 删除结果缓存
		cacheKey := fmt.Sprintf("poll:%d:results", pollUintID)
		if err := redisClient.Del(ctx, cacheKey).Err(); err != nil {
			log.Printf("删除缓存键失败: %s, 错误: %v", cacheKey, err)
		}
	}

	// 6. 获取更新后的结果
	var updatedOptions []models.PollOption
	if err := database.DB.Where("poll_id = ?", pollUintID).Find(&updatedOptions).Error; err != nil {
		log.Printf("获取更新后的选项失败: %v", err)
		c.JSON(http.StatusOK, gin.H{"message": "投票提交成功，但无法获取最新结果"})
		return
	}

	// 7. 计算结果百分比
	results := calculatePercentages(updatedOptions)

	// 8. 异步广播更新
	go func() {
		BroadcastPollUpdate(pollUintID, updatedOptions)
		BroadcastSSEUpdate(pollUintID, updatedOptions)
	}()

	c.JSON(http.StatusOK, gin.H{"message": "投票提交成功", "current_results": results})
}

// OptionResult 表示带有百分比的投票选项结果
type OptionResult struct {
	ID         uint    `json:"id"`
	Text       string  `json:"text"`
	Votes      int64   `json:"votes"`
	Percentage float64 `json:"percentage"`
}

// processPollVote 处理投票并确保数据一致性
func processPollVote(pollID uint, optionID uint, redisAvailable bool) ([]OptionResult, error) {
	// 步骤1: 先删除缓存
	if redisAvailable {
		redisClient, _ := cache.GetClient()
		pollIDStr := fmt.Sprint(pollID)
		ctx := context.Background()
		cacheKeys := []string{
			fmt.Sprintf("poll:%s:results", pollIDStr),
			fmt.Sprintf("poll:%s:data", pollIDStr),
			fmt.Sprintf("poll:%s:options", pollIDStr),
		}

		for _, key := range cacheKeys {
			if err := redisClient.Del(ctx, key).Err(); err != nil {
				log.Printf("删除缓存键失败 (第一次删除): %s, 错误: %v", key, err)
			} else {
				log.Printf("已删除缓存键 (第一次删除): %s", key)
			}
		}
	}

	// 步骤2: 获取投票详情并验证
	var poll models.Poll
	if err := database.DB.Preload("Options").First(&poll, pollID).Error; err != nil {
		return nil, fmt.Errorf("获取投票失败: %w", err)
	}

	// 验证投票是否活跃
	if !poll.IsActive {
		return nil, fmt.Errorf("此投票已关闭")
	}
	if poll.EndTime != nil && time.Now().After(*poll.EndTime) {
		return nil, fmt.Errorf("投票期已结束")
	}

	// 验证选项是否有效
	validOption := false
	for _, opt := range poll.Options {
		if opt.ID == optionID {
			validOption = true
			break
		}
	}
	if !validOption {
		return nil, fmt.Errorf("无效的选项ID: %d", optionID)
	}

	// 步骤3: 在高事务隔离级别下更新数据库
	var updatedOptions []models.PollOption

	// 先设置全局事务隔离级别
	if err := database.DB.Exec("SET SESSION TRANSACTION ISOLATION LEVEL SERIALIZABLE").Error; err != nil {
		return nil, fmt.Errorf("设置会话事务隔离级别失败: %w", err)
	}

	// 使用事务包装所有数据库操作
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// 锁定要更新的选项行
		var option models.PollOption
		if err := tx.Set("gorm:query_option", "FOR UPDATE").
			Where("id = ? AND poll_id = ?", optionID, pollID).
			First(&option).Error; err != nil {
			return fmt.Errorf("锁定选项失败: %w", err)
		}

		// 更新票数
		if err := tx.Model(&option).
			Where("id = ? AND poll_id = ?", optionID, pollID).
			Update("votes", gorm.Expr("votes + ?", 1)).Error; err != nil {
			return fmt.Errorf("更新票数失败: %w", err)
		}

		// 获取所有更新后的选项
		if err := tx.Where("poll_id = ?", pollID).Find(&updatedOptions).Error; err != nil {
			return fmt.Errorf("获取更新后的选项失败: %w", err)
		}

		return nil
	})

	// 恢复默认的事务隔离级别
	database.DB.Exec("SET SESSION TRANSACTION ISOLATION LEVEL REPEATABLE READ")

	if err != nil {
		return nil, fmt.Errorf("投票事务失败: %w", err)
	}

	// 步骤4: 适当延迟后删除缓存
	time.Sleep(10 * time.Millisecond)

	// 再次删除缓存
	if redisAvailable {
		redisClient, _ := cache.GetClient()
		pollIDStr := fmt.Sprint(pollID)
		ctx := context.Background()
		cacheKeys := []string{
			fmt.Sprintf("poll:%s:results", pollIDStr),
			fmt.Sprintf("poll:%s:data", pollIDStr),
			fmt.Sprintf("poll:%s:options", pollIDStr),
		}

		for _, key := range cacheKeys {
			if err := redisClient.Del(ctx, key).Err(); err != nil {
				log.Printf("删除缓存键失败 (第二次删除): %s, 错误: %v", key, err)
			} else {
				log.Printf("已删除缓存键 (第二次删除): %s", key)
			}
		}

		// 更新Redis缓存
		ctx = context.Background()
		for _, option := range updatedOptions {
			cacheKey := fmt.Sprintf("poll:%s:votes:%d", pollIDStr, option.ID)
			if err := redisClient.Set(ctx, cacheKey, option.Votes, 1*time.Hour).Err(); err != nil {
				log.Printf("更新Redis缓存失败: %s, 错误: %v", cacheKey, err)
			}
		}
	}

	// 步骤5: 构建带有百分比的结果
	results := calculatePercentages(updatedOptions)

	go func() {
		BroadcastPollUpdate(pollID, updatedOptions)
		BroadcastSSEUpdate(pollID, updatedOptions)
	}()

	return results, nil
}

// calculatePercentages 计算选项的百分比
func calculatePercentages(options []models.PollOption) []OptionResult {
	// 计算总票数
	var totalVotes int64 = 0
	for _, option := range options {
		totalVotes += option.Votes
	}

	// 转换为带百分比的结果
	results := make([]OptionResult, len(options))
	for i, option := range options {
		results[i] = OptionResult{
			ID:    option.ID,
			Text:  option.Text,
			Votes: option.Votes,
		}

		if totalVotes > 0 {
			results[i].Percentage = float64(option.Votes) / float64(totalVotes) * 100
		} else {
			results[i].Percentage = 0
		}
	}

	return results
}

// min返回两个int之间的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// buildResultsFromCache 从Redis缓存数据构建投票结果
func buildResultsFromCache(counts map[string]int64, pollOptions []models.PollOption) []PollOptionResult {
	// 创建optionID到Option的映射
	optionMap := make(map[string]models.PollOption)
	for _, opt := range pollOptions {
		optionMap[strconv.FormatUint(uint64(opt.ID), 10)] = opt
	}

	// 计算总票数
	var totalVotes int64 = 0
	for _, count := range counts {
		totalVotes += count
	}

	// 构建结果
	results := make([]PollOptionResult, 0, len(counts))
	for optionID, count := range counts {
		// 尝试获取选项对象
		option, exists := optionMap[optionID]
		if !exists {
			// 如果不存在，可能是Redis中有旧数据，跳过
			continue
		}

		// 计算百分比
		percentage := 0.0
		if totalVotes > 0 {
			percentage = float64(count) / float64(totalVotes) * 100
		}

		// 创建结果对象
		optionIDUint, _ := strconv.ParseUint(optionID, 10, 32)
		result := PollOptionResult{
			ID:         uint(optionIDUint),
			Text:       option.Text,
			Votes:      count,
			Percentage: percentage,
		}

		results = append(results, result)
	}

	return results
}

// PollOptionResult defines the structure for returning poll results
type PollOptionResult struct {
	ID         uint    `json:"id"`
	Text       string  `json:"text"`
	Votes      int64   `json:"votes"`
	Percentage float64 `json:"percentage"`
}

// GetCurrentPollResults fetches and calculates the current results for a poll
func GetCurrentPollResults(pollID uint) ([]PollOptionResult, error) {
	// 使用简单的SQL语句查询
	var options []struct {
		ID    uint   `json:"id"`
		Text  string `json:"text"`
		Votes int64  `json:"votes"`
	}

	sql := "SELECT id, text, votes FROM poll_options WHERE poll_id = ? AND deleted_at IS NULL"
	log.Printf("执行SQL: %s [参数: %d]", sql, pollID)

	err := database.DB.Raw(sql, pollID).Scan(&options).Error
	if err != nil {
		log.Printf("获取投票选项失败: %v", err)
		return nil, err
	}

	log.Printf("成功获取投票ID %d 的选项，共 %d 个选项", pollID, len(options))

	var totalVotes int64 = 0
	for _, opt := range options {
		totalVotes += opt.Votes
	}

	results := make([]PollOptionResult, len(options))
	for i, opt := range options {
		percentage := 0.0
		if totalVotes > 0 {
			percentage = (float64(opt.Votes) / float64(totalVotes)) * 100
		}
		results[i] = PollOptionResult{
			ID:         opt.ID,
			Text:       opt.Text,
			Votes:      opt.Votes,
			Percentage: percentage,
		}
	}
	return results, nil
}

// CheckAndCloseExpiredPolls 查找并关闭所有已过期的投票
func CheckAndCloseExpiredPolls() {
	now := time.Now()
	log.Printf("检查过期投票，当前时间: %v", now)

	// 查找状态为活跃但已过期的投票
	result := database.DB.Model(&models.Poll{}).
		Where("is_active = ? AND end_time IS NOT NULL AND end_time < ?", true, now).
		Update("is_active", false)

	if result.Error != nil {
		log.Printf("更新过期投票状态失败: %v", result.Error)
		return
	}

	if result.RowsAffected > 0 {
		log.Printf("已关闭 %d 个过期投票", result.RowsAffected)
	}
}

// --- WebSocket Handler will go here ---

// EnhancedVoteInput 定义带消息ID的投票输入结构
type EnhancedVoteInput struct {
	OptionIDs []uint `json:"option_ids" binding:"required,min=1"` // 选择的选项ID数组
	MessageID string `json:"message_id,omitempty"`                // 可选的消息ID，用于幂等性控制
}

// SubmitEnhancedVote 处理带有幂等性保证的投票提交
func SubmitEnhancedVote(c *gin.Context) {
	pollIDStr := c.Param("id")
	pollID, err := strconv.ParseUint(pollIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的投票ID格式"})
		return
	}
	pollUintID := uint(pollID) // 转换为GORM使用的uint类型

	var input EnhancedVoteInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. 获取投票及其选项
	var poll models.Poll
	if err := database.DB.Preload("Options").First(&poll, pollUintID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "投票未找到"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "无法获取投票数据"})
		}
		return
	}

	// 2. 检查投票是否活跃
	if !poll.IsActive {
		c.JSON(http.StatusForbidden, gin.H{"error": "此投票已关闭"})
		return
	}

	// 检查结束时间
	if poll.EndTime != nil && time.Now().After(*poll.EndTime) {
		c.JSON(http.StatusForbidden, gin.H{"error": "投票期已结束"})
		return
	}

	// 3. 验证提交的OptionIDs
	if len(input.OptionIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "必须至少选择一个选项"})
		return
	}

	validOptionIDs := make(map[uint]bool)
	for _, opt := range poll.Options {
		validOptionIDs[opt.ID] = true
	}

	for _, submittedID := range input.OptionIDs {
		if !validOptionIDs[submittedID] {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("提交了无效的选项ID %d", submittedID)})
			return
		}
	}

	// 4. 检查投票类型约束
	if poll.PollType == models.SingleChoice && len(input.OptionIDs) > 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "单选投票只允许选择一个选项"})
		return
	}

	// 获取客户端IP地址用于防重复提交
	clientIP := c.ClientIP()
	log.Printf("收到来自 %s 的投票: 投票ID=%d, 选项=%v", clientIP, pollUintID, input.OptionIDs)

	// 获取Redis客户端
	redisClient, err := cache.GetClient()
	redisAvailable := redisClient != nil && err == nil

	// 使用消息ID确保幂等性
	messageID := input.MessageID
	if messageID == "" {
		// 如果没有提供消息ID，生成一个
		messageID = fmt.Sprintf("client_msg_%d_%d", time.Now().UnixNano(), len(input.OptionIDs))
	}

	// 实现延迟双删策略
	// 步骤1: 先删除缓存
	if redisAvailable {
		// 删除相关的缓存键
		ctx := context.Background()
		cacheKeys := []string{
			fmt.Sprintf("poll:%d:results", pollID),
			fmt.Sprintf("poll:%d:data", pollID),
			fmt.Sprintf("poll:%d:options", pollID),
		}

		for _, key := range cacheKeys {
			if err := redisClient.Del(ctx, key).Err(); err != nil {
				log.Printf("删除缓存键失败 (第一次删除): %s, 错误: %v", key, err)
			} else {
				log.Printf("已删除缓存键 (第一次删除): %s", key)
			}
		}
	}

	// 步骤2: 更新数据库
	tx := database.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法开始事务"})
		return
	}

	for _, optionID := range input.OptionIDs {
		// 使用gorm.Expr进行原子增加
		result := tx.Model(&models.PollOption{}).Where("id = ? AND poll_id = ?", optionID, pollUintID).
			UpdateColumn("votes", gorm.Expr("votes + ?", 1))

		if result.Error != nil {
			tx.Rollback()
			log.Printf("更新选项 %d 票数失败: %v", optionID, result.Error)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "记录投票失败"})
			return
		}
		if result.RowsAffected == 0 {
			tx.Rollback()
			log.Printf("投票更新失败: 选项ID %d 未找到或不属于投票 %d", optionID, pollUintID)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "由于状态不一致，记录投票失败"})
			return
		}
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		log.Printf("提交投票事务失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "完成投票失败"})
		return
	}

	// 步骤3: 等待一段时间（允许其他可能的读操作完成）
	time.Sleep(10 * time.Millisecond)

	// 步骤4: 再次删除缓存
	if redisAvailable {
		ctx := context.Background()
		cacheKeys := []string{
			fmt.Sprintf("poll:%d:results", pollID),
			fmt.Sprintf("poll:%d:data", pollID),
			fmt.Sprintf("poll:%d:options", pollID),
		}

		for _, key := range cacheKeys {
			if err := redisClient.Del(ctx, key).Err(); err != nil {
				log.Printf("删除缓存键失败 (第二次删除): %s, 错误: %v", key, err)
			} else {
				log.Printf("已删除缓存键 (第二次删除): %s", key)
			}
		}
	}

	// 获取更新后的结果
	updatedResults, err := GetCurrentPollResults(pollUintID)
	if err != nil {
		log.Printf("获取更新后的投票结果失败: %v", err)
		c.JSON(http.StatusOK, gin.H{"message": "投票提交成功但无法获取最新结果"})
		return
	}

	// 如果Redis可用，更新Redis中的计数
	if redisAvailable {
		ctx := context.Background()
		// 将最新数据写入缓存
		for _, option := range updatedResults {
			// 设置投票计数
			cacheKey := fmt.Sprintf("poll:%s:votes:%d", pollIDStr, option.ID)
			if err := redisClient.Set(ctx, cacheKey, option.Votes, 1*time.Hour).Err(); err != nil {
				log.Printf("更新Redis缓存失败: %s, 错误: %v", cacheKey, err)
			}
		}

		// 写入总结果缓存，提高后续查询效率
		resultsCacheKey := fmt.Sprintf("poll:%s:results", pollIDStr)
		if resultsData, err := json.Marshal(updatedResults); err == nil {
			if err := redisClient.Set(ctx, resultsCacheKey, resultsData, 1*time.Hour).Err(); err != nil {
				log.Printf("缓存投票结果失败: %s, 错误: %v", resultsCacheKey, err)
			} else {
				log.Printf("已缓存投票结果: %s", resultsCacheKey)
			}
		}
	}

	// 异步广播更新，但增加稳定性和防丢失机制
	go func() {
		// 延迟一小段时间再广播，确保数据库事务完全提交
		time.Sleep(50 * time.Millisecond)

		// 在广播前再次验证数据正确性
		verifiedResults, err := GetCurrentPollResults(pollUintID)
		if err != nil {
			log.Printf("广播前验证结果失败，使用之前的结果: %v", err)
			verifiedResults = updatedResults
		}

		// 广播消息重试和备份机制
		maxBroadcastRetries := 3
		for i := 0; i < maxBroadcastRetries; i++ {
			if i > 0 {
				time.Sleep(time.Duration(100*i) * time.Millisecond)
				log.Printf("重试广播投票结果 (尝试 %d/%d)：Poll ID=%d", i+1, maxBroadcastRetries, pollUintID)
			}

			BroadcastPollUpdate(pollUintID, verifiedResults)

			// 不管WebSocket是否成功，也通过SSE发送更新消息
			BroadcastSSEUpdate(pollUintID, verifiedResults)

			// 在高并发测试时（例如超过20请求/秒）不要马上退出，确保广播完成
			if i == 0 {
				time.Sleep(30 * time.Millisecond)
			}
		}
	}()

	c.JSON(http.StatusOK, gin.H{"message": "投票提交成功", "current_results": updatedResults})
}

// ResetPollVotesInput 定义重置投票的输入结构
type ResetPollVotesInput struct {
	AdminKey string `json:"admin_key" binding:"required"`
}

// ResetPollVotes 重置投票计数
func ResetPollVotes(c *gin.Context) {
	// 获取投票ID
	pollIDStr := c.Param("id")
	pollID, err := strconv.ParseUint(pollIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的投票ID格式"})
		return
	}
	pollUintID := uint(pollID)

	// 检查请求体，但允许空请求
	var input ResetPollVotesInput
	if err := c.ShouldBindJSON(&input); err != nil {
		// 如果是EOF错误，说明请求体为空，我们忽略它并继续
		if err.Error() != "EOF" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入: " + err.Error()})
			return
		}
		// 使用默认管理员密钥
		input.AdminKey = "admin"
		log.Printf("重置投票请求没有提供管理员密钥，使用默认值")
	}

	// 检查管理员密钥 (简单实现，实际应用中应使用更安全的方式)
	if input.AdminKey != "admin" && input.AdminKey != "admin123" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的管理员密钥"})
		return
	}

	// 先从Redis缓存中删除所有与此投票相关的键
	redisClient, err := cache.GetClient()
	if err == nil && redisClient != nil {
		// 删除所有可能的缓存键模式
		cacheKeyPatterns := []string{
			fmt.Sprintf("poll:%d:*", pollID),
			fmt.Sprintf("vote_lock:*:%d", pollID),
			fmt.Sprintf("poll_data:%d", pollID),
			fmt.Sprintf("poll_results:%d", pollID),
		}

		ctx := context.Background()
		for _, pattern := range cacheKeyPatterns {
			keys, err := redisClient.Keys(ctx, pattern).Result()
			if err != nil {
				log.Printf("获取键模式 %s 失败: %v", pattern, err)
				continue
			}

			if len(keys) > 0 {
				// 批量删除所有匹配的键
				if err := redisClient.Del(ctx, keys...).Err(); err != nil {
					log.Printf("删除缓存键 %v 失败: %v", keys, err)
				} else {
					log.Printf("已从Redis删除 %d 个键，模式: %s", len(keys), pattern)
				}
			}
		}

		// 额外检查单票投票的键
		for _, opt := range []string{"votes", "percentage", "data", "options", "results", "voted"} {
			pattern := fmt.Sprintf("poll:%s:%s:*", pollIDStr, opt)
			keys, err := redisClient.Keys(ctx, pattern).Result()
			if err != nil {
				log.Printf("获取键模式 %s 失败: %v", pattern, err)
				continue
			}

			if len(keys) > 0 {
				if err := redisClient.Del(ctx, keys...).Err(); err != nil {
					log.Printf("删除缓存键 %v 失败: %v", keys, err)
				} else {
					log.Printf("已从Redis删除 %d 个键，模式: %s", len(keys), pattern)
				}
			}
		}
	}

	// 在数据库中重置投票计数
	tx := database.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法开始事务"})
		return
	}

	// 更新所有选项的投票计数为0
	if err := tx.Model(&models.PollOption{}).Where("poll_id = ?", pollUintID).
		UpdateColumn("votes", 0).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "重置投票失败: " + err.Error()})
		return
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "提交事务失败: " + err.Error()})
		return
	}

	// 获取重置后的投票结果
	updatedResults, err := GetCurrentPollResults(pollUintID)
	if err != nil {
		log.Printf("获取重置后的投票结果失败: %v", err)
	} else {
		log.Printf("投票已重置: 投票ID=%d，选项数=%d", pollUintID, len(updatedResults))
	}

	// 使用相同的广播机制通知所有客户端
	go func() {
		// 延迟一小段时间，确保数据库更新完全提交和客户端准备好接收
		time.Sleep(100 * time.Millisecond)

		// 重置后立即重试多次广播，确保所有客户端收到更新
		for i := 0; i < 3; i++ {
			BroadcastPollUpdate(pollUintID, updatedResults)
			BroadcastSSEUpdate(pollUintID, updatedResults)

			// 测试模式下等待更长时间，确保完全更新
			waitTime := 150 * time.Millisecond
			if i < 2 {
				time.Sleep(waitTime)
			}
		}

		// 在高并发测试结束后，确保前端有足够时间更新
		// 这对于测试特别重要，让所有客户端能看到最终状态
		time.Sleep(500 * time.Millisecond)

		log.Printf("投票重置广播完成: 投票ID=%d", pollUintID)
	}()

	c.JSON(http.StatusOK, gin.H{
		"message": "投票已成功重置",
		"poll_id": pollID,
	})
}
