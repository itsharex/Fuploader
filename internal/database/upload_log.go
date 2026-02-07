package database

import (
	"time"

	"gorm.io/gorm"
)

// UploadLog 上传日志
type UploadLog struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	TaskID     uint           `gorm:"index" json:"task_id"`
	AccountID  uint           `gorm:"index" json:"account_id"`
	Platform   string         `gorm:"index;size:50" json:"platform"`
	VideoPath  string         `gorm:"size:500" json:"video_path"`
	Step       string         `gorm:"size:50" json:"step"`         // 当前步骤
	Status     string         `gorm:"size:20;index" json:"status"` // success/failed/processing
	Message    string         `gorm:"size:2000" json:"message"`    // 详细信息
	Duration   int64          `json:"duration"`                    // 耗时（毫秒）
	Screenshot string         `gorm:"size:500" json:"screenshot"`  // 失败时的截图路径
	ErrorCode  string         `gorm:"size:50" json:"error_code"`   // 错误代码
	RetryCount int            `json:"retry_count"`                 // 重试次数
	IPAddress  string         `gorm:"size:50" json:"ip_address"`   // IP地址（用于排查）
	UserAgent  string         `gorm:"size:500" json:"user_agent"`  // User-Agent
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

// UploadLogQuery 上传日志查询条件
type UploadLogQuery struct {
	TaskID    uint
	AccountID uint
	Platform  string
	Status    string
	Step      string
	StartTime *time.Time
	EndTime   *time.Time
	Limit     int
	Offset    int
}

// UploadLogService 上传日志服务
type UploadLogService struct {
	db *gorm.DB
}

// NewUploadLogService 创建上传日志服务
func NewUploadLogService(db *gorm.DB) *UploadLogService {
	return &UploadLogService{db: db}
}

// Create 创建上传日志
func (s *UploadLogService) Create(log *UploadLog) error {
	return s.db.Create(log).Error
}

// GetByID 根据ID获取日志
func (s *UploadLogService) GetByID(id uint) (*UploadLog, error) {
	var log UploadLog
	if err := s.db.First(&log, id).Error; err != nil {
		return nil, err
	}
	return &log, nil
}

