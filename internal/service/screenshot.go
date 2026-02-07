package service

import (
	"Fuploader/internal/types"
	"Fuploader/internal/utils"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// ScreenshotService 截图服务
type ScreenshotService struct {
	config     *types.ScreenshotConfig
	configPath string
	mu         sync.RWMutex
}

// NewScreenshotService 创建截图服务
func NewScreenshotService() *ScreenshotService {
	service := &ScreenshotService{
		configPath: "./config/screenshot.json",
	}

	// 加载配置
	service.loadConfig()

	return service
}

// loadConfig 加载截图配置
func (s *ScreenshotService) loadConfig() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 默认配置
	s.config = types.DefaultScreenshotConfig()

	// 尝试从文件加载
	data, err := ioutil.ReadFile(s.configPath)
	if err != nil {
		utils.Info("[-] 使用默认截图配置")
		return
	}

	var loadedConfig types.ScreenshotConfig
	if err := json.Unmarshal(data, &loadedConfig); err != nil {
		utils.Warn(fmt.Sprintf("[-] 加载截图配置失败: %v, 使用默认配置", err))
		return
	}

	s.config = &loadedConfig
	utils.Info("[-] 已加载截图配置")
}

// saveConfig 保存截图配置
func (s *ScreenshotService) saveConfig() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 确保目录存在
	dir := filepath.Dir(s.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	data, err := json.MarshalIndent(s.config, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	if err := ioutil.WriteFile(s.configPath, data, 0644); err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}

	return nil
}

// GetConfig 获取截图配置
func (s *ScreenshotService) GetConfig() *types.ScreenshotConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 返回副本
	config := *s.config
	return &config
}

// UpdateConfig 更新截图配置
func (s *ScreenshotService) UpdateConfig(config *types.ScreenshotConfig) error {
	s.mu.Lock()
	s.config = config
	s.mu.Unlock()

	return s.saveConfig()
}

// IsScreenshotEnabled 检查是否启用截图
func (s *ScreenshotService) IsScreenshotEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config.Enabled
}

// GetScreenshotDir 获取平台截图目录
func (s *ScreenshotService) GetScreenshotDir(platform string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config.GetPlatformDir(platform)
}

// ListScreenshots 获取截图列表
func (s *ScreenshotService) ListScreenshots(query types.ScreenshotQuery) (*types.ScreenshotListResult, error) {
	s.mu.RLock()
	config := s.config
	s.mu.RUnlock()

	var screenshots []types.ScreenshotInfo
	platformStats := make(map[string]int)
	var totalSize int64

	// 遍历所有截图目录
	dirs := []string{config.GlobalDir}
	for _, dir := range config.PlatformDirs {
		if dir != config.GlobalDir {
			dirs = append(dirs, dir)
		}
	}

	for _, dir := range dirs {
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			utils.Warn(fmt.Sprintf("[-] 读取截图目录失败 %s: %v", dir, err))
			continue
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}

			// 只处理 PNG 文件
			if !strings.HasSuffix(strings.ToLower(file.Name()), ".png") {
				continue
			}

			// 解析文件名获取信息
			info := s.parseScreenshotInfo(file.Name(), filepath.Join(dir, file.Name()), file)

			// 筛选
			if query.Platform != "" && info.Platform != query.Platform {
				continue
			}
			if query.Type != "" && info.Type != query.Type {
				continue
			}
			if query.StartDate != "" {
				startTime, _ := time.Parse("2006-01-02", query.StartDate)
				if info.CreatedAt.Before(startTime) {
					continue
				}
			}
			if query.EndDate != "" {
				endTime, _ := time.Parse("2006-01-02", query.EndDate)
				endTime = endTime.Add(24 * time.Hour)
				if info.CreatedAt.After(endTime) {
					continue
				}
			}

			screenshots = append(screenshots, info)
			platformStats[info.Platform]++
			totalSize += info.Size
		}
	}

	// 按时间倒序排序
	sort.Slice(screenshots, func(i, j int) bool {
		return screenshots[i].CreatedAt.After(screenshots[j].CreatedAt)
	})

	total := len(screenshots)

	// 分页
	page := query.Page
	if page < 1 {
		page = 1
	}
	pageSize := query.PageSize
	if pageSize < 1 {
		pageSize = 20
	}

	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	return &types.ScreenshotListResult{
		List:          screenshots[start:end],
		Total:         total,
		Page:          page,
		PageSize:      pageSize,
		TotalSize:     totalSize,
		PlatformStats: platformStats,
	}, nil
}

// parseScreenshotInfo 解析截图信息
func (s *ScreenshotService) parseScreenshotInfo(filename, fullPath string, fileInfo os.FileInfo) types.ScreenshotInfo {
	// 解析文件名: {platform}_{type}_{timestamp}.png 或 {type}_{timestamp}.png（旧格式）
	parts := strings.Split(filename, "_")
	platform := "unknown"
	screenshotType := "unknown"

	if len(parts) >= 3 {
		// 新格式: platform_type_timestamp.png
		platform = parts[0]
		screenshotType = parts[1]
	} else if len(parts) >= 2 {
		// 旧格式: type_timestamp.png，尝试从路径推断平台
		screenshotType = parts[0]
		dir := filepath.Dir(fullPath)
		for p, d := range s.config.PlatformDirs {
			if strings.Contains(dir, d) {
				platform = p
				break
			}
		}
	}

	// 尝试从文件名解析时间
	createdAt := fileInfo.ModTime()
	if len(parts) >= 3 {
		// 新格式: platform_type_YYYYMMDD_HHMMSS.png
		timeStr := parts[2]
		if len(parts) >= 4 {
			timeStr = parts[2] + parts[3]
		}
		timeStr = strings.TrimSuffix(timeStr, ".png")
		if t, err := time.Parse("20060102150405", timeStr); err == nil {
			createdAt = t
		}
	} else if len(parts) >= 2 {
		// 旧格式: type_YYYYMMDD_HHMMSS.png
		timeStr := parts[1]
		if len(parts) >= 3 {
			timeStr = parts[1] + parts[2]
		}
		timeStr = strings.TrimSuffix(timeStr, ".png")
		if t, err := time.Parse("20060102150405", timeStr); err == nil {
			createdAt = t
		}
	}

	return types.ScreenshotInfo{
		ID:        filename,
		Filename:  filename,
		Platform:  platform,
		Type:      screenshotType,
		Size:      fileInfo.Size(),
		CreatedAt: createdAt,
		Path:      fullPath,
	}
}

