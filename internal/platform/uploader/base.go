package uploader

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"Fuploader/internal/platform/browser"
	"Fuploader/internal/types"
	"Fuploader/internal/utils"

	"github.com/playwright-community/playwright-go"
)

// Step 上传步骤
type Step int

const (
	StepInit Step = iota
	StepValidateSession
	StepNavigate
	StepUploadMedia
	StepFillTitle
	StepFillContent
	StepAddTags
	StepSetCover
	StepSetSchedule
	StepPublish
	StepComplete
)

func (s Step) String() string {
	switch s {
	case StepInit:
		return "初始化"
	case StepValidateSession:
		return "验证会话"
	case StepNavigate:
		return "页面导航"
	case StepUploadMedia:
		return "上传媒体"
	case StepFillTitle:
		return "填写标题"
	case StepFillContent:
		return "填写内容"
	case StepAddTags:
		return "添加标签"
	case StepSetCover:
		return "设置封面"
	case StepSetSchedule:
		return "设置定时"
	case StepPublish:
		return "发布"
	case StepComplete:
		return "完成"
	default:
		return "未知步骤"
	}
}

// StepResult 步骤结果
type StepResult struct {
	Step    Step
	Success bool
	Error   error
	Data    map[string]interface{}
}

// StepFunc 步骤函数
type StepFunc func(ctx *Context) StepResult

// ProgressCallback 进度回调函数
type ProgressCallback func(step Step, progress int, message string)

// Context 上传上下文
type Context struct {
	Ctx              context.Context
	Task             *types.VideoTask
	BrowserCtx       *browser.PooledContext
	Page             playwright.Page
	stepResults      []StepResult
	retryCount       int
	maxRetries       int
	progressCb       ProgressCallback
	screenshotDir    string
	enableScreenshot bool
	retryPolicy      types.RetryPolicy
}

// SetProgressCallback 设置进度回调
func (c *Context) SetProgressCallback(cb ProgressCallback) {
	c.progressCb = cb
}

// ReportProgress 上报进度（导出方法）
func (c *Context) ReportProgress(step Step, progress int, message string) {
	if c.progressCb != nil {
		c.progressCb(step, progress, message)
	}
}

// TakeScreenshot 截图保存（导出方法）
func (c *Context) TakeScreenshot(name string) string {
	if !c.enableScreenshot || c.Page == nil {
		return ""
	}

	timestamp := time.Now().Format("20060102_150405")
	platform := c.Task.Platform
	if platform == "" {
		platform = "unknown"
	}
	filename := fmt.Sprintf("%s_%s_%s.png", platform, name, timestamp)
	filepath := filepath.Join(c.screenshotDir, filename)

	// 确保目录存在
	os.MkdirAll(c.screenshotDir, 0755)

	_, err := c.Page.Screenshot(playwright.PageScreenshotOptions{
		Path:     playwright.String(filepath),
		FullPage: playwright.Bool(true),
	})
	if err != nil {
		utils.Warn(fmt.Sprintf("[-] 截图失败: %v", err))
		return ""
	}

	utils.Info(fmt.Sprintf("[-] 已保存截图: %s", filepath))
	return filepath
}

// Base 基础上传器
type Base struct {
	platform         string
	cookiePath       string
	browserPool      *browser.Pool
	maxRetries       int
	retryBaseDelay   time.Duration
	enableScreenshot bool
	screenshotDir    string
	retryPolicy      types.RetryPolicy
}

// NewBase 创建基础上传器
func NewBase(platform string, cookiePath string, pool *browser.Pool) *Base {
	return &Base{
		platform:         platform,
		cookiePath:       cookiePath,
		browserPool:      pool,
		maxRetries:       3,
		retryBaseDelay:   2 * time.Second,
		enableScreenshot: false,
		screenshotDir:    "./screenshots",
		retryPolicy:      types.DefaultRetryPolicy(),
	}
}

// NewBaseWithScreenshot 创建带截图配置的上传器
func NewBaseWithScreenshot(platform string, cookiePath string, pool *browser.Pool, enableScreenshot bool, screenshotDir string) *Base {
	base := NewBase(platform, cookiePath, pool)
	base.enableScreenshot = enableScreenshot
	if screenshotDir != "" {
		base.screenshotDir = screenshotDir
	}
	return base
}

// SetMaxRetries 设置最大重试次数
func (b *Base) SetMaxRetries(retries int) {
	b.maxRetries = retries
	b.retryPolicy.MaxRetries = retries
}

