package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"realtime-voting-backend/models"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB 是全局数据库连接
var DB *gorm.DB

// InitDB 初始化数据库连接
func InitDB() error {
	// 配置GORM
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second, // 慢SQL阈值
			LogLevel:                  logger.Info, // 日志级别
			IgnoreRecordNotFoundError: true,        // 忽略ErrRecordNotFound错误
			Colorful:                  true,        // 启用彩色打印
		},
	)

	var err error

	// 从环境变量获取MySQL数据库配置
	dbUser := getEnv("DB_USER", "voteuser")
	dbPassword := getEnv("DB_PASSWORD", "votepassword")
	dbHost := getEnv("DB_HOST", "mysql")
	dbPort := getEnv("DB_PORT", "3306")
	dbName := getEnv("DB_NAME", "votingdb")

	// 构建DSN
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	log.Println("使用MySQL数据库")
	// 连接MySQL数据库
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: newLogger,
	})

	if err != nil {
		return fmt.Errorf("连接数据库失败: %v", err)
	}

	// 自动迁移模型
	if err := DB.AutoMigrate(&models.Poll{}, &models.PollOption{}); err != nil {
		return fmt.Errorf("迁移模型失败: %v", err)
	}

	// 添加一些示例数据（仅在开发模式下）
	if getEnv("ENVIRONMENT", "development") == "development" {
		createSampleData()
	}

	log.Println("数据库连接和迁移成功")
	return nil
}

// createSampleData 创建示例数据
func createSampleData() {
	// 检查是否已有数据
	var count int64
	DB.Model(&models.Poll{}).Count(&count)
	if count > 0 {
		log.Println("数据库已有数据，跳过示例数据创建")
		return
	}

	log.Println("创建示例数据...")

	// 创建示例投票
	endTime := time.Now().Add(7 * 24 * time.Hour) // 一周后结束
	poll := models.Poll{
		Question: "你最喜欢的编程语言是什么？",
		PollType: 0, // 单选
		IsActive: true,
		EndTime:  &endTime,
	}

	if err := DB.Create(&poll).Error; err != nil {
		log.Printf("创建示例投票失败: %v", err)
		return
	}

	// 创建选项
	options := []models.PollOption{
		{PollID: poll.ID, Text: "Go", Votes: 10},
		{PollID: poll.ID, Text: "Python", Votes: 8},
		{PollID: poll.ID, Text: "Java", Votes: 6},
		{PollID: poll.ID, Text: "JavaScript", Votes: 9},
		{PollID: poll.ID, Text: "C++", Votes: 4},
	}

	if err := DB.Create(&options).Error; err != nil {
		log.Printf("创建示例选项失败: %v", err)
		return
	}

	log.Println("示例数据创建成功")
}

// IncrementVote 增加选项的投票计数
func IncrementVote(pollIDStr string, optionIDStr string) error {
	// 将字符串ID转换为uint
	pollID, err := strconv.ParseUint(pollIDStr, 10, 32)
	if err != nil {
		return fmt.Errorf("解析投票ID失败: %v", err)
	}

	optionID, err := strconv.ParseUint(optionIDStr, 10, 32)
	if err != nil {
		return fmt.Errorf("解析选项ID失败: %v", err)
	}

	// 开始事务
	tx := DB.Begin()
	if tx.Error != nil {
		return fmt.Errorf("开始事务失败: %v", tx.Error)
	}

	// 验证投票和选项存在
	var poll models.Poll
	if err := tx.First(&poll, uint(pollID)).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("找不到投票: %v", err)
	}

	// 检查投票是否活跃
	if !poll.IsActive {
		tx.Rollback()
		return fmt.Errorf("投票已关闭")
	}

	// 验证选项存在
	var option models.PollOption
	if err := tx.Where("id = ? AND poll_id = ?", uint(optionID), uint(pollID)).First(&option).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("找不到选项: %v", err)
	}

	// 原子增加投票计数
	if err := tx.Model(&option).UpdateColumn("votes", gorm.Expr("votes + ?", 1)).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("更新投票计数失败: %v", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %v", err)
	}

	log.Printf("成功增加投票计数: Poll=%d, Option=%d", pollID, optionID)
	return nil
}

// GetPollResults 获取投票的当前结果
func GetPollResults(pollID uint) ([]map[string]interface{}, error) {
	var options []models.PollOption

	if err := DB.Where("poll_id = ?", pollID).Find(&options).Error; err != nil {
		return nil, fmt.Errorf("获取投票选项失败: %v", err)
	}

	// 计算总票数
	var totalVotes int64 = 0
	for _, opt := range options {
		totalVotes += int64(opt.Votes)
	}

	// 构建结果
	results := make([]map[string]interface{}, len(options))
	for i, opt := range options {
		percentage := 0.0
		if totalVotes > 0 {
			percentage = float64(opt.Votes) / float64(totalVotes) * 100
		}

		results[i] = map[string]interface{}{
			"id":         opt.ID,
			"text":       opt.Text,
			"votes":      opt.Votes,
			"percentage": percentage,
		}
	}

	return results, nil
}

// CloseDB 关闭数据库连接
func CloseDB() {
	sqlDB, err := DB.DB()
	if err != nil {
		log.Printf("获取数据库连接失败: %v", err)
		return
	}

	if err := sqlDB.Close(); err != nil {
		log.Printf("关闭数据库连接失败: %v", err)
		return
	}

	log.Println("数据库连接已关闭")
}

// getEnv 获取环境变量值或使用默认值
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// GetPoll 根据ID获取投票信息
func GetPoll(pollID string) (*Poll, error) {
	// 实现获取投票详情的逻辑
	log.Printf("获取投票信息: %s", pollID)

	// 这里应该查询数据库，但为了简化，我们返回一个模拟的投票
	poll := &Poll{
		ID:          pollID,
		Title:       "模拟投票",
		Description: "这是一个模拟的投票",
		CreatedAt:   time.Now(),
		Options:     []Option{},
	}

	// 添加一些选项
	poll.Options = append(poll.Options, Option{
		ID:      "1",
		Content: "选项1",
		PollID:  pollID,
	})

	poll.Options = append(poll.Options, Option{
		ID:      "2",
		Content: "选项2",
		PollID:  pollID,
	})

	return poll, nil
}

// AddVote 增加投票计数
func AddVote(ctx context.Context, pollID string, optionID string) error {
	// 实现添加投票的逻辑
	log.Printf("添加投票: 投票ID=%s, 选项ID=%s", pollID, optionID)

	// 这里应该更新数据库，但为了简化，我们只记录日志
	return nil
}
