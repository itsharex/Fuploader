package app

import (
	"Fuploader/internal/config"
	"Fuploader/internal/database"
	"Fuploader/internal/platform/baijiahao"
	"Fuploader/internal/platform/bilibili"
	"Fuploader/internal/platform/douyin"
	"Fuploader/internal/platform/kuaishou"
	"Fuploader/internal/platform/tiktok"
	"Fuploader/internal/platform/xiaohongshu"
	"Fuploader/internal/scheduler"
	"Fuploader/internal/service"
	"Fuploader/internal/types"
	"Fuploader/internal/utils"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"gorm.io/gorm"
)

type App struct {
	ctx                  context.Context
	accountService       *service.AccountService
	fileService          *service.FileService
	uploadService        *service.UploadService
	scheduleService      *service.ScheduleService
	logService           *service.LogService
	screenshotService    *service.ScreenshotService
	scheduler            *scheduler.Scheduler
	enhancedScheduler    *scheduler.EnhancedScheduler
	useEnhancedScheduler bool
	initialized          bool
	initError            string
}

func NewApp() *App {
	return &App{
		useEnhancedScheduler: true, // 默认使用增强调度器
	}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	if err := config.Init(); err != nil {
		a.initError = fmt.Sprintf("Config init failed: %v", err)
		fmt.Println(a.initError)
		return
	}

	if err := utils.InitLogger(); err != nil {
		a.initError = fmt.Sprintf("Logger init failed: %v", err)
		fmt.Println(a.initError)
		return
	}

	if err := database.Init(); err != nil {
		a.initError = fmt.Sprintf("Database init failed: %v", err)
		fmt.Println(a.initError)
		return
	}

	db := database.GetDB()
	a.accountService = service.NewAccountService(db)
	a.fileService = service.NewFileService(db)
	a.uploadService = service.NewUploadService(db)
	a.scheduleService = service.NewScheduleService(db)
	a.logService = service.NewLogService()
	a.screenshotService = service.NewScreenshotService()

	// 将 LogService 注入到 logger，使日志同时输出到前端
	utils.SetLogService(a.logService)

	a.setupEventListeners()

	// 使用增强调度器
	if a.useEnhancedScheduler {
		a.setupEnhancedScheduler(db)
	} else {
		// 使用旧版调度器作为 fallback
		a.scheduler = scheduler.NewScheduler(a)
		a.scheduler.Start()
		a.scheduler.LoadPendingTasks()
	}

	a.initialized = true
	utils.Info("Application started successfully")
}

// setupEnhancedScheduler 设置增强调度器
func (a *App) setupEnhancedScheduler(db *gorm.DB) {
	// 创建增强调度器，使用5个工作线程
	a.enhancedScheduler = scheduler.NewEnhancedScheduler(db, 5)

	// 注册各平台上传器
	a.registerUploaders()

	// 启动调度器
	a.enhancedScheduler.Start()

	utils.Info("[+] 增强调度器已启动")
}

// registerUploaders 注册各平台上传器
func (a *App) registerUploaders() {
	// 获取所有账号，为每个账号创建上传器
	accounts, err := a.accountService.GetAccounts(a.ctx)
	if err != nil {
		utils.Error(fmt.Sprintf("[-] 获取账号列表失败: %v", err))
		return
	}

	for _, account := range accounts {
		cookiePath := a.accountService.GetCookiePath(uint(account.ID))
		var uploader types.Uploader

		switch account.Platform {
		case "xiaohongshu":
			uploader = xiaohongshu.NewUploader(cookiePath)
		case "douyin":
			uploader = douyin.NewUploader(cookiePath)
		case "bilibili":
			uploader = bilibili.NewUploader(cookiePath)
		case "kuaishou":
			uploader = kuaishou.NewUploader(cookiePath)
		case "tiktok":
			uploader = tiktok.NewUploader(cookiePath)
		case "baijiahao":
			uploader = baijiahao.NewUploader(cookiePath)
		default:
			utils.Warn(fmt.Sprintf("[-] 未知平台: %s", account.Platform))
			continue
		}

		if uploader != nil {
			a.enhancedScheduler.RegisterUploader(account.Platform, uploader)
			utils.Info(fmt.Sprintf("[+] 已注册上传器 - 平台: %s, 账号: %s", account.Platform, account.Name))
		}
	}
}

// GetAppStatus 获取应用初始化状态
func (a *App) GetAppStatus() (map[string]interface{}, error) {
	return map[string]interface{}{
		"initialized": a.initialized,
		"error":       a.initError,
		"version":     config.AppVersion,
	}, nil
}