// SetRetryBaseDelay 设置重试基础延迟
func (b *Base) SetRetryBaseDelay(delay time.Duration) {
	b.retryBaseDelay = delay
	b.retryPolicy.InitialDelay = delay.Milliseconds()
}

// SetRetryPolicy 设置重试策略
func (b *Base) SetRetryPolicy(policy types.RetryPolicy) {
	b.retryPolicy = policy
}

// SetScreenshotDir 设置截图目录
func (b *Base) SetScreenshotDir(dir string) {
	b.screenshotDir = dir
}

// EnableScreenshot 启用/禁用截图
func (b *Base) EnableScreenshot(enable bool) {
	b.enableScreenshot = enable
}

// Execute 执行上传（带增强错误处理和重试机制）
func (b *Base) Execute(ctx context.Context, task *types.VideoTask, steps []StepFunc) error {
	uploadCtx := &Context{
		Ctx:              ctx,
		Task:             task,
		stepResults:      make([]StepResult, 0),
		maxRetries:       b.maxRetries,
		screenshotDir:    b.screenshotDir,
		enableScreenshot: b.enableScreenshot,
		retryPolicy:      b.retryPolicy,
	}

	utils.Info(fmt.Sprintf("[+] 开始上传任务 - 平台: %s, 视频: %s", b.platform, task.VideoPath))
	uploadCtx.ReportProgress(StepInit, 5, "初始化上传任务")

	for i, step := range steps {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// 检查页面是否已关闭
		if uploadCtx.BrowserCtx != nil && uploadCtx.BrowserCtx.IsPageClosed() {
			utils.Error("[-] 页面已被关闭，上传中断")
			return types.NewUnrecoverableError("Execute", "页面被用户关闭", nil)
		}

		result := step(uploadCtx)
		uploadCtx.stepResults = append(uploadCtx.stepResults, result)

		if !result.Success {
			// 截图记录错误状态
			screenshotPath := uploadCtx.TakeScreenshot(fmt.Sprintf("error_step%d", i))
			if screenshotPath != "" {
				utils.Info(fmt.Sprintf("[-] 错误截图已保存: %s", screenshotPath))
			}

			// 检查是否是页面关闭导致的错误
			if uploadCtx.BrowserCtx != nil && uploadCtx.BrowserCtx.IsPageClosed() {
				utils.Error("[-] 页面已被关闭，上传中断")
				return types.NewUnrecoverableError("Execute", "页面被用户关闭", nil)
			}

			// 使用新的错误分类机制
			classifiedErr := types.ClassifyError(result.Error, result.Step.String())

			// 检查是否需要重试
			if shouldRetry, delay := b.shouldRetry(classifiedErr, uploadCtx.retryCount); shouldRetry {
				uploadCtx.retryCount++
				utils.Warn(fmt.Sprintf("[-] 步骤 %v 失败，第%d次重试，等待 %v...", result.Step, uploadCtx.retryCount, delay))
				utils.Warn(fmt.Sprintf("[-] 错误类型: %s, 原因: %s", classifiedErr.Type, classifiedErr.Message))
				uploadCtx.ReportProgress(result.Step, b.calculateProgress(result.Step), fmt.Sprintf("步骤失败，第%d次重试", uploadCtx.retryCount))
				time.Sleep(delay)
				continue
			}

			// 不可重试，返回错误
			utils.Error(fmt.Sprintf("[-] 上传失败 - 步骤: %v, 错误类型: %s, 错误: %v", result.Step, classifiedErr.Type, classifiedErr.Error()))
			uploadCtx.ReportProgress(result.Step, b.calculateProgress(result.Step), fmt.Sprintf("上传失败: %v", classifiedErr.Message))
			return classifiedErr
		}

		// 重置重试计数
		uploadCtx.retryCount = 0

		// 发送进度日志
		progress := b.calculateProgress(result.Step)
		uploadCtx.ReportProgress(result.Step, progress, fmt.Sprintf("步骤 %v 完成", result.Step))
		utils.Info(fmt.Sprintf("[-] 上传进度: %d%% - 步骤: %v", progress, result.Step))
	}

	utils.Info(fmt.Sprintf("[+] 上传完成 - 平台: %s", b.platform))
	uploadCtx.ReportProgress(StepComplete, 100, "上传完成")

	// 上传成功后，等待一段时间确保后台处理完成，然后关闭页面
	if uploadCtx.BrowserCtx != nil {
		utils.Info(fmt.Sprintf("[-] 等待10秒确保后台处理完成 - 平台: %s", b.platform))
		time.Sleep(10 * time.Second)
		utils.Info(fmt.Sprintf("[-] 正在关闭浏览器页面 - 平台: %s", b.platform))
		uploadCtx.BrowserCtx.ClosePage()
	}

	return nil
}

