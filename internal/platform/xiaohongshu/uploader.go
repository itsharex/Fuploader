package xiaohongshu

import (
	"context"
	"fmt"
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

// Uploader 小红书上器
type Uploader struct {
	*uploader.Base
	cookiePath string
}

// NewUploader 创建上传器
func NewUploader(cookiePath string) *Uploader {
	initBrowserPool()
	// 使用 NewBase，截图配置由前端控制
	base := uploader.NewBase("xiaohongshu", cookiePath, browserPool)
	return &Uploader{
		Base: base,
	}
}

// NewUploaderWithScreenshot 创建带截图配置的上传器
func NewUploaderWithScreenshot(cookiePath string, enableScreenshot bool, screenshotDir string) *Uploader {
	initBrowserPool()
	base := uploader.NewBaseWithScreenshot("xiaohongshu", cookiePath, browserPool, enableScreenshot, screenshotDir)
	return &Uploader{
		Base: base,
	}
}

// Platform 返回平台名称
func (u *Uploader) Platform() string {
	return "xiaohongshu"
}

// GetBrowserPool 获取浏览器池（供增强功能使用）
func (u *Uploader) GetBrowserPool() *browser.Pool {
	return browserPool
}

// GetCookiePath 获取 cookie 路径（供增强功能使用）
func (u *Uploader) GetCookiePath() string {
	return u.cookiePath
}

// ValidateCookie 验证 Cookie 是否有效
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

	// 访问创作者中心
	utils.Info("[-] 正在验证登录状态...")
	if _, err := page.Goto("https://creator.xiaohongshu.com/creator-micro/content/upload"); err != nil {
		return false, fmt.Errorf("goto creator center failed: %w", err)
	}

	// 等待页面加载
	time.Sleep(3 * time.Second)

	// 检查1: 是否成功进入上传页面
	url := page.URL()
	if url == "https://creator.xiaohongshu.com/creator-micro/content/upload" {
		utils.Info("[-] 已进入上传页面，Cookie 有效")
		return true, nil
	}

	// 检查2: 是否有登录按钮（未登录）
	loginBtnCount, _ := page.GetByText("手机号登录").Count()
	scanBtnCount, _ := page.GetByText("扫码登录").Count()
	if loginBtnCount > 0 || scanBtnCount > 0 {
		utils.Info("[-] 检测到登录按钮，Cookie 无效")
		return false, nil
	}

	// 检查3: 是否被重定向到登录页
	if strings.Contains(url, "/login") {
		utils.Info("[-] 被重定向到登录页，Cookie 无效")
		return false, nil
	}

	return false, fmt.Errorf("unknown login status, current url: %s", url)
}

// Upload 上传视频（增强版）
func (u *Uploader) Upload(ctx context.Context, task *types.VideoTask) error {
	steps := []uploader.StepFunc{
		// 1. 先访问小红书主页（模拟自然浏览路径，参考登录逻辑）
		u.StepNavigateToHomepage(),

		// 2. 从首页进入创作者中心发布页面
		u.StepNavigateToCreatorUpload(),

		// 3. 上传视频（增强版，带验证码和反爬虫检测）
		u.StepUploadVideoXiaohongshuEnhanced(),

		// 4. 填写标题（增强版，带人类行为模拟）
		u.StepFillTitleEnhanced(
			"div.plugin.title-container input.d-text", // 新页面
			".notranslate", // 旧页面
			30,
		),

		// 5. 添加标签（增强版）
		u.StepAddTagsEnhanced(".ql-editor"),

		// 6. 设置封面（如果有）
		u.StepSetThumbnailEnhanced(),

		// 7. 设置位置（如果有）
		u.StepSetLocationEnhanced(),

		// 8. 设置定时发布（如果有）
		u.StepSetScheduleXiaohongshuEnhanced(),

		// 9. 点击发布（增强版）
		u.StepClickPublishXiaohongshuEnhanced(),
	}

	return u.Execute(ctx, task, steps)
}

