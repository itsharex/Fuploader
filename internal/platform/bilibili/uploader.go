package bilibili

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"Fuploader/internal/config"
	"Fuploader/internal/platform/browser"
	"Fuploader/internal/types"
	"Fuploader/internal/utils"

	"github.com/playwright-community/playwright-go"
)

// debugLog 调试日志输出，仅在调试模式下显示
func debugLog(format string, args ...interface{}) {
	if config.Config != nil && config.Config.DebugMode {
		utils.InfoWithPlatform("bilibili", fmt.Sprintf("[调试] "+format, args...))
	}
}

// browserPool 全局浏览器池实例
var browserPool *browser.Pool

func init() {
	browserPool = browser.NewPool(2, 5)
}

// Uploader B站上传器
type Uploader struct {
	accountID  uint
	cookiePath string
	platform   string
}

// NewUploader 创建上传器（兼容旧版）
func NewUploader(cookiePath string) *Uploader {
	u := &Uploader{
		accountID:  0,
		cookiePath: cookiePath,
		platform:   "bilibili",
	}
	debugLog("创建上传器 - 地址: %p, cookiePath: '%s'", u, cookiePath)
	if cookiePath == "" {
		utils.Warn("[Bilibili] NewUploader 收到空的cookiePath!")
	}
	return u
}

// NewUploaderWithAccount 创建带accountID的上传器（新接口）
func NewUploaderWithAccount(accountID uint) *Uploader {
	cookiePath := config.GetCookiePath("bilibili", int(accountID))
	u := &Uploader{
		accountID:  accountID,
		cookiePath: cookiePath,
		platform:   "bilibili",
	}
	debugLog("创建上传器 - 地址: %p, accountID: %d, cookiePath: '%s'", u, accountID, cookiePath)
	return u
}

// Platform 返回平台名称
func (u *Uploader) Platform() string {
	return u.platform
}

// ValidateCookie 验证Cookie是否有效
func (u *Uploader) ValidateCookie(ctx context.Context) (bool, error) {
	utils.InfoWithPlatform(u.platform, "验证Cookie")

	if _, err := os.Stat(u.cookiePath); os.IsNotExist(err) {
		utils.WarnWithPlatform(u.platform, "失败: 验证Cookie - Cookie文件不存在")
		return false, nil
	}

	// 使用accountID获取上下文（如果accountID为0则退化为旧逻辑）
	browserCtx, err := browserPool.GetContextByAccount(ctx, u.accountID, u.cookiePath, nil)
	if err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 验证Cookie - 获取浏览器失败: %v", err))
		return false, nil
	}
	defer browserCtx.Release()

	page, err := browserCtx.GetPage()
	if err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 验证Cookie - 获取页面失败: %v", err))
		return false, nil
	}

	utils.InfoWithPlatform(u.platform, "正在打开发布页面...")
	if _, err := page.Goto("https://member.bilibili.com/platform/upload/video/frame", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 验证Cookie - 打开页面失败: %v", err))
		return false, nil
	}

	time.Sleep(3 * time.Second)

	url := page.URL()
	if strings.Contains(url, "member.bilibili.com/platform/home") ||
		strings.Contains(url, "member.bilibili.com/platform/upload") {
		utils.SuccessWithPlatform(u.platform, "Cookie有效")
		return true, nil
	}

	utils.WarnWithPlatform(u.platform, "失败: 验证Cookie - Cookie已失效")
	return false, nil
}