// shouldRetry 判断是否应该重试（使用新的错误分类机制）
func (b *Base) shouldRetry(err error, retryCount int) (bool, time.Duration) {
	if retryCount >= b.retryPolicy.MaxRetries {
		return false, 0
	}

	// 检查是否是 UploadError
	if uploadErr, ok := types.IsUploadError(err); ok {
		if !uploadErr.Retryable {
			return false, 0
		}

		// 根据错误类型调整重试策略
		policy := b.retryPolicy
		if uploadErr.Type == types.UploadErrorTypeRateLimited {
			// 限流错误使用更保守的策略
			policy = types.ConservativeRetryPolicy()
		}

		delay := policy.CalculateRetryDelay(retryCount + 1)
		return true, time.Duration(delay) * time.Millisecond
	}

	// 对于非 UploadError，默认不重试
	return false, 0
}

// calculateRetryDelay 计算重试延迟（指数退避 + 抖动）
func (b *Base) calculateRetryDelay(retryCount int) time.Duration {
	// 使用新的重试策略
	delay := b.retryPolicy.CalculateRetryDelay(retryCount)
	return time.Duration(delay) * time.Millisecond
}

// StepNavigate 导航步骤（增强版，带更好的错误处理）
func (b *Base) StepNavigate(url string) StepFunc {
	return func(ctx *Context) StepResult {
		ctx.ReportProgress(StepNavigate, 10, "正在打开页面...")

		// 获取浏览器上下文
		browserCtx, err := b.browserPool.GetContext(ctx.Ctx, b.cookiePath, b.getContextOptions())
		if err != nil {
			return StepResult{Step: StepNavigate, Success: false, Error: types.NewNetworkError("StepNavigate", err)}
		}
		ctx.BrowserCtx = browserCtx

		page, err := browserCtx.GetPage()
		if err != nil {
			return StepResult{Step: StepNavigate, Success: false, Error: types.NewNetworkError("StepNavigate", err)}
		}
		ctx.Page = page

		ctx.ReportProgress(StepNavigate, 15, fmt.Sprintf("导航到: %s", url))
		
		// 设置超时
		timeout := 30 * time.Second
		if _, err := page.Goto(url, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateNetworkidle,
			Timeout:   playwright.Float(float64(timeout.Milliseconds())),
		}); err != nil {
			return StepResult{Step: StepNavigate, Success: false, Error: types.NewTimeoutError("StepNavigate", err)}
		}

		// 等待页面加载
		time.Sleep(2 * time.Second)

		// 截图记录
		ctx.TakeScreenshot("navigate_complete")

		return StepResult{Step: StepNavigate, Success: true}
	}
}

// StepUploadVideo 上传视频步骤（改进版，支持错误重试）
func (b *Base) StepUploadVideo(inputSelector string, successSelectors []string, errorSelectors []string) StepFunc {
	return func(ctx *Context) StepResult {
		ctx.ReportProgress(StepUploadMedia, 20, "开始上传视频...")

		// 设置输入文件
		input := ctx.Page.Locator(inputSelector)
		if err := input.SetInputFiles(ctx.Task.VideoPath); err != nil {
			return StepResult{Step: StepUploadMedia, Success: false, Error: types.NewSelectorError("StepUploadVideo", inputSelector, err)}
		}

		ctx.ReportProgress(StepUploadMedia, 25, "视频文件已选择，等待上传...")

		// 等待上传完成
		timeout := time.After(5 * time.Minute)
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		checkCount := 0
		for {
			select {
			case <-timeout:
				ctx.TakeScreenshot("upload_timeout")
				return StepResult{Step: StepUploadMedia, Success: false, Error: types.NewTimeoutError("StepUploadVideo", fmt.Errorf("upload timeout after 5 minutes"))}
			case <-ticker.C:
				checkCount++

				// 检查成功标志
				for _, selector := range successSelectors {
					count, _ := ctx.Page.Locator(selector).Count()
					if count > 0 {
						ctx.ReportProgress(StepUploadMedia, 40, "视频上传成功")
						ctx.TakeScreenshot("upload_success")
						return StepResult{Step: StepUploadMedia, Success: true}
					}
				}

				// 检查上传失败
				for _, selector := range errorSelectors {
					count, _ := ctx.Page.Locator(selector).Count()
					if count > 0 {
						ctx.TakeScreenshot("upload_error")
						text, _ := ctx.Page.Locator(selector).TextContent()
						return StepResult{Step: StepUploadMedia, Success: false, Error: types.NewUploadError_("StepUploadVideo", fmt.Errorf("upload failed: %s", text))}
					}
				}

				// 每10秒报告一次进度
				if checkCount%5 == 0 {
					progress := 25 + (checkCount/5)*3
					if progress > 38 {
						progress = 38
					}
					ctx.ReportProgress(StepUploadMedia, progress, "正在上传视频中...")
					utils.Info("[-] 正在上传视频中...")
				}
			}
		}
	}
}