// StepUploadVideoXiaohongshu 小红书专用上传视频步骤（基础版本）
// 注意：此方法被 StepUploadVideoXiaohongshuEnhanced() 包装调用，保留用于回退和测试
func (u *Uploader) StepUploadVideoXiaohongshu() uploader.StepFunc {
	return func(ctx *uploader.Context) uploader.StepResult {
		ctx.ReportProgress(uploader.StepUploadMedia, 20, "开始上传视频...")

		// 设置输入文件
		input := ctx.Page.Locator("div[class^='upload-content'] input[class='upload-input']")
		if err := input.SetInputFiles(ctx.Task.VideoPath); err != nil {
			// 尝试通用选择器
			input = ctx.Page.Locator("input[type='file']")
			if err := input.SetInputFiles(ctx.Task.VideoPath); err != nil {
				return uploader.StepResult{Step: uploader.StepUploadMedia, Success: false, Error: err}
			}
		}

		ctx.ReportProgress(uploader.StepUploadMedia, 25, "视频文件已选择，等待上传...")

		// 等待上传完成 - 参考Python的DOM+XPath检测逻辑
		timeout := time.After(5 * time.Minute)
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		checkCount := 0
		for {
			select {
			case <-timeout:
				ctx.TakeScreenshot("upload_timeout")
				return uploader.StepResult{Step: uploader.StepUploadMedia, Success: false, Error: fmt.Errorf("upload timeout after 5 minutes")}
			case <-ticker.C:
				checkCount++

				// 检测逻辑1: 检查上传成功标识（参考Python实现）
				success, err := u.detectUploadSuccess(ctx.Page)
				if err == nil && success {
					ctx.ReportProgress(uploader.StepUploadMedia, 40, "视频上传成功")
					ctx.TakeScreenshot("upload_success")
					return uploader.StepResult{Step: uploader.StepUploadMedia, Success: true}
				}

				// 检测逻辑2: 检查重新上传按钮（备选）
				reuploadCount, _ := ctx.Page.Locator("[class^=\"long-card\"] div:has-text(\"重新上传\")").Count()
				if reuploadCount > 0 {
					ctx.ReportProgress(uploader.StepUploadMedia, 40, "视频上传完成（检测到重新上传按钮）")
					ctx.TakeScreenshot("upload_success")
					return uploader.StepResult{Step: uploader.StepUploadMedia, Success: true}
				}

				// 检测逻辑3: 检查上传失败
				errorCount, _ := ctx.Page.Locator("div.progress-div > div:has-text(\"上传失败\")").Count()
				if errorCount > 0 {
					ctx.TakeScreenshot("upload_error")
					utils.Warn("[-] 检测到上传失败，尝试重试...")
					// 尝试重新上传
					if err := u.handleUploadError(ctx); err != nil {
						return uploader.StepResult{Step: uploader.StepUploadMedia, Success: false, Error: fmt.Errorf("upload failed and retry failed: %w", err)}
					}
					// 重试后继续检测
					continue
				}

				// 每10秒报告一次进度
				if checkCount%5 == 0 {
					progress := 25 + (checkCount/5)*3
					if progress > 38 {
						progress = 38
					}
					ctx.ReportProgress(uploader.StepUploadMedia, progress, "正在上传视频中...")
					utils.Info("[-] 正在上传视频中...")
				}
			}
		}
	}
}

// detectUploadSuccess 检测上传是否成功（参考Python的DOM+XPath检测）
func (u *Uploader) detectUploadSuccess(page playwright.Page) (bool, error) {
	// 获取upload-input元素
	uploadInput := page.Locator("input.upload-input")
	count, err := uploadInput.Count()
	if err != nil || count == 0 {
		return false, fmt.Errorf("upload input not found")
	}

	// 获取下一个兄弟元素（preview-new）
	previewNew := uploadInput.Locator("xpath=following-sibling::div[contains(@class, 'preview-new')]")
	count, err = previewNew.Count()
	if err != nil || count == 0 {
		return false, fmt.Errorf("preview element not found")
	}

	// 在preview-new元素中查找包含"上传成功"的stage元素
	stageElements := previewNew.Locator("div.stage")
	count, err = stageElements.Count()
	if err != nil || count == 0 {
		return false, fmt.Errorf("stage elements not found")
	}

	// 遍历检查每个stage元素的文本内容
	for i := 0; i < count; i++ {
		text, err := stageElements.Nth(i).TextContent()
		if err != nil {
			continue
		}
		if strings.Contains(text, "上传成功") {
			return true, nil
		}
	}

	return false, fmt.Errorf("upload success text not found")
}

// handleUploadError 处理上传错误（参考Python实现）
func (u *Uploader) handleUploadError(ctx *uploader.Context) error {
	utils.Info("[-] 视频出错了，重新上传中...")

	// 重新设置输入文件
	input := ctx.Page.Locator("div.progress-div [class^=\"upload-btn-input\"]")
	if err := input.SetInputFiles(ctx.Task.VideoPath); err != nil {
		// 尝试通用选择器
		input = ctx.Page.Locator("input[type='file']")
		if err := input.SetInputFiles(ctx.Task.VideoPath); err != nil {
			return fmt.Errorf("retry upload failed: %w", err)
		}
	}

	return nil
}