// Query 查询日志
func (s *UploadLogService) Query(query UploadLogQuery) ([]UploadLog, int64, error) {
	var logs []UploadLog
	var total int64

	db := s.db.Model(&UploadLog{})

	if query.TaskID > 0 {
		db = db.Where("task_id = ?", query.TaskID)
	}
	if query.AccountID > 0 {
		db = db.Where("account_id = ?", query.AccountID)
	}
	if query.Platform != "" {
		db = db.Where("platform = ?", query.Platform)
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.Step != "" {
		db = db.Where("step = ?", query.Step)
	}
	if query.StartTime != nil {
		db = db.Where("created_at >= ?", query.StartTime)
	}
	if query.EndTime != nil {
		db = db.Where("created_at <= ?", query.EndTime)
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	if query.Limit > 0 {
		db = db.Limit(query.Limit)
	}
	if query.Offset > 0 {
		db = db.Offset(query.Offset)
	}

	// 按时间倒序
	db = db.Order("created_at DESC")

	if err := db.Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// GetLatestByTask 获取任务的最新日志
func (s *UploadLogService) GetLatestByTask(taskID uint) (*UploadLog, error) {
	var log UploadLog
	if err := s.db.Where("task_id = ?", taskID).Order("created_at DESC").First(&log).Error; err != nil {
		return nil, err
	}
	return &log, nil
}

// GetTaskLogs 获取任务的所有日志
func (s *UploadLogService) GetTaskLogs(taskID uint) ([]UploadLog, error) {
	var logs []UploadLog
	if err := s.db.Where("task_id = ?", taskID).Order("created_at ASC").Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

// GetPlatformStats 获取平台统计
func (s *UploadLogService) GetPlatformStats(platform string, startTime, endTime time.Time) (map[string]interface{}, error) {
	var totalCount, successCount, failedCount int64

	db := s.db.Model(&UploadLog{}).Where("platform = ?", platform)
	if !startTime.IsZero() {
		db = db.Where("created_at >= ?", startTime)
	}
	if !endTime.IsZero() {
		db = db.Where("created_at <= ?", endTime)
	}

	// 总数
	if err := db.Count(&totalCount).Error; err != nil {
		return nil, err
	}

	// 成功数
	if err := db.Where("status = ?", "success").Count(&successCount).Error; err != nil {
		return nil, err
	}

	// 失败数
	if err := db.Where("status = ?", "failed").Count(&failedCount).Error; err != nil {
		return nil, err
	}

	// 平均耗时
	var avgDuration float64
	s.db.Model(&UploadLog{}).
		Where("platform = ? AND status = ?", platform, "success").
		Select("AVG(duration)").
		Row().Scan(&avgDuration)

	return map[string]interface{}{
		"platform":     platform,
		"total":        totalCount,
		"success":      successCount,
		"failed":       failedCount,
		"success_rate": float64(successCount) / float64(totalCount) * 100,
		"avg_duration": avgDuration,
	}, nil
}

// GetStepStats 获取步骤统计
func (s *UploadLogService) GetStepStats(platform string, startTime, endTime time.Time) ([]map[string]interface{}, error) {
	type StepStat struct {
		Step        string
		Count       int64
		AvgDuration float64
	}

	var stats []StepStat

	db := s.db.Model(&UploadLog{}).
		Select("step, COUNT(*) as count, AVG(duration) as avg_duration").
		Where("platform = ?", platform).
		Group("step")

	if !startTime.IsZero() {
		db = db.Where("created_at >= ?", startTime)
	}
	if !endTime.IsZero() {
		db = db.Where("created_at <= ?", endTime)
	}

	if err := db.Scan(&stats).Error; err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, len(stats))
	for i, stat := range stats {
		result[i] = map[string]interface{}{
			"step":         stat.Step,
			"count":        stat.Count,
			"avg_duration": stat.AvgDuration,
		}
	}

	return result, nil
}

// CleanOldLogs 清理旧日志
func (s *UploadLogService) CleanOldLogs(before time.Time) (int64, error) {
	result := s.db.Where("created_at < ?", before).Delete(&UploadLog{})
	return result.RowsAffected, result.Error
}

// UploadLogWriter 日志写入器
type UploadLogWriter struct {
	service *UploadLogService
	taskID  uint
}

// NewUploadLogWriter 创建日志写入器
func NewUploadLogWriter(service *UploadLogService, taskID uint) *UploadLogWriter {
	return &UploadLogWriter{
		service: service,
		taskID:  taskID,
	}
}

// LogStep 记录步骤
func (w *UploadLogWriter) LogStep(accountID uint, platform, step, status, message string, duration int64) error {
	log := &UploadLog{
		TaskID:    w.taskID,
		AccountID: accountID,
		Platform:  platform,
		Step:      step,
		Status:    status,
		Message:   message,
		Duration:  duration,
	}
	return w.service.Create(log)
}

// LogSuccess 记录成功
func (w *UploadLogWriter) LogSuccess(accountID uint, platform, step, message string, duration int64) error {
	return w.LogStep(accountID, platform, step, "success", message, duration)
}

// LogFailed 记录失败
func (w *UploadLogWriter) LogFailed(accountID uint, platform, step, message, errorCode string, duration int64) error {
	log := &UploadLog{
		TaskID:    w.taskID,
		AccountID: accountID,
		Platform:  platform,
		Step:      step,
		Status:    "failed",
		Message:   message,
		ErrorCode: errorCode,
		Duration:  duration,
	}
	return w.service.Create(log)
}

// LogProcessing 记录处理中
func (w *UploadLogWriter) LogProcessing(accountID uint, platform, step, message string) error {
	return w.LogStep(accountID, platform, step, "processing", message, 0)
}