// StepFillTitle 填写标题步骤（兼容新旧页面结构，带长度限制）
func (b *Base) StepFillTitle(newSelector string, oldSelector string, maxLength int) StepFunc {
	return func(ctx *Context) StepResult {
		ctx.ReportProgress(StepFillTitle, 45, "正在填写标题...")

		title := ctx.Task.Title
		if len(title) > maxLength {
			title = title[:maxLength]
			utils.Warn(fmt.Sprintf("[-] 标题长度超过%d，已截断", maxLength))
		}

		// 尝试新页面结构
		newInput := ctx.Page.Locator(newSelector)
		newCount, _ := newInput.Count()
		if newCount > 0 {
			if err := newInput.Fill(title); err != nil {
				return StepResult{Step: StepFillTitle, Success: false, Error: types.NewSelectorError("StepFillTitle", newSelector, err)}
			}
			ctx.ReportProgress(StepFillTitle, 50, "标题填写完成（新页面结构）")
			return StepResult{Step: StepFillTitle, Success: true, Data: map[string]interface{}{"type": "new"}}
		}

		// 尝试旧页面结构
		oldInput := ctx.Page.Locator(oldSelector)
		oldCount, _ := oldInput.Count()
		if oldCount > 0 {
			if err := oldInput.Click(); err != nil {
				return StepResult{Step: StepFillTitle, Success: false, Error: types.NewSelectorError("StepFillTitle", oldSelector, err)}
			}
			time.Sleep(500 * time.Millisecond)

			// 清空原有内容
			if err := ctx.Page.Keyboard().Press("Backspace"); err != nil {
				return StepResult{Step: StepFillTitle, Success: false, Error: err}
			}
			if err := ctx.Page.Keyboard().Press("Control+KeyA"); err != nil {
				return StepResult{Step: StepFillTitle, Success: false, Error: err}
			}
			if err := ctx.Page.Keyboard().Press("Delete"); err != nil {
				return StepResult{Step: StepFillTitle, Success: false, Error: err}
			}

			// 输入新标题
			if err := ctx.Page.Keyboard().Type(title); err != nil {
				return StepResult{Step: StepFillTitle, Success: false, Error: err}
			}
			if err := ctx.Page.Keyboard().Press("Enter"); err != nil {
				return StepResult{Step: StepFillTitle, Success: false, Error: err}
			}
			ctx.ReportProgress(StepFillTitle, 50, "标题填写完成（旧页面结构）")
			return StepResult{Step: StepFillTitle, Success: true, Data: map[string]interface{}{"type": "old"}}
		}

		return StepResult{Step: StepFillTitle, Success: false, Error: types.NewSelectorError("StepFillTitle", fmt.Sprintf("new: %s, old: %s", newSelector, oldSelector), fmt.Errorf("title input not found"))}
	}
}

// StepAddTags 添加标签步骤
func (b *Base) StepAddTags(contentEditableSelector string) StepFunc {
	return func(ctx *Context) StepResult {
		ctx.ReportProgress(StepAddTags, 52, "正在添加标签...")

		// 点击内容编辑区域
		editor := ctx.Page.Locator(contentEditableSelector)
		if err := editor.Click(); err != nil {
			return StepResult{Step: StepAddTags, Success: false, Error: types.NewSelectorError("StepAddTags", contentEditableSelector, err)}
		}
		time.Sleep(300 * time.Millisecond)

		for _, tag := range ctx.Task.Tags {
			if err := ctx.Page.Keyboard().Type("#" + tag + " "); err != nil {
				return StepResult{Step: StepAddTags, Success: false, Error: err}
			}
			time.Sleep(500 * time.Millisecond)
		}

		ctx.ReportProgress(StepAddTags, 55, fmt.Sprintf("已添加 %d 个标签", len(ctx.Task.Tags)))
		return StepResult{Step: StepAddTags, Success: true}
	}
}