// setScheduleTime 设置定时发布时间
// 支持格式: "2006-01-02 15:04" 或 "2006-01-02T15:04:05" (ISO8601)
func (u *Uploader) setScheduleTime(page playwright.Page, scheduleTime string) error {
	// 解析时间，支持多种格式
	var targetTime time.Time
	var err error

	// 尝试解析标准格式
	targetTime, err = time.Parse("2006-01-02 15:04", scheduleTime)
	if err != nil {
		// 尝试解析ISO8601格式 (2026-02-12T00:00:00)
		targetTime, err = time.Parse("2006-01-02T15:04:05", scheduleTime)
		if err != nil {
			// 尝试解析带时区的ISO8601格式
			targetTime, err = time.Parse(time.RFC3339, scheduleTime)
			if err != nil {
				return fmt.Errorf("失败: 设置定时发布 - 解析时间失败，不支持的时间格式: %w", err)
			}
		}
	}

	// 验证时间范围（≥2小时且≤15天）
	now := time.Now()
	minTime := now.Add(2 * time.Hour)
	maxTime := now.Add(15 * 24 * time.Hour)

	if targetTime.Before(minTime) {
		return fmt.Errorf("失败: 设置定时发布 - 定时时间必须至少提前2小时")
	}
	if targetTime.After(maxTime) {
		return fmt.Errorf("失败: 设置定时发布 - 定时时间不能超过15天")
	}

	// 1. 开启定时开关
	switchContainer := page.Locator(`div.switch-container.switch-container-active`).First()
	if err := switchContainer.WaitFor(playwright.LocatorWaitForOptions{Timeout: playwright.Float(3000)}); err != nil {
		return fmt.Errorf("失败: 设置定时发布 - 未找到定时开关: %w", err)
	}

	if err := switchContainer.Click(); err != nil {
		return fmt.Errorf("失败: 设置定时发布 - 点击定时开关失败: %w", err)
	}
	time.Sleep(1 * time.Second)

	// 2. 选择日期
	datePicker := page.Locator(`div.date-picker-date-wrp`).First()
	if err := datePicker.WaitFor(playwright.LocatorWaitForOptions{Timeout: playwright.Float(3000)}); err != nil {
		return fmt.Errorf("失败: 设置定时发布 - 未找到日期选择器: %w", err)
	}

	// 点击展开日期面板
	if err := datePicker.Click(); err != nil {
		return fmt.Errorf("失败: 设置定时发布 - 点击日期选择器失败: %w", err)
	}
	time.Sleep(500 * time.Millisecond)

	// 使用 aria-label 选择目标日期
	dateStr := targetTime.Format("2006-01-02")
	dateCell := page.Locator(fmt.Sprintf(`div[aria-label="%s"]`, dateStr)).First()
	if err := dateCell.WaitFor(playwright.LocatorWaitForOptions{Timeout: playwright.Float(3000)}); err != nil {
		return fmt.Errorf("失败: 设置定时发布 - 未找到目标日期: %w", err)
	}

	if err := dateCell.Click(); err != nil {
		return fmt.Errorf("失败: 设置定时发布 - 选择日期失败: %w", err)
	}
	time.Sleep(500 * time.Millisecond)

	// 3. 选择时间
	timePicker := page.Locator(`div.date-picker-timer`).First()
	if err := timePicker.WaitFor(playwright.LocatorWaitForOptions{Timeout: playwright.Float(3000)}); err != nil {
		return fmt.Errorf("失败: 设置定时发布 - 未找到时间选择器: %w", err)
	}

	// 点击展开时间面板
	if err := timePicker.Click(); err != nil {
		return fmt.Errorf("失败: 设置定时发布 - 点击时间选择器失败: %w", err)
	}
	time.Sleep(500 * time.Millisecond)

	// 使用 aria-label 选择目标时间
	timeStr := targetTime.Format("15:04")
	timeCell := page.Locator(fmt.Sprintf(`div[aria-label="%s"]`, timeStr)).First()
	if err := timeCell.WaitFor(playwright.LocatorWaitForOptions{Timeout: playwright.Float(3000)}); err != nil {
		return fmt.Errorf("失败: 设置定时发布 - 未找到目标时间: %w", err)
	}

	if err := timeCell.Click(); err != nil {
		return fmt.Errorf("失败: 设置定时发布 - 选择时间失败: %w", err)
	}
	time.Sleep(500 * time.Millisecond)

	return nil
}

