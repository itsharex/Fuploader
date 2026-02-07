package scheduler

import (
	"Fuploader/internal/config"
	"Fuploader/internal/database"
	"Fuploader/internal/utils"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	cron      *cron.Cron
	taskQueue chan int
	executor  TaskExecutor
	isRunning bool
}

type TaskExecutor interface {
	ExecuteTask(taskID int) error
}

func NewScheduler(executor TaskExecutor) *Scheduler {
	return &Scheduler{
		cron:      cron.New(),
		taskQueue: make(chan int, 100),
		executor:  executor,
	}
}

func (s *Scheduler) Start() {
	if s.isRunning {
		return
	}
	s.isRunning = true
	s.cron.Start()
	go s.processQueue()
	utils.Info("Scheduler started")
}

func (s *Scheduler) Stop() {
	if !s.isRunning {
		return
	}
	s.isRunning = false
	s.cron.Stop()
	close(s.taskQueue)
	utils.Info("Scheduler stopped")
}

func (s *Scheduler) ScheduleTask(taskID int, scheduleTime string) error {
	t, err := time.Parse(time.RFC3339, scheduleTime)
	if err != nil {
		return fmt.Errorf("parse schedule time failed: %w", err)
	}

	if t.Before(time.Now()) {
		return fmt.Errorf("schedule time is in the past")
	}

	_, err = s.cron.AddFunc(t.Format("0 15 4 * * *"), func() {
		s.taskQueue <- taskID
	})

	if err != nil {
		return fmt.Errorf("schedule task failed: %w", err)
	}

	return nil
}

func (s *Scheduler) processQueue() {
	for taskID := range s.taskQueue {
		if !s.isRunning {
			return
		}
		go func(id int) {
			if err := s.executor.ExecuteTask(id); err != nil {
				utils.Error(fmt.Sprintf("Execute task %d failed: %v", id, err))
			}
		}(taskID)
	}
}

func (s *Scheduler) LoadPendingTasks() error {
	var tasks []database.UploadTask
	result := database.DB.Where("status = ? AND schedule_time != ''", config.TaskStatusPending).Find(&tasks)
	if result.Error != nil {
		return result.Error
	}

	for _, task := range tasks {
		if task.ScheduleTime != nil && *task.ScheduleTime != "" {
			if err := s.ScheduleTask(task.ID, *task.ScheduleTime); err != nil {
				utils.Error(fmt.Sprintf("Schedule task %d failed: %v", task.ID, err))
			}
		}
	}

	return nil
}