// StepClickPublish 点击发布步骤（循环重试版，20秒间隔）
func (b *Base) StepClickPublish(buttonSelector string, successSelectors []string, successURLs []string) StepFunc {
	return func(ctx *Context) StepResult {
		ctx.ReportProgress(StepPublish, 85, "正在发布...")

		retryCount := 0
		maxRetries := 10 // 最多重试10次
		retryInterval := 20 * time.Second

		for retryCount <= maxRetries {
			if retryCount > 0 {
				ctx.ReportProgress(StepPublish, 85, fmt.Sprintf("第%d次尝试发布...", retryCount+1))
				utils.Info(fmt.Sprintf("[-] 第%d次尝试发布，等待%.0f秒...", retryCount+1, retryInterval.Seconds()))
				time.Sleep(retryInterval)
			}

			// 检查页面是否已关闭
			if ctx.BrowserCtx != nil && ctx.BrowserCtx.IsPageClosed() {
				utils.Error("[-] 页面已被关闭，发布中断")
				return StepResult{Step: StepPublish, Success: false, Error: types.NewUnrecoverableError("StepClickPublish", "页面被用户关闭", nil)}
			}

			// 1. 点击发布按钮
			button := ctx.Page.Locator(buttonSelector)
			if err := button.Click(); err != nil {
				utils.Warn(fmt.Sprintf("[-] 点击发布按钮失败: %v", err))
				retryCount++
				continue
			}

			ctx.ReportProgress(StepPublish, 88, "等待检测完成...")

			// 2. 等待检测完成（给检测留出时间）
			time.Sleep(3 * time.Second)

			// 3. 检查是否需要确认发布
			confirmSelectors := []string{
				"text=确认发布",
				"button:has-text('确认')",
				"[class*='confirm']",
			}
			for _, selector := range confirmSelectors {
				confirmBtn := ctx.Page.Locator(selector)
				if count, _ := confirmBtn.Count(); count > 0 {
					visible, _ := confirmBtn.IsVisible()
					if visible {
						ctx.ReportProgress(StepPublish, 90, "检测完成，确认发布...")
						if err := confirmBtn.Click(); err != nil {
							utils.Warn(fmt.Sprintf("[-] 点击确认发布失败: %v", err))
						}
						break
					}
				}
			}

			// 4. 检查发布成功标志
			for _, selector := range successSelectors {
				count, _ := ctx.Page.Locator(selector).Count()
				if count > 0 {
					visible, _ := ctx.Page.Locator(selector).IsVisible()
					if visible {
						ctx.ReportProgress(StepPublish, 95, "发布成功")
						ctx.TakeScreenshot("publish_success")
						return StepResult{Step: StepPublish, Success: true}
					}
				}
			}

			// 5. 检查是否跳转到成功页面
			url := ctx.Page.URL()
			for _, successURL := range successURLs {
				if strings.Contains(url, successURL) {
					ctx.ReportProgress(StepPublish, 95, "发布成功（URL匹配）")
					ctx.TakeScreenshot("publish_success")
					return StepResult{Step: StepPublish, Success: true}
				}
			}

			// 6. 检查错误信息
			errorSelectors := []string{
				"text=发布失败",
				"text=上传失败",
				".error-message",
				"[class*='error']",
				"text=检测失败",
				"text=审核不通过",
			}
			for _, selector := range errorSelectors {
				count, _ := ctx.Page.Locator(selector).Count()
				if count > 0 {
					visible, _ := ctx.Page.Locator(selector).IsVisible()
					if visible {
						text, _ := ctx.Page.Locator(selector).TextContent()
						ctx.TakeScreenshot("publish_error")
						return StepResult{Step: StepPublish, Success: false, Error: types.NewPlatformError("StepClickPublish", text, nil)}
					}
				}
			}

			// 未成功，增加重试计数
			retryCount++
			utils.Info(fmt.Sprintf("[-] 第%d次发布尝试未成功，准备重试...", retryCount))
		}

		// 超过最大重试次数
		ctx.TakeScreenshot("publish_max_retry")
		return StepResult{Step: StepPublish, Success: false, Error: types.NewPlatformError("StepClickPublish", fmt.Sprintf("发布失败，已重试%d次", maxRetries), nil)}
	}
}