// StepSetThumbnail 设置封面（基础版本）
// 注意：此方法被 StepSetThumbnailEnhanced() 替代，保留用于回退和测试
func (u *Uploader) StepSetThumbnail() uploader.StepFunc {
	return func(ctx *uploader.Context) uploader.StepResult {
		if ctx.Task.Thumbnail == "" {
			return uploader.StepResult{Step: uploader.StepSetCover, Success: true}
		}

		ctx.ReportProgress(uploader.StepSetCover, 58, "正在设置封面...")

		// 点击"选择封面"
		if err := ctx.Page.GetByText("选择封面").Click(); err != nil {
			utils.Warn(fmt.Sprintf("[-] 点击选择封面失败: %v", err))
			return uploader.StepResult{Step: uploader.StepSetCover, Success: true} // 非关键步骤，失败继续
		}
		time.Sleep(2 * time.Second)

		// 点击"设置竖封面"
		if err := ctx.Page.GetByText("设置竖封面").Click(); err != nil {
			utils.Warn(fmt.Sprintf("[-] 点击设置竖封面失败: %v", err))
			return uploader.StepResult{Step: uploader.StepSetCover, Success: true}
		}
		time.Sleep(2 * time.Second)

		// 上传封面文件
		coverInput := ctx.Page.Locator("div[class^='semi-upload upload'] >> input.semi-upload-hidden-input")
		if err := coverInput.SetInputFiles(ctx.Task.Thumbnail); err != nil {
			utils.Warn(fmt.Sprintf("[-] 上传封面失败: %v", err))
			return uploader.StepResult{Step: uploader.StepSetCover, Success: true}
		}
		time.Sleep(2 * time.Second)

		// 点击完成
		finishBtn := ctx.Page.Locator("div[class^='extractFooter'] button:visible:has-text('完成')")
		if err := finishBtn.Click(); err != nil {
			utils.Warn(fmt.Sprintf("[-] 点击完成按钮失败: %v", err))
			return uploader.StepResult{Step: uploader.StepSetCover, Success: true}
		}

		ctx.ReportProgress(uploader.StepSetCover, 62, "封面设置完成")
		return uploader.StepResult{Step: uploader.StepSetCover, Success: true}
	}
}

// StepSetLocation 设置位置（基础版本）
// 注意：此方法被 StepSetLocationEnhanced() 替代，保留用于回退和测试
func (u *Uploader) StepSetLocation() uploader.StepFunc {
	return func(ctx *uploader.Context) uploader.StepResult {
		if ctx.Task.Location == "" {
			return uploader.StepResult{Step: uploader.StepFillContent, Success: true}
		}

		ctx.ReportProgress(uploader.StepFillContent, 64, fmt.Sprintf("正在设置位置: %s...", ctx.Task.Location))

		// 点击地点输入框
		locEle := ctx.Page.Locator("div.d-text.d-select-placeholder.d-text-ellipsis.d-text-nowrap")
		if err := locEle.Click(); err != nil {
			utils.Warn(fmt.Sprintf("[-] 点击位置输入框失败: %v", err))
			return uploader.StepResult{Step: uploader.StepFillContent, Success: true}
		}
		time.Sleep(1 * time.Second)

		// 输入位置名称
		if err := ctx.Page.Keyboard().Type(ctx.Task.Location); err != nil {
			utils.Warn(fmt.Sprintf("[-] 输入位置失败: %v", err))
			return uploader.StepResult{Step: uploader.StepFillContent, Success: true}
		}
		time.Sleep(3 * time.Second)

		// 尝试选择位置
		flexibleXPath := fmt.Sprintf(
			"//div[contains(@class, 'd-popover') and contains(@class, 'd-dropdown')]"+
				"//div[contains(@class, 'd-options-wrapper')]"+
				"//div[contains(@class, 'd-grid') and contains(@class, 'd-options')]"+
				"//div[contains(@class, 'name') and text()='%s']",
			ctx.Task.Location,
		)

		locationOption := ctx.Page.Locator(flexibleXPath)
		count, err := locationOption.Count()
		if err != nil || count == 0 {
			utils.Warn(fmt.Sprintf("[-] 未找到位置选项: %s", ctx.Task.Location))
			return uploader.StepResult{Step: uploader.StepFillContent, Success: true}
		}

		if err := locationOption.Click(); err != nil {
			utils.Warn(fmt.Sprintf("[-] 点击位置选项失败: %v", err))
			return uploader.StepResult{Step: uploader.StepFillContent, Success: true}
		}

		ctx.ReportProgress(uploader.StepFillContent, 66, "位置设置完成")
		return uploader.StepResult{Step: uploader.StepFillContent, Success: true}
	}
}

