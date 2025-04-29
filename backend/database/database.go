// 旧版数据库连接代码，已弃用，使用db.go中的实现
package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 此变量已在db.go中重新定义
// var DB *gorm.DB

// ConnectDB 已被 InitDB 替代
func OldConnectDB() {
	var err error
	// Read DSN from environment variable
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		// Fallback to default for local testing if needed, but log a warning
		log.Println("Warning: DATABASE_URL environment variable not set. Using default DSN for local dev.")
		dsn = "user:password@tcp(127.0.0.1:3306)/realtime_voting?charset=utf8mb4&parseTime=True&loc=Local"
		// Or, alternatively, enforce setting the env var:
		// log.Fatal("DATABASE_URL environment variable is not set")
	}

	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second, // Slow SQL threshold
			LogLevel:                  logger.Info, // Log level
			IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
			ParameterizedQueries:      true,        // Don't include params in the SQL log
			Colorful:                  true,        // Enable color
		},
	)

	_, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: newLogger,
	})

	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	fmt.Println("Database connection successful")

	// Optional: Configure connection pool
	// sqlDB, err := DB.DB()
	// if err != nil {
	// 	log.Fatal("Failed to get underlying sql.DB:", err)
	// }
	// sqlDB.SetMaxIdleConns(10)
	// sqlDB.SetMaxOpenConns(100)
	// sqlDB.SetConnMaxLifetime(time.Hour)
}
