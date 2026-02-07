package tiktok

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"Fuploader/internal/platform/browser"
	"Fuploader/internal/platform/uploader"
	"Fuploader/internal/types"
	"Fuploader/internal/utils"

	"github.com/playwright-community/playwright-go"
)

// browserPool 全局浏览器池实例
var browserPool *browser.Pool

// initBrowserPool 初始化浏览器池
func initBrowserPool() {
	if browserPool == nil {
		browserPool = browser.NewPool(2, 5) // 最多2个浏览器，每个5个上下文
	}
}

// Uploader TikTok上传器
type Uploader struct {
	*uploader.Base
	locatorBase playwright.Locator // 动态定位器基类
}

// NewUploader 创建上传器
func NewUploader(cookiePath string) *Uploader {
	initBrowserPool()
	return &Uploader{
		Base: uploader.NewBase("tiktok", cookiePath, browserPool),
	}
}

// Platform 返回平台名称
func (u *Uploader) Platform() string {
	return "tiktok"
}

// ValidateCookie 验证 Cookie 是否有效
// 参照Python版本：访问tiktokstudio/upload，检测select元素的class
func (u *Uploader) ValidateCookie(ctx context.Context) (bool, error) {
	// 从浏览器池获取上下文
	browserCtx, err := browserPool.GetContext(ctx, u.GetCookiePath(), u.GetContextOptions())
	if err != nil {
		return false, fmt.Errorf("get browser context failed: %w", err)
	}
	defer browserCtx.Release()

	page, err := browserCtx.GetPage()
	if err != nil {
		return false, fmt.Errorf("get page failed: %w", err)
	}

	// 访问 TikTok Studio（参照Python版本）
	utils.Info("[-] 正在验证 TikTok 登录状态...")
	if _, err := page.Goto("https://www.tiktok.com/tiktokstudio/upload?lang=en", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	}); err != nil {
		return false, fmt.Errorf("goto upload page failed: %w", err)
	}

	// 等待页面加载
	time.Sleep(3 * time.Second)

	// 参照Python版本：检测select元素的class
	selectElements, err := page.QuerySelectorAll("select")
	if err != nil {
		utils.Warn(fmt.Sprintf("[-] 查询select元素失败: %v", err))
		// 如果查询失败，尝试其他检测方式
		return u.validateByAlternative(page)
	}

	for _, element := range selectElements {
		className, err := element.GetAttribute("class")
		if err != nil {
			continue
		}

		// 参照Python版本：使用正则表达式匹配特定模式的class名称
		// re.match(r'tiktok-.*-SelectFormContainer.*', class_name)
		matched, _ := regexp.MatchString(`tiktok-.*-SelectFormContainer.*`, className)
		if matched {
			utils.Info("[-] 检测到未登录特征元素（SelectFormContainer），Cookie 已失效")
			return false, nil
		}
	}

	// 如果没有检测到未登录特征，说明Cookie有效
	utils.Info("[-] Cookie 有效")
	return true, nil
}

// validateByAlternative 备用验证方式
func (u *Uploader) validateByAlternative(page playwright.Page) (bool, error) {
	// 检查当前URL
	currentURL := page.URL()
	utils.Info(fmt.Sprintf("[-] 当前URL: %s", currentURL))

	// 如果被重定向到登录页，说明Cookie无效
	if strings.Contains(currentURL, "/login") {
		utils.Info("[-] 被重定向到登录页，Cookie 无效")
		return false, nil
	}

	// 检查是否有创作者中心特征元素
	uploadBtn, _ := page.GetByText("Upload").Count()
	dashboardMenu, _ := page.GetByText("Dashboard").Count()
	contentMenu, _ := page.GetByText("Content").Count()

	if uploadBtn > 0 || dashboardMenu > 0 || contentMenu > 0 {
		utils.Info("[-] 检测到创作者中心特征元素，Cookie 有效")
		return true, nil
	}

	utils.Info("[-] 未检测到登录特征，Cookie 无效")
	return false, nil
}

