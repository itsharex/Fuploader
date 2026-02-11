package types

import "context"

// VideoTask 视频任务
type VideoTask struct {
	Platform            string // 平台名称
	VideoPath           string
	Title               string
	Description         string
	Tags                []string
	Thumbnail           string // 封面路径
	ScheduleTime        *string
	IsDraft             bool   // 是否保存为草稿
	Location            string // 地理位置
	SyncToutiao         bool   // 同步到今日头条
	SyncXigua           bool   // 同步到西瓜视频
	ShortTitle          string // 视频号短标题
	IsOriginal          bool   // 是否声明原创
	OriginalType        string // 原创类型
	Collection          string // 合集名称
	ProductLink         string // 商品链接（抖音）
	ProductTitle        string // 商品短标题（抖音）
	Copyright           string // 转载类型（B站）：1=自制，2=转载
	AllowDownload       bool   // 是否允许下载（抖音/快手）
	AllowComment        bool   // 是否允许评论（抖音/TikTok）
	AllowDuet           bool   // 是否允许合拍（TikTok）
	AIDeclaration       bool   // AI创作声明（百家号）
	AutoGenerateAudio   bool   // 自动生成音频（百家号）
	CoverType           string // 封面模式（百家号）：auto/single/triple
	Category            string // 分类（百家号）
	UseIframe           bool   // 是否使用iframe模式（TikTok）
	UseFileChooser      bool   // 是否使用文件选择器（快手）
	SkipNewFeatureGuide bool   // 是否跳过新功能引导（快手）
}

// Uploader 上传器接口
type Uploader interface {
	ValidateCookie(ctx context.Context) (bool, error)
	Upload(ctx context.Context, task *VideoTask) error
	Login() error
	Platform() string
}

// PlatformFields 平台特定字段
type PlatformFields struct {
	Title               string `json:"title"`
	Collection          string `json:"collection"`
	ShortTitle          string `json:"shortTitle"`
	IsOriginal          bool   `json:"isOriginal"`
	OriginalType        string `json:"originalType"`
	Location            string `json:"location"`
	Thumbnail           string `json:"thumbnail"`
	SyncToutiao         bool   `json:"syncToutiao"`
	SyncXigua           bool   `json:"syncXigua"`
	IsDraft             bool   `json:"isDraft"`
	Copyright           string `json:"copyright"`           // 转载类型（B站）：1=自制，2=转载
	AllowDownload       bool   `json:"allowDownload"`       // 是否允许下载（抖音/快手）
	AllowComment        bool   `json:"allowComment"`        // 是否允许评论（抖音/TikTok）
	AllowDuet           bool   `json:"allowDuet"`           // 是否允许合拍（TikTok）
	AIDeclaration       bool   `json:"aiDeclaration"`       // AI创作声明（百家号）
	AutoGenerateAudio   bool   `json:"autoGenerateAudio"`   // 自动生成音频（百家号）
	CoverType           string `json:"coverType"`           // 封面模式（百家号）：auto/single/triple
	Category            string `json:"category"`            // 分类（百家号）
	UseIframe           bool   `json:"useIframe"`           // 是否使用iframe模式（TikTok）
	UseFileChooser      bool   `json:"useFileChooser"`      // 是否使用文件选择器（快手）
	SkipNewFeatureGuide bool   `json:"skipNewFeatureGuide"` // 是否跳过新功能引导（快手）
}

// CommonMetadata 通用元数据
type CommonMetadata struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// UploadTaskMetadata 上传任务元数据
type UploadTaskMetadata struct {
	Common    CommonMetadata            `json:"common"`
	Platforms map[string]PlatformFields `json:"platforms"`
}
