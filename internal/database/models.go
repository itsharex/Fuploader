package database

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

type Account struct {
	ID         int    `json:"id" gorm:"primaryKey"`
	Platform   string `json:"platform" gorm:"index;not null"`
	Name       string `json:"name" gorm:"not null"`
	Username   string `json:"username"`
	Avatar     string `json:"avatar"`
	CookiePath string `json:"cookiePath"`
	Status     int    `json:"status" gorm:"default:0"`
	CreatedAt  string `json:"createdAt"`
	UpdatedAt  string `json:"updatedAt"`
}

func (a *Account) BeforeCreate(tx *gorm.DB) (err error) {
	now := time.Now().Format(time.RFC3339)
	a.CreatedAt = now
	a.UpdatedAt = now
	return nil
}

func (a *Account) BeforeUpdate(tx *gorm.DB) (err error) {
	a.UpdatedAt = time.Now().Format(time.RFC3339)
	return nil
}

type Video struct {
	ID          int      `json:"id" gorm:"primaryKey"`
	Filename    string   `json:"filename" gorm:"not null"`
	FilePath    string   `json:"filePath" gorm:"not null"`
	FileSize    int64    `json:"fileSize"`
	Duration    float64  `json:"duration"`
	Width       int      `json:"width"`
	Height      int      `json:"height"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Tags        []string `json:"tags" gorm:"-"`
	TagsJSON    string   `json:"-" gorm:"column:tags"`
	Thumbnail   string   `json:"thumbnail"`
	CreatedAt   string   `json:"createdAt"`
}

func (v *Video) BeforeCreate(tx *gorm.DB) (err error) {
	v.CreatedAt = time.Now().Format(time.RFC3339)
	if len(v.Tags) > 0 {
		data, _ := json.Marshal(v.Tags)
		v.TagsJSON = string(data)
	}
	return nil
}

func (v *Video) AfterFind(tx *gorm.DB) (err error) {
	if v.TagsJSON != "" {
		json.Unmarshal([]byte(v.TagsJSON), &v.Tags)
	}
	return nil
}

func (v *Video) BeforeSave(tx *gorm.DB) (err error) {
	if len(v.Tags) > 0 {
		data, _ := json.Marshal(v.Tags)
		v.TagsJSON = string(data)
	}
	return nil
}

type UploadTask struct {
	ID           int     `json:"id" gorm:"primaryKey"`
	VideoID      int     `json:"videoId" gorm:"index"`
	Video        Video   `json:"video" gorm:"foreignKey:VideoID"`
	AccountID    int     `json:"accountId" gorm:"index"`
	Account      Account `json:"account" gorm:"foreignKey:AccountID"`
	Platform     string  `json:"platform" gorm:"index;not null"`
	Status       string  `json:"status"`
	Progress     int     `json:"progress" gorm:"default:0"`
	ScheduleTime *string `json:"scheduleTime"`
	PublishURL   string  `json:"publishUrl"`
	ErrorMsg     string  `json:"errorMsg"`
	RetryCount   int     `json:"retryCount" gorm:"default:0"`
	CreatedAt    string  `json:"createdAt"`
	UpdatedAt    string  `json:"updatedAt"`

	// 平台特定字段
	Title        string `json:"title"`        // 用户自定义标题（覆盖视频标题）
	Collection   string `json:"collection"`   // 视频号合集名称
	ShortTitle   string `json:"shortTitle"`   // 视频号短标题
	IsOriginal   bool   `json:"isOriginal"`   // 是否声明原创
	OriginalType string `json:"originalType"` // 原创类型
	Location     string `json:"location"`     // 地理位置
	Thumbnail    string `json:"thumbnail"`    // 封面路径
	SyncToutiao  bool   `json:"syncToutiao"`  // 同步到今日头条
	SyncXigua    bool   `json:"syncXigua"`    // 同步到西瓜视频
	IsDraft      bool   `json:"isDraft"`      // 是否保存为草稿
}

func (t *UploadTask) BeforeCreate(tx *gorm.DB) (err error) {
	now := time.Now().Format(time.RFC3339)
	t.CreatedAt = now
	t.UpdatedAt = now
	return nil
}

func (t *UploadTask) BeforeUpdate(tx *gorm.DB) (err error) {
	t.UpdatedAt = time.Now().Format(time.RFC3339)
	return nil
}

type ScheduleConfig struct {
	ID             int      `json:"id" gorm:"primaryKey"`
	VideosPerDay   int      `json:"videosPerDay" gorm:"default:1"`
	DailyTimes     []string `json:"dailyTimes" gorm:"-"`
	DailyTimesJSON string   `json:"-" gorm:"column:daily_times"`
	StartDays      int      `json:"startDays" gorm:"default:0"`
	TimeZone       string   `json:"timeZone" gorm:"default:'Asia/Shanghai'"`
}

func (s *ScheduleConfig) BeforeCreate(tx *gorm.DB) (err error) {
	if len(s.DailyTimes) > 0 {
		data, _ := json.Marshal(s.DailyTimes)
		s.DailyTimesJSON = string(data)
	}
	return nil
}

func (s *ScheduleConfig) AfterFind(tx *gorm.DB) (err error) {
	if s.DailyTimesJSON != "" {
		json.Unmarshal([]byte(s.DailyTimesJSON), &s.DailyTimes)
	}
	return nil
}

func (s *ScheduleConfig) BeforeSave(tx *gorm.DB) (err error) {
	if len(s.DailyTimes) > 0 {
		data, _ := json.Marshal(s.DailyTimes)
		s.DailyTimesJSON = string(data)
	}
	return nil
}