func (a *App) Shutdown(ctx context.Context) {
	utils.Info("Application is shutting down...")

	// 优雅关闭：等待运行中的任务完成
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// 停止调度器
	if a.enhancedScheduler != nil {
		a.enhancedScheduler.Stop()
		utils.Info("[-] 增强调度器已停止")
	}
	if a.scheduler != nil {
		a.scheduler.Stop()
		utils.Info("[-] 旧版调度器已停止")
	}

	// 等待所有任务完成或超时
	select {
	case <-shutdownCtx.Done():
		utils.Warn("[-] 关闭超时，强制退出")
	case <-time.After(1 * time.Second):
		// 给一点时间让日志写入
	}

	// 关闭数据库
	if err := database.Close(); err != nil {
		utils.Error(fmt.Sprintf("[-] 数据库关闭失败: %v", err))
	}

	utils.Info("Application shutdown complete")
}

// ExecuteTask 执行指定任务（由调度器调用）
func (a *App) ExecuteTask(taskID int) error {
	// 调度器会调用此方法执行定时任务
	// 实际执行逻辑在 uploadService.executeTask 中
	// 这里触发事件通知前端
	a.emitEvent(config.EventTaskStatusChanged, types.TaskStatusChangedEvent{
		TaskID:    taskID,
		OldStatus: config.TaskStatusPending,
		NewStatus: config.TaskStatusUploading,
	})
	return nil
}

func (a *App) emitEvent(eventName string, data interface{}) {
	if a.ctx == nil {
		return
	}
	wailsRuntime.EventsEmit(a.ctx, eventName, data)
}

func (a *App) setupEventListeners() {
	eventBus := a.uploadService.GetEventBus()

	eventBus.Subscribe(config.EventUploadProgress, func(data interface{}) {
		if progress, ok := data.(types.UploadProgressEvent); ok {
			a.emitEvent(config.EventUploadProgress, progress)
		}
	})

	eventBus.Subscribe(config.EventUploadComplete, func(data interface{}) {
		if result, ok := data.(types.UploadCompleteEvent); ok {
			a.emitEvent(config.EventUploadComplete, result)
		}
	})

	eventBus.Subscribe(config.EventUploadError, func(data interface{}) {
		if result, ok := data.(types.UploadErrorEvent); ok {
			a.emitEvent(config.EventUploadError, result)
		}
	})

	eventBus.Subscribe(config.EventTaskStatusChanged, func(data interface{}) {
		if eventData, ok := data.(types.TaskStatusChangedEvent); ok {
			a.emitEvent(config.EventTaskStatusChanged, eventData)
		}
	})
}

func (a *App) GetAccounts() ([]database.Account, error) {
	return a.accountService.GetAccounts(a.ctx)
}

func (a *App) AddAccount(platform string, name string) (*database.Account, error) {
	return a.accountService.AddAccount(a.ctx, platform, name)
}

func (a *App) DeleteAccount(id int) error {
	return a.accountService.DeleteAccount(a.ctx, id)
}

func (a *App) UpdateAccount(account database.Account) error {
	return a.accountService.UpdateAccount(a.ctx, &account)
}

func (a *App) ValidateAccount(id int) (bool, error) {
	return a.accountService.ValidateAccount(a.ctx, id)
}

func (a *App) LoginAccount(id int) error {
	account, err := a.accountService.GetAccounts(a.ctx)
	if err != nil {
		a.emitEvent(config.EventLoginError, types.LoginErrorEvent{
			AccountID: id,
			Platform:  "",
			Error:     "failed to get accounts",
		})
		return fmt.Errorf("failed to get accounts: %w", err)
	}

	var targetAccount *database.Account
	for i := range account {
		if account[i].ID == id {
			targetAccount = &account[i]
			break
		}
	}

	if targetAccount == nil {
		a.emitEvent(config.EventLoginError, types.LoginErrorEvent{
			AccountID: id,
			Platform:  "",
			Error:     "account not found",
		})
		return fmt.Errorf("account not found")
	}

	err = a.accountService.LoginAccount(a.ctx, id)
	if err != nil {
		a.emitEvent(config.EventLoginError, types.LoginErrorEvent{
			AccountID: id,
			Platform:  targetAccount.Platform,
			Error:     err.Error(),
		})
		return err
	}

	a.emitEvent(config.EventLoginSuccess, types.LoginSuccessEvent{
		AccountID: id,
		Platform:  targetAccount.Platform,
		Username:  targetAccount.Username,
	})

	a.emitEvent(config.EventAccountStatusChanged, types.AccountStatusChangedEvent{
		AccountID: id,
		OldStatus: targetAccount.Status,
		NewStatus: config.AccountStatusValid,
	})

	// 登录成功后，重新注册上传器
	if a.enhancedScheduler != nil {
		a.registerUploaders()
	}

	return nil
}

