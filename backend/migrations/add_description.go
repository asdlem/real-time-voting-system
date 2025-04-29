package migrations

import (
	"log"

	"gorm.io/gorm"
)

// AddDescriptionToPoll 为Poll表添加Description字段
func AddDescriptionToPoll(db *gorm.DB) error {
	log.Println("执行迁移: 为Poll表添加Description字段")

	// 检查字段是否已存在
	if !db.Migrator().HasColumn(&Poll{}, "description") {
		// 添加description字段
		err := db.Exec("ALTER TABLE polls ADD COLUMN description TEXT").Error
		if err != nil {
			log.Printf("迁移失败: %v", err)
			return err
		}
		log.Println("迁移成功: 已添加description字段")
	} else {
		log.Println("迁移跳过: description字段已存在")
	}

	return nil
}

// 定义一个简单的Poll结构体，仅用于检查字段
type Poll struct {
	Description string
}

// 确保Poll结构体实现了gorm.Tabler接口
func (Poll) TableName() string {
	return "polls"
}