// StepSetSchedule 设置定时发布步骤（统一时间格式处理）
func (b *Base) StepSetSchedule(checkboxSelector string, inputSelector string, timeFormat string) StepFunc {
	return func(ctx *Context) StepResult {
		if ctx.Task.ScheduleTime == nil || *ctx.Task.ScheduleTime == "" {
			return StepResult{Step: StepSetSchedule, Success: true}
		}

		ctx.ReportProgress(StepSetSchedule, 75, "正在设置定时发布...")

		// 解析时间并转换为平台特定格式
		scheduleTime, err := parseScheduleTime(*ctx.Task.ScheduleTime)
		if err != nil {
			return StepResult{Step: StepSetSchedule, Success: false, Error: types.NewValidationError("StepSetSchedule", "定时时间格式无效", err)}
		}

		// 点击定时发布复选框
		if err := ctx.Page.Locator(checkboxSelector).Click(); err != nil {
			return StepResult{Step: StepSetSchedule, Success: false, Error: types.NewSelectorError("StepSetSchedule", checkboxSelector, err)}
		}
		time.Sleep(1 * time.Second)

		// 设置日期时间
		scheduleInput := ctx.Page.Locator(inputSelector)
		if err := scheduleInput.Click(); err != nil {
			return StepResult{Step: StepSetSchedule, Success: false, Error: types.NewSelectorError("StepSetSchedule", inputSelector, err)}
		}
		if err := ctx.Page.Keyboard().Press("Control+KeyA"); err != nil {
			return StepResult{Step: StepSetSchedule, Success: false, Error: err}
		}

		// 使用平台特定格式
		formattedTime := scheduleTime.Format(timeFormat)
		if err := ctx.Page.Keyboard().Type(formattedTime); err != nil {
			return StepResult{Step: StepSetSchedule, Success: false, Error: err}
		}
		if err := ctx.Page.Keyboard().Press("Enter"); err != nil {
			return StepResult{Step: StepSetSchedule, Success: false, Error: err}
		}

		ctx.ReportProgress(StepSetSchedule, 80, "定时发布设置完成")
		return StepResult{Step: StepSetSchedule, Success: true}
	}
}

// parseScheduleTime 解析定时时间（支持多种格式）
func parseScheduleTime(timeStr string) (time.Time, error) {
	formats := []string{
		"2006-01-02 15:04",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04",
		"2006-01-02T15:04:05",
		"2006/01/02 15:04",
		"2006/01/02 15:04:05",
	}

	for _, format := range formats {
		if t, err := time.ParseInLocation(format, timeStr, time.Local); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("无法解析时间: %s", timeStr)
}

// Release 释放资源
func (b *Base) Release(ctx *Context) {
	if ctx != nil && ctx.BrowserCtx != nil {
		ctx.BrowserCtx.Release()
	}
}

// canRetry 判断步骤是否可以重试（已废弃，使用 shouldRetry 替代）
func (b *Base) canRetry(step Step) bool {
	switch step {
	case StepNavigate, StepUploadMedia, StepPublish:
		return true
	default:
		return false
	}
}

// calculateProgress 计算进度
func (b *Base) calculateProgress(currentStep Step) int {
	totalSteps := 10
	return int(currentStep) * 100 / totalSteps
}

// GetCookiePath 获取Cookie路径（公共方法）
func (b *Base) GetCookiePath() string {
	return b.cookiePath
}

// GetContextOptions 获取上下文选项（公共方法）
func (b *Base) GetContextOptions() *browser.ContextOptions {
	return &browser.ContextOptions{
		UserAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		Viewport:    &playwright.Size{Width: 1920, Height: 1080},
		Locale:      "zh-CN",
		TimezoneId:  "Asia/Shanghai",
		Geolocation: &playwright.Geolocation{Latitude: 39.9042, Longitude: 116.4074},
		ExtraHeaders: map[string]string{
			"Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8",
		},
	}
}

// getContextOptions 获取上下文选项（私有方法，保持向后兼容）
func (b *Base) getContextOptions() *browser.ContextOptions {
	return b.GetContextOptions()
}

// init 初始化随机种子
func init() {
	rand.Seed(time.Now().UnixNano())
}