// Upload 上传视频
func (u *Uploader) Upload(ctx context.Context, task *types.VideoTask) error {
	utils.InfoWithPlatform(u.platform, fmt.Sprintf("开始上传: %s", task.VideoPath))

	if _, err := os.Stat(task.VideoPath); err != nil {
		return fmt.Errorf("失败: 开始上传 - 视频文件不存在: %w", err)
	}

	// 使用accountID获取上下文（如果accountID为0则退化为旧逻辑）
	browserCtx, err := browserPool.GetContextByAccount(ctx, u.accountID, u.cookiePath, nil)
	if err != nil {
		return fmt.Errorf("失败: 开始上传 - 获取浏览器失败: %w", err)
	}
	defer browserCtx.Release()

	page, err := browserCtx.GetPage()
	if err != nil {
		return fmt.Errorf("失败: 开始上传 - 获取页面失败: %w", err)
	}

	utils.InfoWithPlatform(u.platform, "正在打开发布页面...")
	if _, err := page.Goto("https://member.bilibili.com/platform/upload/video/frame", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		return fmt.Errorf("失败: 开始上传 - 打开页面失败: %w", err)
	}

	if err := page.Locator(`input[type="file"]`).First().WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		utils.WarnWithPlatform(u.platform, "失败: 等待文件输入框 - 超时")
	}

	utils.InfoWithPlatform(u.platform, "正在上传视频...")

	fileInput := page.Locator(`input[type="file"][accept*="video/mp4"]`).First()
	if count, _ := fileInput.Count(); count == 0 {
		fileInput = page.Locator(`div.bcc-upload-wrapper input[type="file"]`).First()
	}
	if count, _ := fileInput.Count(); count == 0 {
		fileInput = page.Locator(`input[type="file"]`).First()
	}

	if count, _ := fileInput.Count(); count == 0 {
		return fmt.Errorf("未找到文件上传输入框")
	}

	type result struct {
		err error
	}
	done := make(chan result, 1)

	go func() {
		err := fileInput.SetInputFiles(task.VideoPath)
		done <- result{err: err}
	}()

	select {
	case res := <-done:
		if res.err != nil {
			return fmt.Errorf("失败: 选择视频文件 - %w", res.err)
		}
	case <-time.After(30 * time.Second):
		return fmt.Errorf("失败: 选择视频文件 - 超时")
	}

	utils.InfoWithPlatform(u.platform, "等待视频上传完成...")

	uploadTimeout := 10 * time.Minute
	uploadCheckInterval := 2 * time.Second
	uploadStartTime := time.Now()
	uploadCompleted := false
	lastProgressCount := -1
	stuckCount := 0

	for time.Since(uploadStartTime) < uploadTimeout {
		select {
		case <-ctx.Done():
			return fmt.Errorf("上传已取消")
		default:
		}

		if browserCtx.IsPageClosed() {
			return fmt.Errorf("浏览器已关闭")
		}

		progressBar := page.Locator(`.bcc-upload-progress, .upload-progress, [class*="progress"]`).First()
		uploadSuccessText := page.Locator(`text=/上传完成|转码中|处理中|视频上传成功/`).First()
		uploadDoneIcon := page.Locator(`.upload-done, .upload-success, [class*="success"]`).First()

		progressCount, _ := progressBar.Count()
		successCount, _ := uploadSuccessText.Count()
		doneCount, _ := uploadDoneIcon.Count()

		if config.Config != nil && config.Config.DebugMode {
			utils.InfoWithPlatform(u.platform, fmt.Sprintf("[调试] 进度条: %d, 成功文本: %d, 完成图标: %d", progressCount, successCount, doneCount))
		}

		if (progressCount == 0 && successCount > 0) || doneCount > 0 {
			utils.SuccessWithPlatform(u.platform, "视频上传完成")
			uploadCompleted = true
			break
		}

		if progressCount == lastProgressCount {
			stuckCount++
			if progressCount == 0 && stuckCount >= 3 {
				utils.SuccessWithPlatform(u.platform, "视频上传完成")
				uploadCompleted = true
				break
			}
		} else {
			stuckCount = 0
		}
		lastProgressCount = progressCount

		uploadError := page.Locator(`text=/上传失败|错误|失败|Upload failed/`).First()
		if count, _ := uploadError.Count(); count > 0 {
			return fmt.Errorf("失败: 视频上传 - 上传失败")
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("上传已取消")
		case <-time.After(uploadCheckInterval):
		}
	}

	if !uploadCompleted {
		return fmt.Errorf("失败: 视频上传 - 超时")
	}

	time.Sleep(2 * time.Second)

	// 设置转载类型（自制/转载）
	if task.Copyright != "" {
		utils.InfoWithPlatform(u.platform, fmt.Sprintf("设置转载类型: %s", task.Copyright))
		var copyrightText string
		if task.Copyright == "1" {
			copyrightText = "自制"
		} else if task.Copyright == "2" {
			copyrightText = "转载"
		}
		if copyrightText != "" {
			copyrightLocator := page.Locator(fmt.Sprintf(`span:has-text("%s")`, copyrightText)).First()
			if err := copyrightLocator.WaitFor(playwright.LocatorWaitForOptions{Timeout: playwright.Float(3000)}); err != nil {
				utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 设置转载类型 - 等待%s选项超时: %v", copyrightText, err))
			} else if count, _ := copyrightLocator.Count(); count > 0 {
				if err := copyrightLocator.Click(); err != nil {
					utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 设置转载类型 - 点击%s选项失败: %v", copyrightText, err))
				} else {
					utils.InfoWithPlatform(u.platform, fmt.Sprintf("已选择%s", copyrightText))
				}
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	if task.Title != "" {
		utils.InfoWithPlatform(u.platform, "填写标题...")
		titleInput := page.Locator(`input[type="text"][placeholder="请输入稿件标题"]`).First()
		if err := titleInput.WaitFor(playwright.LocatorWaitForOptions{Timeout: playwright.Float(3000)}); err != nil {
			titleInput = page.Locator(`div.video-title-container input[type="text"]`).First()
		}
		if count, _ := titleInput.Count(); count > 0 {
			if err := titleInput.Fill(task.Title); err != nil {
				utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 填写标题 - %v", err))
			} else {
				utils.InfoWithPlatform(u.platform, fmt.Sprintf("标题已填写: %s", task.Title))
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	// 处理标签：先删除默认标签，再添加用户标签
	utils.InfoWithPlatform(u.platform, "添加标签...")

	// 1. 删除默认标签（从第一个开始删除，避免索引错位）
	for {
		// 精准定位：svg.close.icon-sprite.icon-sprite-off
		tagCloseBtn := page.Locator(`svg.close.icon-sprite.icon-sprite-off`).First()
		if count, _ := tagCloseBtn.Count(); count == 0 {
			// 兜底：层级定位 div.label-item-v2-container >> svg.close
			tagCloseBtn = page.Locator(`div.label-item-v2-container >> svg.close`).First()
		}
		if count, _ := tagCloseBtn.Count(); count == 0 {
			break // 没有更多标签需要删除
		}
		if err := tagCloseBtn.Click(); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 删除默认标签 - %v", err))
			break
		}
		time.Sleep(300 * time.Millisecond)
	}

	// 2. 添加用户标签
	if len(task.Tags) > 0 {
		tagInput := page.Locator(`div.tag-input-wrp >> input[type="text"]`).First()
		if err := tagInput.WaitFor(playwright.LocatorWaitForOptions{Timeout: playwright.Float(3000)}); err != nil {
			tagInput = page.Locator(`input[type="text"][placeholder="按回车键Enter创建标签"]`).First()
		}
		if count, _ := tagInput.Count(); count > 0 {
			for i, tag := range task.Tags {
				if err := tagInput.Fill(tag); err != nil {
					utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 输入标签[%d] - %v", i, err))
					continue
				}
				time.Sleep(300 * time.Millisecond)
				if err := tagInput.Press("Enter"); err != nil {
					utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 确认标签[%d] - %v", i, err))
				}
				time.Sleep(400 * time.Millisecond)
			}
			utils.InfoWithPlatform(u.platform, "标签添加完成")
		} else {
			utils.WarnWithPlatform(u.platform, "失败: 添加标签 - 未找到标签输入框")
		}
	}

	if task.Description != "" {
		utils.InfoWithPlatform(u.platform, "填写描述...")
		descEditor := page.Locator(`div.ql-editor[data-placeholder*="相关信息"]`).First()
		if err := descEditor.WaitFor(playwright.LocatorWaitForOptions{Timeout: playwright.Float(3000)}); err != nil {
			descEditor = page.Locator(`div.desc-text-wrp div.ql-editor`).First()
		}
		if count, _ := descEditor.Count(); count == 0 {
			descEditor = page.Locator(`div.archive-info-editor div.ql-editor`).First()
		}
		if count, _ := descEditor.Count(); count > 0 {
			if err := descEditor.Fill(task.Description); err != nil {
				utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 填写描述 - %v", err))
			} else {
				utils.InfoWithPlatform(u.platform, "描述已填写")
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	// 设置封面
	utils.InfoWithPlatform(u.platform, "设置封面...")
	// 先声明所有变量，避免 goto 跳过变量声明
	coverFilled := false
	var coverCheckStart time.Time
	var coverCheckTimeout time.Duration
	var confirmBtn playwright.Locator
	// 1. 点击封面区域打开弹窗
	coverMain := page.Locator(`div.cover-main >> span.edit-text:text("封面设置")`).First()
	if err := coverMain.WaitFor(playwright.LocatorWaitForOptions{Timeout: playwright.Float(3000)}); err != nil {
		coverMain = page.Locator(`div.cover-main`).First()
		if err := coverMain.WaitFor(playwright.LocatorWaitForOptions{Timeout: playwright.Float(3000)}); err != nil {
			utils.WarnWithPlatform(u.platform, "失败: 设置封面 - 未找到封面区域")
			goto CoverDone
		}
	}
	if err := coverMain.ScrollIntoViewIfNeeded(); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 设置封面 - 滚动到封面区域失败: %v", err))
	}
	if err := coverMain.Click(playwright.LocatorClickOptions{Force: playwright.Bool(true)}); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 设置封面 - 点击封面区域失败: %v", err))
		goto CoverDone
	}
	time.Sleep(2 * time.Second)

	// 2. 判断用户是否提供了自定义封面
	if task.Thumbnail != "" {
		// 用户提供了封面，需要上传
		// 直接定位隐藏的文件输入框，不点击按钮（避免弹出系统文件对话框）
		coverInput := page.Locator(`input[type="file"][accept="image/png, image/jpeg"]`).First()
		if count, _ := coverInput.Count(); count == 0 {
			// 兜底：更通用的选择器
			coverInput = page.Locator(`input[type="file"]`).First()
		}
		if count, _ := coverInput.Count(); count == 0 {
			utils.WarnWithPlatform(u.platform, "失败: 设置封面 - 未找到封面文件输入框")
			page.Keyboard().Press("Escape")
			time.Sleep(500 * time.Millisecond)
			goto CoverDone
		}
		if err := coverInput.SetInputFiles(task.Thumbnail); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 设置封面 - 上传封面失败: %v", err))
			page.Keyboard().Press("Escape")
			time.Sleep(500 * time.Millisecond)
			goto CoverDone
		}
		time.Sleep(3 * time.Second)
	}

	// 3. 点击完成关闭弹窗
	confirmBtn = page.Locator(`div.button.submit:text("完成")`).First()
	if err := confirmBtn.WaitFor(playwright.LocatorWaitForOptions{Timeout: playwright.Float(3000)}); err != nil {
		// 兜底：层级定位
		confirmBtn = page.Locator(`div.cover-editor-button >> div.button.submit`).First()
	}
	if count, _ := confirmBtn.Count(); count > 0 {
		if err := confirmBtn.Click(); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 设置封面 - 点击完成按钮失败: %v", err))
			page.Keyboard().Press("Escape")
		}
		time.Sleep(1 * time.Second)
	} else {
		utils.WarnWithPlatform(u.platform, "失败: 设置封面 - 未找到完成按钮")
		page.Keyboard().Press("Escape")
		time.Sleep(1 * time.Second)
	}

	// 4. 验证封面是否成功填充
	coverCheckStart = time.Now()
	coverCheckTimeout = 10 * time.Second
	// coverFilled 已在前面声明，这里直接赋值
	coverFilled = false
	for time.Since(coverCheckStart) < coverCheckTimeout {
		// 方式1：检查success类
		coverWithSuccess := page.Locator(`div.cover-main-img.success`).First()
		if count, _ := coverWithSuccess.Count(); count > 0 {
			if isVisible, _ := coverWithSuccess.IsVisible(); isVisible {
				coverFilled = true
				break
			}
		}
		// 方式2：检查背景图
		hasBackground, err := page.Evaluate(`() => {
			const cover = document.querySelector('div.cover-main-img');
			return cover && cover.style.backgroundImage && cover.style.backgroundImage !== '' && cover.style.backgroundImage !== 'none';
		}`)
		if err == nil && hasBackground.(bool) {
			coverFilled = true
			break
		}
		// 方式3：检查"封面设置"文本是否消失
		coverText := page.Locator(`div.cover-main >> span.edit-text:text("封面设置")`).First()
		if count, _ := coverText.Count(); count == 0 {
			coverFilled = true
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	if coverFilled {
		utils.InfoWithPlatform(u.platform, "封面设置完成")
	} else {
		utils.WarnWithPlatform(u.platform, "失败: 设置封面 - 封面设置可能未完成")
	}

CoverDone:

	// 设置定时发布
	if task.ScheduleTime != nil && *task.ScheduleTime != "" {
		if err := u.setScheduleTime(page, *task.ScheduleTime); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 设置定时发布 - %v", err))
		}
	}

	utils.InfoWithPlatform(u.platform, "准备发布...")
	// 投稿按钮定位 - 按优先级：文本精准 > 属性 > 层级
	submitBtn := page.Locator(`span.submit-add:text("立即投稿")`).First()
	if err := submitBtn.WaitFor(playwright.LocatorWaitForOptions{Timeout: playwright.Float(3000)}); err != nil {
		// 兜底：属性定位
		submitBtn = page.Locator(`span[data-reporter-id="89"].submit-add`).First()
		if err := submitBtn.WaitFor(playwright.LocatorWaitForOptions{Timeout: playwright.Float(3000)}); err != nil {
			// 兜底：层级定位
			submitBtn = page.Locator(`div.submit-container >> span.submit-add`).First()
			if err := submitBtn.WaitFor(playwright.LocatorWaitForOptions{Timeout: playwright.Float(3000)}); err != nil {
				utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 等待发布按钮 - 超时: %v", err))
			}
		}
	}
	if count, _ := submitBtn.Count(); count == 0 {
		return fmt.Errorf("失败: 准备发布 - 未找到发布按钮")
	}

	// 滚动到投稿按钮并确保可见
	if err := submitBtn.ScrollIntoViewIfNeeded(); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 准备发布 - 滚动到发布按钮失败: %v", err))
	}

	urlBeforeSubmit := page.URL()
	maxClickAttempts := 3
	clickAttempt := 0
	submitSuccess := false

	for clickAttempt < maxClickAttempts && !submitSuccess {
		select {
		case <-ctx.Done():
			return fmt.Errorf("失败: 发布 - 已取消")
		default:
		}

		clickAttempt++
		utils.InfoWithPlatform(u.platform, fmt.Sprintf("第%d次尝试发布...", clickAttempt))

		// 使用Force点击，确保即使元素被遮挡也能点击
		if err := submitBtn.Click(playwright.LocatorClickOptions{
			Force: playwright.Bool(true),
		}); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 点击发布按钮 - %v", err))
			select {
			case <-ctx.Done():
				return fmt.Errorf("失败: 发布 - 已取消")
			case <-time.After(2 * time.Second):
			}
			continue
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("失败: 发布 - 已取消")
		case <-time.After(3 * time.Second):
		}

		// 处理确认弹窗
		confirmDialogBtn := page.Locator(`button:has-text("确定"), button:has-text("确认")`).First()
		if count, _ := confirmDialogBtn.Count(); count > 0 {
			confirmDialogBtn.Click()
			select {
			case <-ctx.Done():
				return fmt.Errorf("失败: 发布 - 已取消")
			case <-time.After(2 * time.Second):
			}
		}

		// 检测投稿结果
		submitCheckTimeout := 60 * time.Second
		submitCheckInterval := 2 * time.Second
		submitCheckStart := time.Now()

		for time.Since(submitCheckStart) < submitCheckTimeout {
			select {
			case <-ctx.Done():
				return fmt.Errorf("失败: 发布 - 已取消")
			default:
			}

			if browserCtx.IsPageClosed() {
				return fmt.Errorf("失败: 发布 - 浏览器已关闭")
			}

			currentURL := page.URL()

			// 成功标志1：页面跳转到管理页或首页
			if strings.Contains(currentURL, "member.bilibili.com/platform/upload/manage") ||
				strings.Contains(currentURL, "member.bilibili.com/platform/home") {
				submitSuccess = true
				break
			}

			// 成功标志2：URL变化且不再包含frame
			if currentURL != urlBeforeSubmit && !strings.Contains(currentURL, "frame") {
				submitSuccess = true
				break
			}

			// 成功标志3：成功提示文本
			successToast := page.Locator(`text=/投稿成功|发布成功|提交成功|审核中|稿件已提交/`).First()
			if count, _ := successToast.Count(); count > 0 {
				text, _ := successToast.TextContent()
				if !strings.Contains(text, "投稿中") && !strings.Contains(text, "处理中") && !strings.Contains(text, "正在提交") {
					submitSuccess = true
					break
				}
			}

			// 成功标志4：投稿按钮消失
			submitBtnCount, _ := submitBtn.Count()
			if submitBtnCount == 0 {
				select {
				case <-ctx.Done():
					return fmt.Errorf("失败: 发布 - 已取消")
				case <-time.After(2 * time.Second):
				}
				// 再次确认按钮确实消失了
				if count, _ := submitBtn.Count(); count == 0 {
					submitSuccess = true
					break
				}
			}

			// 失败标志
			errorToast := page.Locator(`text=/投稿失败|发布失败|提交失败|错误|请完善/`).First()
			if count, _ := errorToast.Count(); count > 0 {
				text, _ := errorToast.TextContent()
				utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 发布 - %s", text))
				break
			}

			select {
			case <-ctx.Done():
				return fmt.Errorf("失败: 发布 - 已取消")
			case <-time.After(submitCheckInterval):
			}
		}

		if !submitSuccess && clickAttempt < maxClickAttempts {
			time.Sleep(3 * time.Second)
		}
	}

	if !submitSuccess {
		return fmt.Errorf("失败: 发布 - 发布失败")
	}

	utils.SuccessWithPlatform(u.platform, "发布成功")
	return nil
}

// Login 登录
func (u *Uploader) Login() error {
	debugLog("Login开始 - cookiePath: '%s'", u.cookiePath)
	if u.cookiePath == "" {
		return fmt.Errorf("失败: 登录 - cookie路径为空")
	}

	ctx := context.Background()

	// 登录时不使用accountID（因为是新登录）
	browserCtx, err := browserPool.GetContextByAccount(ctx, 0, "", nil)
	if err != nil {
		return fmt.Errorf("失败: 登录 - 获取浏览器失败: %w", err)
	}
	defer browserCtx.Release()

	page, err := browserCtx.GetPage()
	if err != nil {
		return fmt.Errorf("失败: 登录 - 获取页面失败: %w", err)
	}

	utils.InfoWithPlatform(u.platform, "正在打开发布页面...")
	if _, err := page.Goto("https://member.bilibili.com/platform/upload/video/frame", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		return fmt.Errorf("失败: 登录 - 打开页面失败: %w", err)
	}

	utils.InfoWithPlatform(u.platform, "请使用APP扫码登录")

	cookieConfig, ok := browser.GetCookieConfig("bilibili")
	if !ok {
		return fmt.Errorf("失败: 登录 - 获取Cookie配置失败")
	}

	if err := browserCtx.WaitForLoginCookies(cookieConfig); err != nil {
		return fmt.Errorf("失败: 登录 - 等待登录Cookie失败: %w", err)
	}

	maxWaitAttempts := 30
	loginSuccess := false
	for i := 0; i < maxWaitAttempts; i++ {
		url := page.URL()
		if strings.Contains(url, "member.bilibili.com/platform/home") ||
			strings.Contains(url, "member.bilibili.com/platform/upload") {
			loginSuccess = true
			break
		}
		time.Sleep(1 * time.Second)
		if i == maxWaitAttempts-1 {
			return fmt.Errorf("失败: 登录 - 等待跳转超时")
		}
	}

	if !loginSuccess {
		return fmt.Errorf("失败: 登录 - 登录验证失败")
	}

	utils.SuccessWithPlatform(u.platform, "登录成功")
	return u.saveCookiesFromPage(page)
}

// saveCookiesFromPage 从页面保存Cookie
func (u *Uploader) saveCookiesFromPage(page playwright.Page) error {
	debugLog("saveCookiesFromPage - cookiePath: '%s'", u.cookiePath)
	if u.cookiePath == "" {
		return fmt.Errorf("失败: 保存Cookie - cookie路径为空")
	}

	storageState, err := page.Context().StorageState()
	if err != nil {
		return fmt.Errorf("失败: 保存Cookie - 获取存储状态失败: %w", err)
	}

	data, err := json.Marshal(storageState)
	if err != nil {
		return fmt.Errorf("失败: 保存Cookie - 序列化失败: %w", err)
	}

	if err := os.WriteFile(u.cookiePath, data, 0644); err != nil {
		return fmt.Errorf("失败: 保存Cookie - 写入失败: %w", err)
	}

	return nil
}