// Upload 上传视频
func (u *Uploader) Upload(ctx context.Context, task *types.VideoTask) error {
	steps := []uploader.StepFunc{
		// 1. 导航到上传页面
		u.StepNavigate("https://www.tiktok.com/tiktokstudio/upload"),

		// 2. 初始化定位器基类（检测iframe）
		u.StepInitLocatorBase(),

		// 3. 上传视频
		u.StepUploadTikTokVideo(task.VideoPath),

		// 4. 填写标题和标签（使用DraftEditor）
		u.StepFillTikTokTitleAndTags(task.Title, task.Tags),

		// 5. 设置定时发布（如果有）
		u.StepSetScheduleTikTok(task.ScheduleTime),

		// 6. 点击发布
		u.StepClickPublishTikTok(task.ScheduleTime != nil),
	}

	return u.Execute(ctx, task, steps)
}

// StepInitLocatorBase 初始化定位器基类（检测iframe深度适配）
func (u *Uploader) StepInitLocatorBase() uploader.StepFunc {
	return func(ctx *uploader.Context) uploader.StepResult {
		ctx.ReportProgress(uploader.StepNavigate, 18, "检测页面结构...")

		// 等待iframe或普通容器出现
		timeout := time.After(10 * time.Second)
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-timeout:
				return uploader.StepResult{Step: uploader.StepNavigate, Success: false, Error: fmt.Errorf("timeout waiting for upload container")}
			case <-ticker.C:
				// 检测iframe
				iframeCount, _ := ctx.Page.Locator("iframe[data-tt='Upload_index_iframe']").Count()
				if iframeCount > 0 {
					// 使用iframe内的定位器
					frame := ctx.Page.FrameLocator("iframe[data-tt='Upload_index_iframe']")
					u.locatorBase = frame.Locator("div.upload-container")
					ctx.ReportProgress(uploader.StepNavigate, 19, "检测到iframe结构")
					return uploader.StepResult{Step: uploader.StepNavigate, Success: true}
				}

				// 检测普通容器
				containerCount, _ := ctx.Page.Locator("div.upload-container").Count()
				if containerCount > 0 {
					u.locatorBase = ctx.Page.Locator("div.upload-container")
					ctx.ReportProgress(uploader.StepNavigate, 19, "使用普通容器结构")
					return uploader.StepResult{Step: uploader.StepNavigate, Success: true}
				}
			}
		}
	}
}

// getLocatorBase 获取当前定位器基类
func (u *Uploader) getLocatorBase(ctx *uploader.Context) playwright.Locator {
	if u.locatorBase != nil {
		return u.locatorBase
	}
	// 降级处理：返回页面级别的定位器
	return ctx.Page.Locator("body")
}

// StepUploadTikTokVideo TikTok上传视频步骤（支持iframe和错误重试）
func (u *Uploader) StepUploadTikTokVideo(videoPath string) uploader.StepFunc {
	return func(ctx *uploader.Context) uploader.StepResult {
		ctx.ReportProgress(uploader.StepUploadMedia, 20, "开始上传视频...")

		// 使用文件选择器模式上传
		uploadButton := u.getLocatorBase(ctx).Locator("button:has-text('Select video'):visible")
		if err := uploadButton.WaitFor(playwright.LocatorWaitForOptions{State: playwright.WaitForSelectorStateVisible}); err != nil {
			return uploader.StepResult{Step: uploader.StepUploadMedia, Success: false, Error: fmt.Errorf("upload button not visible: %w", err)}
		}

		// 使用文件选择器
		fileChooser, err := ctx.Page.ExpectFileChooser(func() error {
			return uploadButton.Click()
		})
		if err != nil {
			return uploader.StepResult{Step: uploader.StepUploadMedia, Success: false, Error: fmt.Errorf("expect file chooser failed: %w", err)}
		}

		if err := fileChooser.SetFiles(videoPath); err != nil {
			return uploader.StepResult{Step: uploader.StepUploadMedia, Success: false, Error: fmt.Errorf("set files failed: %w", err)}
		}

		ctx.ReportProgress(uploader.StepUploadMedia, 25, "视频文件已选择，等待上传...")

		// 等待上传完成（支持错误重试）
		return u.waitForUploadComplete(ctx)
	}
}