// StepSetScheduleXiaohongshu 小红书专用定时发布步骤（基础版本）
// 注意：此方法被 StepSetScheduleXiaohongshuEnhanced() 替代，保留用于回退和测试
func (u *Uploader) StepSetScheduleXiaohongshu() uploader.StepFunc {
	return func(ctx *uploader.Context) uploader.StepResult {
		if ctx.Task.ScheduleTime == nil || *ctx.Task.ScheduleTime == "" {
			return uploader.StepResult{Step: uploader.StepSetSchedule, Success: true}
		}

		ctx.ReportProgress(uploader.StepSetSchedule, 75, "正在设置定时发布...")

		// 点击定时发布
		labelElement := ctx.Page.Locator("label:has-text('定时发布')")
		if err := labelElement.Click(); err != nil {
			return uploader.StepResult{Step: uploader.StepSetSchedule, Success: false, Error: err}
		}
		time.Sleep(1 * time.Second)

		// 设置日期时间
		scheduleInput := ctx.Page.Locator(".el-input__inner[placeholder=\"选择日期和时间\"]")
		if err := scheduleInput.Click(); err != nil {
			return uploader.StepResult{Step: uploader.StepSetSchedule, Success: false, Error: err}
		}
		if err := ctx.Page.Keyboard().Press("Control+KeyA"); err != nil {
			return uploader.StepResult{Step: uploader.StepSetSchedule, Success: false, Error: err}
		}
		if err := ctx.Page.Keyboard().Type(*ctx.Task.ScheduleTime); err != nil {
			return uploader.StepResult{Step: uploader.StepSetSchedule, Success: false, Error: err}
		}
		if err := ctx.Page.Keyboard().Press("Enter"); err != nil {
			return uploader.StepResult{Step: uploader.StepSetSchedule, Success: false, Error: err}
		}

		ctx.ReportProgress(uploader.StepSetSchedule, 80, "定时发布设置完成")
		return uploader.StepResult{Step: uploader.StepSetSchedule, Success: true}
	}
}

// StepClickPublishXiaohongshu 小红书专用发布步骤（基础版本）
// 注意：此方法被 StepClickPublishXiaohongshuEnhanced() 替代，保留用于回退和测试
func (u *Uploader) StepClickPublishXiaohongshu() uploader.StepFunc {
	return func(ctx *uploader.Context) uploader.StepResult {
		ctx.ReportProgress(uploader.StepPublish, 85, "正在发布...")

		// 判断是定时发布还是立即发布
		buttonText := "发布"
		if ctx.Task.ScheduleTime != nil && *ctx.Task.ScheduleTime != "" {
			buttonText = "定时发布"
		}

		button := ctx.Page.Locator(fmt.Sprintf("button:has-text('%s')", buttonText))
		if err := button.Click(); err != nil {
			return uploader.StepResult{Step: uploader.StepPublish, Success: false, Error: err}
		}

		ctx.ReportProgress(uploader.StepPublish, 88, "等待发布结果...")

		// 等待发布成功
		timeout := time.After(30 * time.Second)
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-timeout:
				ctx.TakeScreenshot("publish_timeout")
				return uploader.StepResult{Step: uploader.StepPublish, Success: false, Error: fmt.Errorf("publish timeout")}
			case <-ticker.C:
				// 检查是否跳转到成功页面
				url := ctx.Page.URL()
				if strings.Contains(url, "/publish/success") {
					// 保存cookie
					if ctx.BrowserCtx != nil {
						if err := ctx.BrowserCtx.SaveCookies(); err != nil {
							utils.Warn(fmt.Sprintf("[-] 保存cookie失败: %v", err))
						} else {
							utils.Info("[-] Cookie已保存")
						}
					}
					ctx.ReportProgress(uploader.StepPublish, 95, "发布成功")
					ctx.TakeScreenshot("publish_success")
					return uploader.StepResult{Step: uploader.StepPublish, Success: true}
				}

				// 截图记录发布状态
				ctx.TakeScreenshot("publishing")
			}
		}
	}
}

