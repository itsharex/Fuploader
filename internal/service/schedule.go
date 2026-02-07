package service

import (
	"Fuploader/internal/database"
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type ScheduleService struct {
	db *gorm.DB
}

func NewScheduleService(db *gorm.DB) *ScheduleService {
	return &ScheduleService{db: db}
}

func (s *ScheduleService) GetScheduleConfig(ctx context.Context) (*database.ScheduleConfig, error) {
	var cfg database.ScheduleConfig
	result := s.db.First(&cfg)
	if result.Error != nil {
		cfg = database.ScheduleConfig{
			VideosPerDay:   1,
			DailyTimes:     []string{"09:00"},
			DailyTimesJSON: `["09:00"]`,
			StartDays:      0,
			TimeZone:       "Asia/Shanghai",
		}
		s.db.Create(&cfg)
	}
	return &cfg, nil
}

func (s *ScheduleService) UpdateScheduleConfig(ctx context.Context, cfg *database.ScheduleConfig) error {
	result := s.db.Save(cfg)
	if result.Error != nil {
		return fmt.Errorf("update schedule config failed: %w", result.Error)
	}
	return nil
}

func (s *ScheduleService) GenerateScheduleTimes(ctx context.Context, videoCount int) ([]time.Time, error) {
	cfg, err := s.GetScheduleConfig(ctx)
	if err != nil {
		return nil, err
	}

	dailyTimes := cfg.DailyTimes
	if len(dailyTimes) == 0 {
		dailyTimes = []string{"09:00"}
	}

	loc, err := time.LoadLocation(cfg.TimeZone)
	if err != nil {
		loc = time.Local
	}

	startDate := time.Now().In(loc).AddDate(0, 0, cfg.StartDays)
	startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, loc)

	var scheduleTimes []time.Time
	currentDay := startDate
	count := 0

	for count < videoCount {
		for _, timeStr := range dailyTimes {
			if count >= videoCount {
				break
			}

			t, err := time.ParseInLocation("15:04", timeStr, loc)
			if err != nil {
				continue
			}

			scheduleTime := time.Date(
				currentDay.Year(),
				currentDay.Month(),
				currentDay.Day(),
				t.Hour(),
				t.Minute(),
				0,
				0,
				loc,
			)

			if scheduleTime.Before(time.Now()) {
				continue
			}

			scheduleTimes = append(scheduleTimes, scheduleTime)
			count++
		}
		currentDay = currentDay.AddDate(0, 0, 1)
	}

	return scheduleTimes, nil
}