// waitForUploadComplete 等待上传完成，支持错误重试
func (u *Uploader) waitForUploadComplete(ctx *uploader.Context) uploader.StepResult {
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return uploader.StepResult{Step: uploader.StepUploadMedia, Success: false, Error: fmt.Errorf("upload timeout after 5 minutes")}
		case <-ticker.C:
			// 检查发布按钮是否可用（上传完成的标志）
			postButton := u.getLocatorBase(ctx).Locator("div.btn-post > button")
			disabledAttr, _ := postButton.GetAttribute("disabled")
			if disabledAttr == "" || disabledAttr == "false" {
				ctx.ReportProgress(uploader.StepUploadMedia, 40, "视频上传完成")
				return uploader.StepResult{Step: uploader.StepUploadMedia, Success: true}
			}

			// 检测上传错误并重试
			selectFileBtn := u.getLocatorBase(ctx).Locator("button[aria-label='Select file']")
			if count, _ := selectFileBtn.Count(); count > 0 {
				ctx.ReportProgress(uploader.StepUploadMedia, 30, "检测到上传错误，正在重试...")
				if err := u.handleUploadError(ctx); err != nil {
					return uploader.StepResult{Step: uploader.StepUploadMedia, Success: false, Error: fmt.Errorf("upload retry failed: %w", err)}
				}
			}

			utils.Info("[-] 正在上传视频中...")
		}
	}
}

// handleUploadError 处理上传错误并重试
func (u *Uploader) handleUploadError(ctx *uploader.Context) error {
	utils.Info("[-] 处理上传错误，重新选择文件...")

	selectFileBtn := u.getLocatorBase(ctx).Locator("button[aria-label='Select file']")
	videoPath := ctx.Task.VideoPath

	// 使用文件选择器重试
	fileChooser, err := ctx.Page.ExpectFileChooser(func() error {
		return selectFileBtn.Click()
	})
	if err != nil {
		return fmt.Errorf("expect file chooser failed: %w", err)
	}

	if err := fileChooser.SetFiles(videoPath); err != nil {
		return fmt.Errorf("set files failed: %w", err)
	}

	return nil
}

