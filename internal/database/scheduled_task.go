package database

import (
	"time"
)

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusScheduled TaskStatus = "scheduled"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

// TaskPriority 任务优先级
type TaskPriority int

const (
	PriorityLow    TaskPriority = 1
	PriorityNormal TaskPriority = 5
	PriorityHigh   TaskPriority = 10
)

// ScheduledTask 定时任务
type ScheduledTask struct {
	ID           string       `json:"id" gorm:"primaryKey"`
	AccountID    uint         `json:"account_id" gorm:"index"`
	Platform     string       `json:"platform" gorm:"index"`
	VideoPath    string       `json:"video_path"`
	Title        string       `json:"title"`
	Description  string       `json:"description"`
	Tags         string       `json:"tags"`
	ScheduleTime time.Time    `json:"schedule_time" gorm:"index"`
	Status       TaskStatus   `json:"status" gorm:"index"`
	Priority     TaskPriority `json:"priority"`
	RetryCount   int          `json:"retry_count"`
	MaxRetries   int          `json:"max_retries"`
	Error        string       `json:"error"`
	Result       string       `json:"result" gorm:"type:text"`
	ExecutedAt   *time.Time   `json:"executed_at"`
	CompletedAt  *time.Time   `json:"completed_at"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

// TableName 指定表名
func (ScheduledTask) TableName() string {
	return "scheduled_tasks"
}