func (a *App) GetVideos() ([]database.Video, error) {
	return a.fileService.GetVideos(a.ctx)
}

func (a *App) AddVideo(filePath string) (*database.Video, error) {
	return a.fileService.AddVideo(a.ctx, filePath)
}

func (a *App) UpdateVideo(video database.Video) error {
	return a.fileService.UpdateVideo(a.ctx, &video)
}

func (a *App) DeleteVideo(id int) error {
	return a.fileService.DeleteVideo(a.ctx, id)
}

func (a *App) CreateUploadTask(
	videoID int,
	accountIDs []int,
	scheduleTime *string,
	metadata *string,
) ([]database.UploadTask, error) {
	// 解析元数据
	var taskMetadata *service.UploadTaskMetadata
	if metadata != nil && *metadata != "" {
		taskMetadata = &service.UploadTaskMetadata{}
		if err := json.Unmarshal([]byte(*metadata), taskMetadata); err != nil {
			return nil, fmt.Errorf("failed to parse metadata: %w", err)
		}
	}

	return a.uploadService.CreateUploadTask(a.ctx, videoID, accountIDs, scheduleTime, taskMetadata)
}

func (a *App) GetUploadTasks(status string) ([]database.UploadTask, error) {
	return a.uploadService.GetUploadTasks(a.ctx, status)
}

func (a *App) CancelUploadTask(id int) error {
	return a.uploadService.CancelUploadTask(a.ctx, id)
}

func (a *App) RetryUploadTask(id int) error {
	return a.uploadService.RetryUploadTask(a.ctx, id)
}

func (a *App) DeleteUploadTask(id int) error {
	return a.uploadService.DeleteUploadTask(a.ctx, id)
}

func (a *App) GetScheduleConfig() (*database.ScheduleConfig, error) {
	return a.scheduleService.GetScheduleConfig(a.ctx)
}

func (a *App) UpdateScheduleConfig(config database.ScheduleConfig) error {
	return a.scheduleService.UpdateScheduleConfig(a.ctx, &config)
}

func (a *App) GenerateScheduleTimes(videoCount int) ([]string, error) {
	times, err := a.scheduleService.GenerateScheduleTimes(a.ctx, videoCount)
	if err != nil {
		return nil, err
	}

	result := make([]string, len(times))
	for i, t := range times {
		result[i] = t.Format(time.RFC3339)
	}
	return result, nil
}

func (a *App) GetAppVersion() (types.AppVersion, error) {
	return types.AppVersion{
		Version:      config.AppVersion,
		BuildTime:    time.Now().Format("2006-01-02"),
		GoVersion:    runtime.Version(),
		WailsVersion: "v2.10.1",
	}, nil
}

func (a *App) SelectVideoFile() (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("context not initialized")
	}

	selection, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "选择视频文件",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "视频文件 (*.mp4, *.mov, *.avi)", Pattern: "*.mp4;*.mov;*.avi"},
			{DisplayName: "所有文件 (*.*)", Pattern: "*.*"},
		},
	})

	if err != nil {
		return "", fmt.Errorf("open file dialog failed: %w", err)
	}

	return selection, nil
}

func (a *App) OpenDirectory(path string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", path)
	case "darwin":
		cmd = exec.Command("open", path)
	case "linux":
		cmd = exec.Command("xdg-open", path)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open directory failed: %w", err)
	}

	return nil
}

// ============================================
// 平台特定 API（前端调用）
// ============================================

// GetCollections 获取平台合集列表（视频号）
func (a *App) GetCollections(platform string) ([]map[string]string, error) {
	// TODO: 实现获取合集列表逻辑
	// 这里需要根据平台调用相应的上传器获取合集
	utils.Info(fmt.Sprintf("[-] 获取 %s 合集列表", platform))

	// 返回模拟数据，实际实现需要从平台获取
	return []map[string]string{
		{"label": "默认合集", "value": "default"},
		{"label": "测试合集", "value": "test"},
	}, nil
}

// AutoSelectCover 自动选择推荐封面（抖音）
func (a *App) AutoSelectCover(videoID int) (map[string]string, error) {
	utils.Info(fmt.Sprintf("[-] 为视频 %d 自动选择推荐封面", videoID))

	// 获取视频信息
	video, err := a.fileService.GetVideoByID(a.ctx, videoID)
	if err != nil {
		return nil, fmt.Errorf("获取视频信息失败: %w", err)
	}

	// TODO: 实际实现需要从视频中提取帧作为封面
	// 这里返回视频路径作为占位符
	return map[string]string{
		"thumbnailPath": video.FilePath,
	}, nil
}

