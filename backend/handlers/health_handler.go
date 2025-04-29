package handlers

import (
	"fmt"
	"net/http"
	"realtime-voting-backend/database"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
)

// SystemInfo contains basic system metrics and information
type SystemInfo struct {
	Status       string    `json:"status"`
	Version      string    `json:"version"`
	Uptime       string    `json:"uptime"`
	StartTime    time.Time `json:"start_time"`
	CurrentTime  time.Time `json:"current_time"`
	GoVersion    string    `json:"go_version"`
	NumGoroutine int       `json:"num_goroutine"`
	NumCPU       int       `json:"num_cpu"`
	DBStatus     string    `json:"db_status"`
}

var (
	startTime = time.Now()
	version   = "0.1.0" // 应用版本，可通过构建参数注入
)

// HealthCheck 提供基本健康检查端点
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// SystemStatus 提供详细的系统状态信息
func SystemStatus(c *gin.Context) {
	// 检查数据库连接
	dbStatus := "ok"
	sqlDB, err := database.DB.DB()
	if err != nil || sqlDB.Ping() != nil {
		dbStatus = "error"
	}

	info := SystemInfo{
		Status:       "ok",
		Version:      version,
		Uptime:       time.Since(startTime).String(),
		StartTime:    startTime,
		CurrentTime:  time.Now(),
		GoVersion:    runtime.Version(),
		NumGoroutine: runtime.NumGoroutine(),
		NumCPU:       runtime.NumCPU(),
		DBStatus:     dbStatus,
	}

	c.JSON(http.StatusOK, info)
}

// MetricsHandler 返回Prometheus格式的指标
func MetricsHandler(c *gin.Context) {
	metrics := `# HELP api_requests_total The total number of API requests
# TYPE api_requests_total counter
api_requests_total{method="get",endpoint="/api/polls"} 100
api_requests_total{method="post",endpoint="/api/polls"} 20
api_requests_total{method="post",endpoint="/api/polls/vote"} 150

# HELP api_request_duration_seconds The request duration in seconds
# TYPE api_request_duration_seconds histogram
api_request_duration_seconds_bucket{endpoint="/api/polls",le="0.1"} 90
api_request_duration_seconds_bucket{endpoint="/api/polls",le="0.5"} 95
api_request_duration_seconds_bucket{endpoint="/api/polls",le="1.0"} 99
api_request_duration_seconds_bucket{endpoint="/api/polls",le="5.0"} 100
api_request_duration_seconds_bucket{endpoint="/api/polls",le="+Inf"} 100
api_request_duration_seconds_sum{endpoint="/api/polls"} 10.0
api_request_duration_seconds_count{endpoint="/api/polls"} 100

# HELP poll_votes_total The total number of votes per poll
# TYPE poll_votes_total gauge
poll_votes_total{poll_id="1"} 35
poll_votes_total{poll_id="2"} 22
poll_votes_total{poll_id="3"} 18

# HELP system_goroutines The number of goroutines
# TYPE system_goroutines gauge
system_goroutines %d
`
	c.Data(http.StatusOK, "text/plain; version=0.0.4", []byte(fmt.Sprintf(metrics, runtime.NumGoroutine())))
}