// StepFillTikTokTitleAndTags TikTok填写标题和标签步骤（使用DraftEditor）
func (u *Uploader) StepFillTikTokTitleAndTags(title string, tags []string) uploader.StepFunc {
	return func(ctx *uploader.Context) uploader.StepResult {
		ctx.ReportProgress(uploader.StepFillTitle, 45, "正在填写标题...")

		// 使用DraftEditor-content定位器
		editorLocator := u.getLocatorBase(ctx).Locator("div.public-DraftEditor-content")
		if err := editorLocator.Click(); err != nil {
			return uploader.StepResult{Step: uploader.StepFillTitle, Success: false, Error: fmt.Errorf("click editor failed: %w", err)}
		}

		time.Sleep(500 * time.Millisecond)

		// 清空原有内容
		if err := ctx.Page.Keyboard().Press("End"); err != nil {
			return uploader.StepResult{Step: uploader.StepFillTitle, Success: false, Error: err}
		}
		if err := ctx.Page.Keyboard().Press("Control+KeyA"); err != nil {
			return uploader.StepResult{Step: uploader.StepFillTitle, Success: false, Error: err}
		}
		if err := ctx.Page.Keyboard().Press("Delete"); err != nil {
			return uploader.StepResult{Step: uploader.StepFillTitle, Success: false, Error: err}
		}
		if err := ctx.Page.Keyboard().Press("End"); err != nil {
			return uploader.StepResult{Step: uploader.StepFillTitle, Success: false, Error: err}
		}

		time.Sleep(500 * time.Millisecond)

		// 输入标题
		if err := ctx.Page.Keyboard().Type(title); err != nil {
			return uploader.StepResult{Step: uploader.StepFillTitle, Success: false, Error: fmt.Errorf("type title failed: %w", err)}
		}

		time.Sleep(500 * time.Millisecond)
		if err := ctx.Page.Keyboard().Press("End"); err != nil {
			return uploader.StepResult{Step: uploader.StepFillTitle, Success: false, Error: err}
		}
		if err := ctx.Page.Keyboard().Press("Enter"); err != nil {
			return uploader.StepResult{Step: uploader.StepFillTitle, Success: false, Error: err}
		}

		ctx.ReportProgress(uploader.StepFillTitle, 50, "标题填写完成")

		// 添加标签
		ctx.ReportProgress(uploader.StepAddTags, 52, "正在添加标签...")
		for index, tag := range tags {
			if err := ctx.Page.Keyboard().Press("End"); err != nil {
				return uploader.StepResult{Step: uploader.StepAddTags, Success: false, Error: err}
			}
			time.Sleep(500 * time.Millisecond)

			if err := ctx.Page.Keyboard().Type("#" + tag + " "); err != nil {
				return uploader.StepResult{Step: uploader.StepAddTags, Success: false, Error: fmt.Errorf("type tag failed: %w", err)}
			}
			if err := ctx.Page.Keyboard().Press("Space"); err != nil {
				return uploader.StepResult{Step: uploader.StepAddTags, Success: false, Error: err}
			}
			time.Sleep(500 * time.Millisecond)
			if err := ctx.Page.Keyboard().Press("Backspace"); err != nil {
				return uploader.StepResult{Step: uploader.StepAddTags, Success: false, Error: err}
			}
			if err := ctx.Page.Keyboard().Press("End"); err != nil {
				return uploader.StepResult{Step: uploader.StepAddTags, Success: false, Error: err}
			}

			utils.Info(fmt.Sprintf("[-] 已设置第 %d 个标签", index+1))
		}

		ctx.ReportProgress(uploader.StepAddTags, 55, fmt.Sprintf("已添加 %d 个标签", len(tags)))
		return uploader.StepResult{Step: uploader.StepFillTitle, Success: true}
	}
}

