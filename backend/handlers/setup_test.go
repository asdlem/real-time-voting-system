package handlers

import (
	"log"
	"realtime-voting-backend/database"
	"realtime-voting-backend/models"
	"testing"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// SetupTestEnvironment sets up the Gin router and in-memory SQLite database for testing.
func SetupTestEnvironment(t *testing.T) (*gin.Engine, *gorm.DB) {
	testing.Init()
	gin.SetMode(gin.TestMode)

	// Use in-memory SQLite for testing
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		// Silence GORM logger for tests unless needed
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("Failed to connect to in-memory database: %v", err)
	}

	// Assign the test database to the global DB variable (or pass it around)
	// Be cautious with global variables in tests, ensure proper cleanup.
	// A better approach might involve dependency injection.
	database.DB = db

	// Migrate the schema
	err = database.DB.AutoMigrate(&models.Poll{}, &models.PollOption{})
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Clean up function to close DB connection after tests
	t.Cleanup(func() {
		sqlDB, _ := database.DB.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
		// If you need to clear tables between tests, do it here or in test setup/teardown
	})

	// Setup Router
	router := gin.Default()
	// CORS Middleware (might not be strictly needed for unit tests but good for consistency)
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"*"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	router.Use(cors.New(config))

	// Setup Routes (same as in main.go)
	api := router.Group("/api")
	{
		api.POST("/polls", CreatePoll)
		api.GET("/polls", GetPolls)
		api.GET("/polls/:id", GetPoll)
		api.PUT("/polls/:id", UpdatePoll)
		api.DELETE("/polls/:id", DeletePoll)
		api.POST("/polls/:id/vote", SubmitVote)
		// WebSocket testing is more complex and often done via integration tests
		// api.GET("/ws/polls/:id", HandleWebSocket)
	}

	return router, db
}

// Helper function to clear tables between tests if needed
func ClearTables(db *gorm.DB) {
	// Order matters due to foreign key constraints
	db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.PollOption{})
	db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.Poll{})
}
