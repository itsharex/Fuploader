package service

import (
	"Fuploader/internal/types"
	"strings"
	"sync"
	"time"
)

// LogService 日志服务
type LogService struct {
	logs  []types.SimpleLog
	mutex sync.RWMutex
	limit int // 最大保留日志条数
}

// NewLogService 创建日志服务
func NewLogService() *LogService {
	return &LogService{
		logs:  make([]types.SimpleLog, 0, 500),
		limit: 500,
	}
}

// Add 添加日志
func (s *LogService) Add(message string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := time.Now()
	log := types.SimpleLog{
		Date:    now.Format("2006/1/2"),
		Time:    now.Format("15:04:05"),
		Message: message,
	}

	s.logs = append(s.logs, log)

	// 超过限制时，移除最旧的日志
	if len(s.logs) > s.limit {
		s.logs = s.logs[len(s.logs)-s.limit:]
	}
}

// Query 查询日志
func (s *LogService) Query(query types.LogQuery) []types.SimpleLog {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	limit := query.Limit
	if limit <= 0 {
		limit = 100
	}

	result := make([]types.SimpleLog, 0, limit)

	// 倒序遍历，最新的在前面
	for i := len(s.logs) - 1; i >= 0 && len(result) < limit; i-- {
		log := s.logs[i]

		// 关键词筛选
		if query.Keyword != "" && !strings.Contains(log.Message, query.Keyword) {
			continue
		}

		result = append(result, log)
	}

	return result
}

// GetAll 获取所有日志
func (s *LogService) GetAll(limit int) []types.SimpleLog {
	if limit <= 0 {
		limit = 100
	}

	return s.Query(types.LogQuery{Limit: limit})
}

// Clear 清空日志
func (s *LogService) Clear() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.logs = make([]types.SimpleLog, 0, s.limit)
}

// Count 获取日志数量
func (s *LogService) Count() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return len(s.logs)
}