// StepSetScheduleTikTok TikTok设置定时发布步骤（完整日历选择器）
func (u *Uploader) StepSetScheduleTikTok(scheduleTime *string) uploader.StepFunc {
	return func(ctx *uploader.Context) uploader.StepResult {
		if scheduleTime == nil || *scheduleTime == "" {
			return uploader.StepResult{Step: uploader.StepSetSchedule, Success: true}
		}

		// 解析时间
		publishDate, err := time.Parse("2006-01-02 15:04", *scheduleTime)
		if err != nil {
			return uploader.StepResult{Step: uploader.StepSetSchedule, Success: false, Error: fmt.Errorf("parse schedule time failed: %w", err)}
		}

		ctx.ReportProgress(uploader.StepSetSchedule, 70, "正在设置定时发布...")

		// 点击Schedule按钮
		scheduleBtn := u.getLocatorBase(ctx).GetByLabel("Schedule")
		if err := scheduleBtn.WaitFor(playwright.LocatorWaitForOptions{State: playwright.WaitForSelectorStateVisible}); err != nil {
			return uploader.StepResult{Step: uploader.StepSetSchedule, Success: false, Error: fmt.Errorf("schedule button not visible: %w", err)}
		}
		if err := scheduleBtn.Click(); err != nil {
			return uploader.StepResult{Step: uploader.StepSetSchedule, Success: false, Error: fmt.Errorf("click schedule button failed: %w", err)}
		}

		time.Sleep(1 * time.Second)

		// 打开日历选择器
		scheduledPicker := u.getLocatorBase(ctx).Locator("div.scheduled-picker")
		calendarBtn := scheduledPicker.Locator("div.TUXInputBox").Nth(1)
		if err := calendarBtn.Click(); err != nil {
			return uploader.StepResult{Step: uploader.StepSetSchedule, Success: false, Error: fmt.Errorf("click calendar button failed: %w", err)}
		}

		time.Sleep(500 * time.Millisecond)

		// 获取当前月份并切换
		monthTitle := u.getLocatorBase(ctx).Locator("div.calendar-wrapper span.month-title")
		monthText, err := monthTitle.TextContent()
		if err != nil {
			return uploader.StepResult{Step: uploader.StepSetSchedule, Success: false, Error: fmt.Errorf("get month title failed: %w", err)}
		}

		currentMonth := parseMonth(monthText)
		targetMonth := int(publishDate.Month())

		// 月份切换
		if currentMonth != targetMonth {
			ctx.ReportProgress(uploader.StepSetSchedule, 72, fmt.Sprintf("切换月份: %d -> %d", currentMonth, targetMonth))

			arrowIndex := 0
			if currentMonth < targetMonth {
				// 点击下一个箭头
				arrows := u.getLocatorBase(ctx).Locator("div.calendar-wrapper span.arrow")
				count, _ := arrows.Count()
				arrowIndex = int(count) - 1
			}

			arrow := u.getLocatorBase(ctx).Locator("div.calendar-wrapper span.arrow").Nth(arrowIndex)
			if err := arrow.Click(); err != nil {
				return uploader.StepResult{Step: uploader.StepSetSchedule, Success: false, Error: fmt.Errorf("click month arrow failed: %w", err)}
			}
			time.Sleep(500 * time.Millisecond)
		}

		// 选择日期
		validDays := u.getLocatorBase(ctx).Locator("div.calendar-wrapper span.day.valid")
		count, err := validDays.Count()
		if err != nil {
			return uploader.StepResult{Step: uploader.StepSetSchedule, Success: false, Error: fmt.Errorf("get valid days failed: %w", err)}
		}

		targetDay := strconv.Itoa(publishDate.Day())
		for i := 0; i < count; i++ {
			dayText, err := validDays.Nth(i).TextContent()
			if err != nil {
				continue
			}
			if strings.TrimSpace(dayText) == targetDay {
				if err := validDays.Nth(i).Click(); err != nil {
					return uploader.StepResult{Step: uploader.StepSetSchedule, Success: false, Error: fmt.Errorf("click day failed: %w", err)}
				}
				break
			}
		}

		ctx.ReportProgress(uploader.StepSetSchedule, 75, "日期选择完成")

		// 选择时间
		timeBtn := scheduledPicker.Locator("div.TUXInputBox").Nth(0)
		if err := timeBtn.Click(); err != nil {
			return uploader.StepResult{Step: uploader.StepSetSchedule, Success: false, Error: fmt.Errorf("click time button failed: %w", err)}
		}
		time.Sleep(500 * time.Millisecond)

		// 选择小时
		hourStr := publishDate.Format("15")
		hourSelector := fmt.Sprintf("span.tiktok-timepicker-left:has-text('%s')", hourStr)
		hourElement := u.getLocatorBase(ctx).Locator(hourSelector)
		if err := hourElement.Click(); err != nil {
			return uploader.StepResult{Step: uploader.StepSetSchedule, Success: false, Error: fmt.Errorf("select hour failed: %w", err)}
		}

		time.Sleep(500 * time.Millisecond)

		// 重新打开时间选择器选择分钟
		if err := timeBtn.Click(); err != nil {
			return uploader.StepResult{Step: uploader.StepSetSchedule, Success: false, Error: fmt.Errorf("click time button again failed: %w", err)}
		}
		time.Sleep(500 * time.Millisecond)

		// 选择分钟（TikTok使用5分钟间隔）
		correctMinute := int(publishDate.Minute()/5) * 5
		minuteStr := fmt.Sprintf("%02d", correctMinute)
		minuteSelector := fmt.Sprintf("span.tiktok-timepicker-right:has-text('%s')", minuteStr)
		minuteElement := u.getLocatorBase(ctx).Locator(minuteSelector)
		if err := minuteElement.Click(); err != nil {
			return uploader.StepResult{Step: uploader.StepSetSchedule, Success: false, Error: fmt.Errorf("select minute failed: %w", err)}
		}

		// 点击标题移除焦点
		uploadTitle := u.getLocatorBase(ctx).Locator("h1:has-text('Upload video')")
		uploadTitle.Click()

		ctx.ReportProgress(uploader.StepSetSchedule, 80, "定时发布设置完成")
		return uploader.StepResult{Step: uploader.StepSetSchedule, Success: true}
	}
}

