package kuaishou

import (
	"context"
	"fmt"
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

// Uploader 快手上传器
type Uploader struct {
	*uploader.Base
}

// NewUploader 创建上传器
func NewUploader(cookiePath string) *Uploader {
	initBrowserPool()
	return &Uploader{
		Base: uploader.NewBase("kuaishou", cookiePath, browserPool),
	}
}

// Platform 返回平台名称
func (u *Uploader) Platform() string {
	return "kuaishou"
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

	// 访问快手创作者中心
	utils.Info("[-] 正在验证快手登录状态...")
	if _, err := page.Goto("https://cp.kuaishou.com/"); err != nil {
		return false, fmt.Errorf("goto creator page failed: %w", err)
	}

	time.Sleep(3 * time.Second)

	// 检查当前URL
	currentURL := page.URL()
	utils.Info(fmt.Sprintf("[-] 当前URL: %s", currentURL))

	// 如果被重定向到登录页，说明Cookie无效
	if currentURL == "https://cp.kuaishou.com/login" ||
		currentURL == "https://passport.kuaishou.com/" {
		utils.Info("[-] 被重定向到登录页，Cookie 无效")
		return false, nil
	}

	// 检查是否有登录页特征
	loginCount, _ := page.Locator("button:has-text(\"登录\")").Count()
	if loginCount > 0 {
		utils.Info("[-] 检测到登录按钮，Cookie 无效")
		return false, nil
	}

	// 检查是否有创作者中心特征元素
	publishBtn, _ := page.GetByText("发布视频").Count()
	contentManage, _ := page.GetByText("内容管理").Count()
	dataCenter, _ := page.GetByText("数据中心").Count()

	if publishBtn > 0 || contentManage > 0 || dataCenter > 0 {
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
		u.StepNavigate("https://cp.kuaishou.com/article/publish/video"),

		// 2. 处理新功能引导（"我知道了"按钮）
		u.StepHandleNewFeatureGuide(),

		// 3. 上传视频
		u.StepUploadKuaishouVideo(task.VideoPath),

		// 4. 填写描述和标签
		u.StepFillKuaishouDescAndTags(task.Title, task.Tags),

		// 5. 设置定时发布（如果有）
		u.StepSetScheduleKuaishou(task.ScheduleTime),

		// 6. 点击发布并确认
		u.StepClickPublishWithConfirm(),
	}

	return u.Execute(ctx, task, steps)
}

// StepHandleNewFeatureGuide 处理新功能引导（"我知道了"按钮）
func (u *Uploader) StepHandleNewFeatureGuide() uploader.StepFunc {
	return func(ctx *uploader.Context) uploader.StepResult {
		// 等待一下让页面完全加载
		time.Sleep(2 * time.Second)

		// 检测"我知道了"按钮（新功能引导）
		newFeatureBtn := ctx.Page.Locator("button[type='button'] span:has-text('我知道了')")
		count, err := newFeatureBtn.Count()
		if err != nil {
			// 出错也继续，不是关键步骤
			return uploader.StepResult{Step: uploader.StepNavigate, Success: true}
		}

		if count > 0 {
			utils.Info("[-] 检测到新功能引导，点击'我知道了'...")
			if err := newFeatureBtn.Click(); err != nil {
				utils.Warn(fmt.Sprintf("[-] 点击新功能引导按钮失败: %v", err))
				// 失败也继续，不是关键步骤
			} else {
				utils.Info("[-] 已关闭新功能引导")
				time.Sleep(1 * time.Second)
			}
		}

		return uploader.StepResult{Step: uploader.StepNavigate, Success: true}
	}
}

// StepUploadKuaishouVideo 快手上传视频步骤（使用文件选择器模式）
func (u *Uploader) StepUploadKuaishouVideo(videoPath string) uploader.StepFunc {
	return func(ctx *uploader.Context) uploader.StepResult {
		ctx.ReportProgress(uploader.StepUploadMedia, 20, "开始上传视频...")

		// 使用与Python版本相同的选择器: button[class^='_upload-btn']
		uploadButton := ctx.Page.Locator("button[class^='_upload-btn']")

		// 确保按钮可见
		if err := uploadButton.WaitFor(playwright.LocatorWaitForOptions{
			State:   playwright.WaitForSelectorStateVisible,
			Timeout: playwright.Float(10000),
		}); err != nil {
			return uploader.StepResult{Step: uploader.StepUploadMedia, Success: false, Error: fmt.Errorf("upload button not visible: %w", err)}
		}

		ctx.ReportProgress(uploader.StepUploadMedia, 25, "点击上传按钮...")

		// 使用文件选择器模式（与Python版本一致）
		fileChooser, err := ctx.Page.ExpectFileChooser(func() error {
			return uploadButton.Click()
		})
		if err != nil {
			return uploader.StepResult{Step: uploader.StepUploadMedia, Success: false, Error: fmt.Errorf("expect file chooser failed: %w", err)}
		}

		if err := fileChooser.SetFiles(videoPath); err != nil {
			return uploader.StepResult{Step: uploader.StepUploadMedia, Success: false, Error: fmt.Errorf("set files failed: %w", err)}
		}

		ctx.ReportProgress(uploader.StepUploadMedia, 30, "视频文件已选择，等待上传完成...")

		// 等待上传完成（使用Python版本的检测方式：检测"上传中"文本消失）
		maxRetries := 60 // 最大等待2分钟
		retryCount := 0

		for retryCount < maxRetries {
			// 检查"上传中"文本
			uploadingCount, _ := ctx.Page.Locator("text=上传中").Count()

			if uploadingCount == 0 {
				// 再检查一下是否有"上传成功"标志
				successCount, _ := ctx.Page.Locator("[class*='success'] >> text=上传成功").Count()
				if successCount > 0 {
					ctx.ReportProgress(uploader.StepUploadMedia, 50, "视频上传成功")
					return uploader.StepResult{Step: uploader.StepUploadMedia, Success: true}
				}

				// 如果没有"上传中"了，也认为是成功了
				ctx.ReportProgress(uploader.StepUploadMedia, 50, "视频上传完成")
				return uploader.StepResult{Step: uploader.StepUploadMedia, Success: true}
			}

			if retryCount%5 == 0 {
				ctx.ReportProgress(uploader.StepUploadMedia, 30+retryCount/2, "正在上传视频中...")
				utils.Info("[-] 正在上传视频中...")
			}

			time.Sleep(2 * time.Second)
			retryCount++
		}

		return uploader.StepResult{Step: uploader.StepUploadMedia, Success: false, Error: fmt.Errorf("upload timeout after 2 minutes")}
	}
}

// StepFillKuaishouDescAndTags 快手填写描述和标签步骤
func (u *Uploader) StepFillKuaishouDescAndTags(title string, tags []string) uploader.StepFunc {
	return func(ctx *uploader.Context) uploader.StepResult {
		ctx.ReportProgress(uploader.StepFillContent, 55, "正在填充标题和话题...")

		// 点击描述区域（与Python版本一致）
		descArea := ctx.Page.GetByText("描述").Locator("xpath=following-sibling::div")
		if err := descArea.Click(); err != nil {
			// 如果上面的方式失败，尝试使用textarea
			descLocator := ctx.Page.Locator("textarea[placeholder*='描述']")
			if err := descLocator.Click(); err != nil {
				return uploader.StepResult{Step: uploader.StepFillContent, Success: false, Error: fmt.Errorf("click description area failed: %w", err)}
			}
		}

		// 清除原有内容（与Python版本一致）
		if err := ctx.Page.Keyboard().Press("Backspace"); err != nil {
			return uploader.StepResult{Step: uploader.StepFillContent, Success: false, Error: fmt.Errorf("press backspace failed: %w", err)}
		}
		if err := ctx.Page.Keyboard().Press("Control+KeyA"); err != nil {
			return uploader.StepResult{Step: uploader.StepFillContent, Success: false, Error: fmt.Errorf("press control+a failed: %w", err)}
		}
		if err := ctx.Page.Keyboard().Press("Delete"); err != nil {
			return uploader.StepResult{Step: uploader.StepFillContent, Success: false, Error: fmt.Errorf("press delete failed: %w", err)}
		}

		// 输入标题
		if err := ctx.Page.Keyboard().Type(title); err != nil {
			return uploader.StepResult{Step: uploader.StepFillContent, Success: false, Error: fmt.Errorf("type title failed: %w", err)}
		}
		if err := ctx.Page.Keyboard().Press("Enter"); err != nil {
			return uploader.StepResult{Step: uploader.StepFillContent, Success: false, Error: fmt.Errorf("press enter failed: %w", err)}
		}

		ctx.ReportProgress(uploader.StepFillContent, 60, "标题填写完成")

		// 添加话题标签（快手最多3个，与Python版本一致）
		maxTags := 3
		if len(tags) < maxTags {
			maxTags = len(tags)
		}

		for index, tag := range tags[:maxTags] {
			ctx.ReportProgress(uploader.StepAddTags, 60+index*5, fmt.Sprintf("正在添加第%d个话题", index+1))
			if err := ctx.Page.Keyboard().Type(fmt.Sprintf("#%s ", tag)); err != nil {
				return uploader.StepResult{Step: uploader.StepAddTags, Success: false, Error: fmt.Errorf("type tag failed: %w", err)}
			}
			time.Sleep(2 * time.Second) // 与Python版本一致
		}

		ctx.ReportProgress(uploader.StepAddTags, 75, fmt.Sprintf("已添加 %d 个话题", maxTags))
		return uploader.StepResult{Step: uploader.StepFillContent, Success: true}
	}
}

// StepSetScheduleKuaishou 快手设置定时发布步骤
func (u *Uploader) StepSetScheduleKuaishou(scheduleTime *string) uploader.StepFunc {
	return func(ctx *uploader.Context) uploader.StepResult {
		if scheduleTime == nil || *scheduleTime == "" {
			return uploader.StepResult{Step: uploader.StepSetSchedule, Success: true}
		}

		ctx.ReportProgress(uploader.StepSetSchedule, 76, "正在设置定时发布...")

		// 点击定时发布（与Python版本一致：使用label选择器）
		scheduleLabel := ctx.Page.Locator("label:has-text('发布时间')")
		if err := scheduleLabel.Locator("xpath=following-sibling::div").Locator(".ant-radio-input").Nth(1).Click(); err != nil {
			// 如果上面的方式失败，尝试直接点击"定时发布"
			if err := ctx.Page.GetByText("定时发布").Click(); err != nil {
				return uploader.StepResult{Step: uploader.StepSetSchedule, Success: false, Error: fmt.Errorf("click schedule failed: %w", err)}
			}
		}
		time.Sleep(1 * time.Second)

		// 设置日期时间（与Python版本一致）
		scheduleInput := ctx.Page.Locator("div.ant-picker-input input[placeholder*='选择日期时间']")
		if err := scheduleInput.Click(); err != nil {
			// 尝试其他选择器
			scheduleInput = ctx.Page.Locator("input[placeholder*='时间']")
			if err := scheduleInput.Click(); err != nil {
				return uploader.StepResult{Step: uploader.StepSetSchedule, Success: false, Error: fmt.Errorf("click schedule input failed: %w", err)}
			}
		}
		time.Sleep(1 * time.Second)

		if err := ctx.Page.Keyboard().Press("Control+KeyA"); err != nil {
			return uploader.StepResult{Step: uploader.StepSetSchedule, Success: false, Error: fmt.Errorf("press control+a failed: %w", err)}
		}
		if err := ctx.Page.Keyboard().Type(*scheduleTime); err != nil {
			return uploader.StepResult{Step: uploader.StepSetSchedule, Success: false, Error: fmt.Errorf("type schedule time failed: %w", err)}
		}
		if err := ctx.Page.Keyboard().Press("Enter"); err != nil {
			return uploader.StepResult{Step: uploader.StepSetSchedule, Success: false, Error: fmt.Errorf("press enter failed: %w", err)}
		}

		ctx.ReportProgress(uploader.StepSetSchedule, 80, "定时发布设置完成")
		return uploader.StepResult{Step: uploader.StepSetSchedule, Success: true}
	}
}

// StepClickPublishWithConfirm 点击发布并处理确认按钮
func (u *Uploader) StepClickPublishWithConfirm() uploader.StepFunc {
	return func(ctx *uploader.Context) uploader.StepResult {
		ctx.ReportProgress(uploader.StepPublish, 85, "正在发布...")

		// 循环处理发布和确认（与Python版本一致）
		maxAttempts := 30
		for attempt := 0; attempt < maxAttempts; attempt++ {
			// 检查页面是否已关闭
			if ctx.BrowserCtx != nil && ctx.BrowserCtx.IsPageClosed() {
				utils.Error("[-] 页面已被关闭，发布中断")
				return uploader.StepResult{Step: uploader.StepPublish, Success: false, Error: fmt.Errorf("page closed by user")}
			}

			// 点击发布按钮
			publishButton := ctx.Page.GetByText("发布", playwright.PageGetByTextOptions{Exact: playwright.Bool(true)})
			count, _ := publishButton.Count()
			if count > 0 {
				if err := publishButton.Click(); err != nil {
					utils.Warn(fmt.Sprintf("[-] 点击发布按钮失败: %v", err))
				}
			}

			time.Sleep(1 * time.Second)

			// 检查确认发布按钮
			confirmButton := ctx.Page.GetByText("确认发布")
			confirmCount, _ := confirmButton.Count()
			if confirmCount > 0 {
				utils.Info("[-] 检测到确认发布按钮，点击确认...")
				if err := confirmButton.Click(); err != nil {
					utils.Warn(fmt.Sprintf("[-] 点击确认发布按钮失败: %v", err))
				}
			}

			// 检查是否发布成功（URL跳转）
			currentURL := ctx.Page.URL()
			if currentURL == "https://cp.kuaishou.com/article/manage/video?status=2&from=publish" {
				ctx.ReportProgress(uploader.StepPublish, 95, "视频发布成功")
				ctx.TakeScreenshot("publish_success")
				utils.Info("[-] 视频发布成功")
				return uploader.StepResult{Step: uploader.StepPublish, Success: true}
			}

			// 检查成功提示
			successCount, _ := ctx.Page.Locator("text=发布成功").Count()
			if successCount > 0 {
				visible, _ := ctx.Page.Locator("text=发布成功").IsVisible()
				if visible {
					ctx.ReportProgress(uploader.StepPublish, 95, "视频发布成功")
					ctx.TakeScreenshot("publish_success")
					utils.Info("[-] 视频发布成功")
					return uploader.StepResult{Step: uploader.StepPublish, Success: true}
				}
			}

			utils.Info(fmt.Sprintf("[-] 视频正在发布中... (尝试 %d/%d)", attempt+1, maxAttempts))
			ctx.TakeScreenshot(fmt.Sprintf("publishing_attempt_%d", attempt+1))
			time.Sleep(1 * time.Second)
		}

		ctx.TakeScreenshot("publish_timeout")
		return uploader.StepResult{Step: uploader.StepPublish, Success: false, Error: fmt.Errorf("publish timeout after %d attempts", maxAttempts)}
	}
}

// Login 登录
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

	// 访问发布页面（需要登录才能访问）
	utils.Info("[-] 正在打开快手创作者中心...")
	if _, err := page.Goto("https://cp.kuaishou.com/article/publish/video", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	}); err != nil {
		return fmt.Errorf("goto login page failed: %w", err)
	}

	// 等待页面完全加载
	if err := browserCtx.WaitForPageLoad(); err != nil {
		utils.Warn(fmt.Sprintf("[-] 等待页面加载警告: %v", err))
	}
	time.Sleep(3 * time.Second)

	utils.Info("[-] 请在浏览器窗口中完成登录，登录成功后会自动保存")

	// 检测登录成功
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

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

			// 多重检测机制
			loginBtn1, _ := page.Locator("button:has-text(\"登录\")").Count()
			loginBtn2, _ := page.GetByText("手机号登录").Count()
			loginBtn3, _ := page.GetByText("验证码登录").Count()

			currentURL := page.URL()
			isLoginPage := currentURL == "https://cp.kuaishou.com/" ||
				currentURL == "https://passport.kuaishou.com/" ||
				currentURL == "https://id.kuaishou.com/"

			publishBtn, _ := page.GetByText("发布视频").Count()
			uploadBtn, _ := page.GetByText("上传视频").Count()
			contentArea, _ := page.Locator(".content-publish").Count()

			// 如果还在登录页面，继续等待
			if isLoginPage || loginBtn1 > 0 || loginBtn2 > 0 || loginBtn3 > 0 {
				continue
			}

			// 如果检测到创作者中心特征元素，说明登录成功
			if publishBtn > 0 || uploadBtn > 0 || contentArea > 0 {
				utils.Info("[-] 登录成功，已进入创作者中心")
				return nil
			}
		}
	}
}