// Login 登录（增强版，带自然浏览路径和验证码检测）
func (u *Uploader) Login() error {
	ctx := context.Background()

	// 从浏览器池获取上下文
	browserCtx, err := browserPool.GetContext(ctx, u.GetCookiePath(), u.GetContextOptions())
	if err != nil {
		return fmt.Errorf("get browser context failed: %w", err)
	}
	defer browserCtx.Release()

	page, err := browserCtx.GetPage()
	if err != nil {
		return fmt.Errorf("get page failed: %w", err)
	}

	// 步骤1: 先访问小红书主页（模拟自然浏览路径）
	utils.Info("[-] 正在访问小红书主页...")
	if _, err := page.Goto("https://www.xiaohongshu.com", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	}); err != nil {
		utils.Warn(fmt.Sprintf("[-] 访问主页失败，直接访问登录页: %v", err))
	} else {
		// 模拟人类浏览行为
		utils.Info("[-] 模拟浏览主页...")
		if err := simulateHumanBehavior(page); err != nil {
			utils.Warn(fmt.Sprintf("[-] 模拟浏览行为失败: %v", err))
		}
		humanLikeDelay(2 * time.Second)
	}

	// 步骤2: 访问创作者中心登录页
	utils.Info("[-] 正在打开小红书创作者中心登录页...")
	if _, err := page.Goto("https://creator.xiaohongshu.com/login", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	}); err != nil {
		return fmt.Errorf("goto login page failed: %w", err)
	}

	// 等待页面完全加载
	if err := browserCtx.WaitForPageLoad(); err != nil {
		utils.Warn(fmt.Sprintf("[-] 等待页面加载警告: %v", err))
	}
	humanLikeDelay(3 * time.Second)

	// 步骤3: 检测验证码/滑块
	utils.Info("[-] 检测登录页面状态...")
	hasCaptcha, captchaType, err := u.detectCaptcha(page)
	if err != nil {
		utils.Warn(fmt.Sprintf("[-] 验证码检测出错: %v", err))
	}
	if hasCaptcha {
		utils.Warn(fmt.Sprintf("[-] 检测到%s，请手动完成验证", captchaType))
	}

	// 等待用户登录
	utils.Info("[-] 请在浏览器窗口中完成登录...")

	// 检测登录成功
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	captchaDetected := false
	for {
		select {
		case <-timeout:
			return fmt.Errorf("login timeout")
		case <-ticker.C:
			// 检查页面是否已关闭
			if browserCtx.IsPageClosed() {
				utils.Error("[-] 页面已被关闭，登录中断")
				return fmt.Errorf("page closed by user")
			}

			// 检测验证码（登录过程中可能出现）
			if !captchaDetected {
				hasCaptchaNow, captchaTypeNow, _ := u.detectCaptcha(page)
				if hasCaptchaNow {
					captchaDetected = true
					utils.Warn(fmt.Sprintf("[-] 检测到%s，请手动完成", captchaTypeNow))
				}
			}

			// 检测反爬虫
			hasAntiBot, antiBotMsg, _ := u.detectAntiBot(page)
			if hasAntiBot {
				utils.Error(fmt.Sprintf("[-] 触发反爬虫检测: %s", antiBotMsg))
				return fmt.Errorf("anti-bot detected: %s", antiBotMsg)
			}

			// 检查当前URL
			url := page.URL()
			utils.Info(fmt.Sprintf("[-] 当前URL: %s", url))

			// 检查是否已进入创作者中心主页（登录成功后跳转的页面）
			if url == "https://creator.xiaohongshu.com/new/home" ||
				url == "https://creator.xiaohongshu.com/creator/home" ||
				strings.Contains(url, "/new/home") {
				// 检查创作者中心特征元素
				publishBtn, _ := page.GetByText("发布笔记").Count()
				homeMenu, _ := page.GetByText("首页").Count()
				contentManage, _ := page.GetByText("笔记管理").Count()

				utils.Info(fmt.Sprintf("[-] 检测到元素: 发布笔记=%d, 首页=%d, 笔记管理=%d",
					publishBtn, homeMenu, contentManage))

				if publishBtn > 0 || homeMenu > 0 || contentManage > 0 {
					utils.Info("[-] 登录成功，已进入创作者中心")
					// 保存Cookie
					if err := browserCtx.SaveCookies(); err != nil {
						utils.Warn(fmt.Sprintf("[-] 保存Cookie失败: %v", err))
					} else {
						utils.Info("[-] Cookie已保存")
					}
					return nil
				}
			}

			// 检查是否有用户头像（备用检测方式）
			avatarCount, _ := page.Locator(".user-avatar, .avatar, img[src*='avatar']").Count()
			if avatarCount > 0 {
				utils.Info("[-] 检测到用户头像，登录成功")
				// 保存Cookie
				if err := browserCtx.SaveCookies(); err != nil {
					utils.Warn(fmt.Sprintf("[-] 保存Cookie失败: %v", err))
				} else {
					utils.Info("[-] Cookie已保存")
				}
				return nil
			}
		}
	}
}