// parseMonth 解析英文月份名称到数字
func parseMonth(monthName string) int {
	months := map[string]int{
		"January":   1,
		"February":  2,
		"March":     3,
		"April":     4,
		"May":       5,
		"June":      6,
		"July":      7,
		"August":    8,
		"September": 9,
		"October":   10,
		"November":  11,
		"December":  12,
	}

	// 尝试直接匹配
	if month, ok := months[monthName]; ok {
		return month
	}

	// 尝试部分匹配（处理可能的额外字符）
	for name, month := range months {
		if strings.Contains(monthName, name) {
			return month
		}
	}

	return 0
}

// StepClickPublishTikTok TikTok点击发布步骤
func (u *Uploader) StepClickPublishTikTok(isScheduled bool) uploader.StepFunc {
	return func(ctx *uploader.Context) uploader.StepResult {
		ctx.ReportProgress(uploader.StepPublish, 85, "正在发布...")

		// 成功标志选择器
		successFlagDiv := "#\\:r9\\:"

		for {
			// 点击发布按钮
			publishBtn := u.getLocatorBase(ctx).Locator("div.btn-post")
			if count, _ := publishBtn.Count(); count > 0 {
				if err := publishBtn.Click(); err != nil {
					return uploader.StepResult{Step: uploader.StepPublish, Success: false, Error: fmt.Errorf("click publish button failed: %w", err)}
				}
			}

			// 等待成功标志
			time.Sleep(3 * time.Second)

			// 检查成功标志
			successLocator := u.getLocatorBase(ctx).Locator(successFlagDiv)
			if visible, _ := successLocator.IsVisible(); visible {
				// 保存cookie（参照Python版本）
				if ctx.BrowserCtx != nil {
					if err := ctx.BrowserCtx.SaveCookies(); err != nil {
						utils.Warn(fmt.Sprintf("[-] 保存cookie失败: %v", err))
					} else {
						utils.Info("[-] Cookie已保存")
					}
				}
				ctx.ReportProgress(uploader.StepPublish, 95, "发布成功")
				return uploader.StepResult{Step: uploader.StepPublish, Success: true}
			}

			// 检查是否成功（通过计数）
			if count, _ := successLocator.Count(); count > 0 {
				// 保存cookie
				if ctx.BrowserCtx != nil {
					if err := ctx.BrowserCtx.SaveCookies(); err != nil {
						utils.Warn(fmt.Sprintf("[-] 保存cookie失败: %v", err))
					} else {
						utils.Info("[-] Cookie已保存")
					}
				}
				ctx.ReportProgress(uploader.StepPublish, 95, "发布成功")
				return uploader.StepResult{Step: uploader.StepPublish, Success: true}
			}

			// 检查是否跳转到内容管理页面
			url := ctx.Page.URL()
			if url == "https://www.tiktok.com/tiktokstudio/content" {
				// 保存cookie
				if ctx.BrowserCtx != nil {
					if err := ctx.BrowserCtx.SaveCookies(); err != nil {
						utils.Warn(fmt.Sprintf("[-] 保存cookie失败: %v", err))
					} else {
						utils.Info("[-] Cookie已保存")
					}
				}
				ctx.ReportProgress(uploader.StepPublish, 95, "发布成功（跳转到内容页面）")
				return uploader.StepResult{Step: uploader.StepPublish, Success: true}
			}

			utils.Info("[-] 等待发布完成...")
			time.Sleep(500 * time.Millisecond)
		}
	}
}

