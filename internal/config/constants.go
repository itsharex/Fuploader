package config

const (
	AppName    = "Fuploader"
	AppVersion = "1.0.0"
)

const (
	PlatformDouyin      = "douyin"
	PlatformTencent     = "tencent"
	PlatformKuaishou    = "kuaishou"
	PlatformTiktok      = "tiktok"
	PlatformBilibili    = "bilibili"
	PlatformXiaohongshu = "xiaohongshu"
	PlatformBaijiahao   = "baijiahao"
)

var SupportedPlatforms = []string{
	PlatformDouyin,
	PlatformTencent,
	PlatformKuaishou,
	PlatformTiktok,
	PlatformBilibili,
	PlatformXiaohongshu,
	PlatformBaijiahao,
}

const (
	AccountStatusInvalid = 0
	AccountStatusValid   = 1
	AccountStatusExpired = 2
)

const (
	TaskStatusPending   = "pending"
	TaskStatusUploading = "uploading"
	TaskStatusSuccess   = "success"
	TaskStatusFailed    = "failed"
	TaskStatusCancelled = "cancelled"
)

const (
	ErrInvalidParam     = "ERR_INVALID_PARAM"
	ErrAccountNotFound  = "ERR_ACCOUNT_NOT_FOUND"
	ErrAccountInvalid   = "ERR_ACCOUNT_INVALID"
	ErrVideoNotFound    = "ERR_VIDEO_NOT_FOUND"
	ErrVideoInvalid     = "ERR_VIDEO_INVALID"
	ErrTaskNotFound     = "ERR_TASK_NOT_FOUND"
	ErrTaskCannotCancel = "ERR_TASK_CANNOT_CANCEL"
	ErrUploadFailed     = "ERR_UPLOAD_FAILED"
	ErrNetworkError     = "ERR_NETWORK_ERROR"
	ErrPlatformError    = "ERR_PLATFORM_ERROR"
	ErrScheduleInvalid  = "ERR_SCHEDULE_INVALID"
	ErrInternal         = "ERR_INTERNAL"
)

const (
	EventUploadProgress       = "upload:progress"
	EventUploadComplete       = "upload:complete"
	EventUploadError          = "upload:error"
	EventLoginSuccess         = "login:success"
	EventLoginError           = "login:error"
	EventTaskStatusChanged    = "task:statusChanged"
	EventAccountStatusChanged = "account:statusChanged"
)

const (
	DefaultDbPath        = "storage/data.db"
	DefaultCookiePath    = "storage/cookies"
	DefaultVideoPath     = "storage/videos"
	DefaultLogPath       = "storage/logs"
	DefaultThumbnailPath = "storage/thumbnails"
)

const (
	MaxUploadRetry    = 3
	DefaultTimeout    = 30
	UploadConcurrency = 2
)