// DeleteScreenshot 删除单个截图
func (s *ScreenshotService) DeleteScreenshot(id string) error {
	// 查找截图路径
	s.mu.RLock()
	dirs := []string{s.config.GlobalDir}
	for _, dir := range s.config.PlatformDirs {
		if dir != s.config.GlobalDir {
			dirs = append(dirs, dir)
		}
	}
	s.mu.RUnlock()

	for _, dir := range dirs {
		path := filepath.Join(dir, id)
		if _, err := os.Stat(path); err == nil {
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("删除截图失败: %w", err)
			}
			utils.Info(fmt.Sprintf("[-] 已删除截图: %s", path))
			return nil
		}
	}

	return fmt.Errorf("截图不存在: %s", id)
}

// BatchDeleteScreenshots 批量删除截图
func (s *ScreenshotService) BatchDeleteScreenshots(ids []string) (int, error) {
	deleted := 0
	var lastErr error

	for _, id := range ids {
		if err := s.DeleteScreenshot(id); err != nil {
			lastErr = err
			utils.Warn(fmt.Sprintf("[-] 删除截图失败 %s: %v", id, err))
		} else {
			deleted++
		}
	}

	return deleted, lastErr
}

// DeleteAllScreenshots 删除所有截图
func (s *ScreenshotService) DeleteAllScreenshots() (int, error) {
	s.mu.RLock()
	dirs := []string{s.config.GlobalDir}
	for _, dir := range s.config.PlatformDirs {
		if dir != s.config.GlobalDir {
			dirs = append(dirs, dir)
		}
	}
	s.mu.RUnlock()

	deleted := 0
	for _, dir := range dirs {
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return deleted, err
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}
			if !strings.HasSuffix(strings.ToLower(file.Name()), ".png") {
				continue
			}

			path := filepath.Join(dir, file.Name())
			if err := os.Remove(path); err != nil {
				utils.Warn(fmt.Sprintf("[-] 删除截图失败 %s: %v", path, err))
			} else {
				deleted++
			}
		}
	}

	utils.Info(fmt.Sprintf("[-] 已删除 %d 个截图", deleted))
	return deleted, nil
}

// CleanOldScreenshots 清理旧截图
func (s *ScreenshotService) CleanOldScreenshots() (int, error) {
	s.mu.RLock()
	config := s.config
	s.mu.RUnlock()

	if !config.AutoClean {
		return 0, nil
	}

	deleted := 0
	cutoffTime := time.Now().AddDate(0, 0, -config.MaxAgeDays)

	// 遍历所有目录
	dirs := []string{config.GlobalDir}
	for _, dir := range config.PlatformDirs {
		if dir != config.GlobalDir {
			dirs = append(dirs, dir)
		}
	}

	for _, dir := range dirs {
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return deleted, err
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}
			if !strings.HasSuffix(strings.ToLower(file.Name()), ".png") {
				continue
			}

			// 检查文件修改时间
			if file.ModTime().Before(cutoffTime) {
				path := filepath.Join(dir, file.Name())
				if err := os.Remove(path); err != nil {
					utils.Warn(fmt.Sprintf("[-] 清理截图失败 %s: %v", path, err))
				} else {
					deleted++
				}
			}
		}
	}

	utils.Info(fmt.Sprintf("[-] 已清理 %d 个旧截图", deleted))
	return deleted, nil
}

// GetPlatformScreenshotStats 获取各平台截图统计
func (s *ScreenshotService) GetPlatformScreenshotStats() []types.PlatformScreenshotConfig {
	s.mu.RLock()
	config := s.config
	s.mu.RUnlock()

	var stats []types.PlatformScreenshotConfig

	// 全局目录
	globalCount := s.countScreenshotsInDir(config.GlobalDir)
	stats = append(stats, types.PlatformScreenshotConfig{
		Platform:        "",
		Name:            "全局",
		Dir:             config.GlobalDir,
		ScreenshotCount: globalCount,
	})

	// 各平台目录
	platformNames := map[string]string{
		"xiaohongshu": "小红书",
		"tencent":     "视频号",
		"douyin":      "抖音",
		"kuaishou":    "快手",
		"baijiahao":   "百家号",
		"tiktok":      "TikTok",
	}

	for platform, dir := range config.PlatformDirs {
		count := s.countScreenshotsInDir(dir)
		name := platformNames[platform]
		if name == "" {
			name = platform
		}
		stats = append(stats, types.PlatformScreenshotConfig{
			Platform:        platform,
			Name:            name,
			Dir:             dir,
			ScreenshotCount: count,
		})
	}

	return stats
}

// countScreenshotsInDir 统计目录中的截图数量
func (s *ScreenshotService) countScreenshotsInDir(dir string) int {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return 0
	}

	count := 0
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(file.Name()), ".png") {
			count++
		}
	}

	return count
}