// Login 登录（参照Python版本使用page.pause）
func (u *Uploader) Login() error {
	ctx := context.Background()

	// 创建新的浏览器上下文（不使用现有cookie）
	browserCtx, err := browserPool.GetContext(ctx, "", u.GetContextOptions())
	if err != nil {
		return fmt.Errorf("get browser context failed: %w", err)
	}
	// 注意：不在此处defer Release，在登录成功或失败时手动释放

	page, err := browserCtx.GetPage()
	if err != nil {
		browserCtx.Release()
		return fmt.Errorf("get page failed: %w", err)
	}

	// 参照Python版本：访问登录页面
	utils.Info("[-] 正在打开 TikTok 登录页面...")
	if _, err := page.Goto("https://www.tiktok.com/login?lang=en", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	}); err != nil {
		browserCtx.Release()
		return fmt.Errorf("goto login page failed: %w", err)
	}

	utils.Info("[-] 请在浏览器窗口中完成登录")
	utils.Info("[-] 提示：TikTok 支持多种登录方式（Gmail/手机号/社交账号等）")
	utils.Info("[-] 登录完成后，请在开发者工具中点击继续（Resume）按钮")

	// 参照Python版本：使用page.pause()暂停，等待用户完成登录
	if err := page.Pause(); err != nil {
		utils.Warn(fmt.Sprintf("[-] page.pause()返回: %v", err))
	}

	// 用户点击继续后，检查是否登录成功
	utils.Info("[-] 正在检查登录状态...")
	time.Sleep(2 * time.Second)

	// 检查当前URL
	currentURL := page.URL()
	utils.Info(fmt.Sprintf("[-] 当前URL: %s", currentURL))

	// 检查是否已进入创作者中心或首页
	if u.isLoginSuccessURL(currentURL) {
		utils.Info("[-] 登录成功")
		// 保存cookie
		if err := browserCtx.SaveCookiesTo(u.GetCookiePath()); err != nil {
			utils.Warn(fmt.Sprintf("[-] 保存Cookie失败: %v", err))
			browserCtx.Release()
			return fmt.Errorf("save cookies failed: %w", err)
		}
		utils.Info("[-] Cookie已保存")
		browserCtx.Release()
		return nil
	}

	// 如果没有进入创作者中心，尝试保存cookie anyway
	utils.Warn("[-] 未检测到创作者中心页面，但仍尝试保存Cookie")
	if err := browserCtx.SaveCookiesTo(u.GetCookiePath()); err != nil {
		utils.Warn(fmt.Sprintf("[-] 保存Cookie失败: %v", err))
	}

	browserCtx.Release()
	return fmt.Errorf("login may not be complete, current url: %s", currentURL)
}

// isLoginSuccessURL 检查URL是否表示登录成功
func (u *Uploader) isLoginSuccessURL(url string) bool {
	successURLs := []string{
		"https://www.tiktok.com/tiktokstudio",
		"https://www.tiktok.com/tiktokstudio/",
		"https://www.tiktok.com/tiktokstudio/content",
		"https://www.tiktok.com/tiktokstudio/upload",
		"https://www.tiktok.com/foryou",
		"https://www.tiktok.com/foryou/",
		"https://www.tiktok.com/",
		"https://www.tiktok.com",
	}

	for _, successURL := range successURLs {
		if strings.HasPrefix(url, successURL) {
			return true
		}
	}
	return false
}

// GetContextOptions 获取TikTok特定的上下文选项
func (u *Uploader) GetContextOptions() *browser.ContextOptions {
	return &browser.ContextOptions{
		UserAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:120.0) Gecko/20100101 Firefox/120.0",
		Viewport:    &playwright.Size{Width: 1920, Height: 1080},
		Locale:      "en-GB",
		TimezoneId:  "Europe/London",
		Geolocation: &playwright.Geolocation{Latitude: 51.5074, Longitude: -0.1278},
		ExtraHeaders: map[string]string{
			"Accept-Language": "en-GB,en;q=0.9",
		},
	}
}
