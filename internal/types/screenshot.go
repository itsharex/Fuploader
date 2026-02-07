package types

import "time"

// ScreenshotConfig 截图配置
type ScreenshotConfig struct {
	Enabled      bool              `json:"enabled"`      // 是否启用截图
	GlobalDir    string            `json:"globalDir"`    // 全局截图目录
	PlatformDirs map[string]string `json:"platformDirs"` // 平台专属目录
	AutoClean    bool              `json:"autoClean"`    // 自动清理
	MaxAgeDays   int               `json:"maxAgeDays"`   // 最大保留天数
	MaxSizeMB    int               `json:"maxSizeMB"`    // 最大总大小(MB)
}

// ScreenshotInfo 截图信息
type ScreenshotInfo struct {
	ID        string    `json:"id"`        // 唯一标识（文件名）
	Filename  string    `json:"filename"`  // 文件名
	Platform  string    `json:"platform"`  // 平台名称
	Type      string    `json:"type"`      // 截图类型（upload_success, error 等）
	Size      int64     `json:"size"`      // 文件大小（字节）
	CreatedAt time.Time `json:"createdAt"` // 创建时间
	Path      string    `json:"path"`      // 完整路径
}

// ScreenshotQuery 截图查询参数
type ScreenshotQuery struct {
	Platform  string `json:"platform,omitempty"`  // 平台筛选
	Type      string `json:"type,omitempty"`      // 类型筛选
	StartDate string `json:"startDate,omitempty"` // 开始日期
	EndDate   string `json:"endDate,omitempty"`   // 结束日期
	Page      int    `json:"page"`                // 页码
	PageSize  int    `json:"pageSize"`            // 每页数量
}

// ScreenshotListResult 截图列表结果
type ScreenshotListResult struct {
	List          []ScreenshotInfo `json:"list"`          // 截图列表
	Total         int              `json:"total"`         // 总数
	Page          int              `json:"page"`          // 当前页
	PageSize      int              `json:"pageSize"`      // 每页数量
	TotalSize     int64            `json:"totalSize"`     // 总大小（字节）
	PlatformStats map[string]int   `json:"platformStats"` // 各平台统计
}

// BatchDeleteRequest 批量删除请求
type BatchDeleteRequest struct {
	IDs []string `json:"ids"` // 要删除的截图ID列表
}

// PlatformScreenshotConfig 平台截图配置
type PlatformScreenshotConfig struct {
	Platform        string `json:"platform"`        // 平台代码
	Name            string `json:"name"`            // 平台名称
	Dir             string `json:"dir"`             // 截图目录
	ScreenshotCount int    `json:"screenshotCount"` // 截图数量
}

// DefaultScreenshotConfig 返回默认截图配置
func DefaultScreenshotConfig() *ScreenshotConfig {
	return &ScreenshotConfig{
		Enabled:   false,
		GlobalDir: "./screenshots",
		PlatformDirs: map[string]string{
			"xiaohongshu": "./screenshots/xiaohongshu",
			"tencent":     "./screenshots/tencent",
			"douyin":      "./screenshots/douyin",
			"kuaishou":    "./screenshots/kuaishou",
			"baijiahao":   "./screenshots/baijiahao",
			"tiktok":      "./screenshots/tiktok",
		},
		AutoClean:  false,
		MaxAgeDays: 30,
		MaxSizeMB:  500,
	}
}

// GetPlatformDir 获取平台截图目录
func (c *ScreenshotConfig) GetPlatformDir(platform string) string {
	if dir, ok := c.PlatformDirs[platform]; ok {
		return dir
	}
	return c.GlobalDir
}