// ValidateProductLink 验证商品链接（抖音）
func (a *App) ValidateProductLink(link string) (map[string]interface{}, error) {
	utils.Info(fmt.Sprintf("[-] 验证商品链接: %s", link))

	// 简单的链接格式验证
	if link == "" {
		return map[string]interface{}{
			"valid": false,
			"error": "链接不能为空",
		}, nil
	}

	// 检查链接格式
	if !strings.HasPrefix(link, "http://") && !strings.HasPrefix(link, "https://") {
		return map[string]interface{}{
			"valid": false,
			"error": "链接格式不正确，必须以 http:// 或 https:// 开头",
		}, nil
	}

	// TODO: 实际实现需要调用抖音API验证链接有效性
	// 这里返回模拟成功结果
	return map[string]interface{}{
		"valid": true,
		"title": "商品标题（待获取）",
	}, nil
}

// SelectImageFile 选择图片文件
func (a *App) SelectImageFile() (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("context not initialized")
	}

	selection, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "选择图片文件",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "图片文件 (*.jpg, *.jpeg, *.png, *.gif)", Pattern: "*.jpg;*.jpeg;*.png;*.gif"},
			{DisplayName: "所有文件 (*.*)", Pattern: "*.*"},
		},
	})

	if err != nil {
		return "", fmt.Errorf("open file dialog failed: %w", err)
	}

	return selection, nil
}

// SelectFile 选择文件
func (a *App) SelectFile(accept string) (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("context not initialized")
	}

	filters := []wailsRuntime.FileFilter{
		{DisplayName: "所有文件 (*.*)", Pattern: "*.*"},
	}

	// 根据 accept 参数设置过滤器
	if accept != "" && accept != "*/*" {
		filters = []wailsRuntime.FileFilter{
			{DisplayName: fmt.Sprintf("指定类型 (%s)", accept), Pattern: accept},
			{DisplayName: "所有文件 (*.*)", Pattern: "*.*"},
		}
	}

	selection, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title:   "选择文件",
		Filters: filters,
	})

	if err != nil {
		return "", fmt.Errorf("open file dialog failed: %w", err)
	}

	return selection, nil
}

// GetLogs 获取日志
func (a *App) GetLogs(query types.LogQuery) ([]types.SimpleLog, error) {
	return a.logService.Query(query), nil
}

// AddLog 添加日志（供内部使用）
func (a *App) AddLog(message string) {
	a.logService.Add(message)
}

// ============================================
// 截图管理 API
// ============================================

// GetScreenshotConfig 获取截图配置
func (a *App) GetScreenshotConfig() (*types.ScreenshotConfig, error) {
	return a.screenshotService.GetConfig(), nil
}

// UpdateScreenshotConfig 更新截图配置
func (a *App) UpdateScreenshotConfig(config types.ScreenshotConfig) error {
	return a.screenshotService.UpdateConfig(&config)
}

// GetScreenshots 获取截图列表
func (a *App) GetScreenshots(query types.ScreenshotQuery) (*types.ScreenshotListResult, error) {
	return a.screenshotService.ListScreenshots(query)
}

// DeleteScreenshot 删除单个截图
func (a *App) DeleteScreenshot(id string) error {
	return a.screenshotService.DeleteScreenshot(id)
}

// BatchDeleteScreenshots 批量删除截图
func (a *App) BatchDeleteScreenshots(ids []string) (int, error) {
	return a.screenshotService.BatchDeleteScreenshots(ids)
}

// DeleteAllScreenshots 删除所有截图
func (a *App) DeleteAllScreenshots() (int, error) {
	return a.screenshotService.DeleteAllScreenshots()
}

// GetPlatformScreenshotStats 获取各平台截图统计
func (a *App) GetPlatformScreenshotStats() ([]types.PlatformScreenshotConfig, error) {
	return a.screenshotService.GetPlatformScreenshotStats(), nil
}

// CleanOldScreenshots 清理旧截图
func (a *App) CleanOldScreenshots() (int, error) {
	return a.screenshotService.CleanOldScreenshots()
}

// OpenScreenshotDir 打开截图目录
func (a *App) OpenScreenshotDir(platform string) error {
	dir := a.screenshotService.GetScreenshotDir(platform)
	return a.OpenDirectory(dir)
}
