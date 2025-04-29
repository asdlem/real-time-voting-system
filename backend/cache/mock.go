package cache

import (
	"sync"
)

// 模拟模式相关变量
var (
	mockMode  bool
	mockMutex sync.Mutex
	mockData  = make(map[string]string)
	mockLocks = make(map[string]bool)
)

// 初始化mock数据
func init() {
	mockData = make(map[string]string)
	mockLocks = make(map[string]bool)
}
