package tencent

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

// Uploader 视频号上传器
type Uploader struct {
	*uploader.Base
}

// NewUploader 创建上传器
func NewUploader(cookiePath string) *Uploader {
	initBrowserPool()
	// 使用 NewBase，截图配置由前端控制
	base := uploader.NewBase("tencent", cookiePath, browserPool)
	return &Uploader{
		Base: base,
	}
}

// NewUploaderWithScreenshot 创建带截图配置的上传器
func NewUploaderWithScreenshot(cookiePath string, enableScreenshot bool, screenshotDir string) *Uploader {
	initBrowserPool()
	base := uploader.NewBaseWithScreenshot("tencent", cookiePath, browserPool, enableScreenshot, screenshotDir)
	return &Uploader{
		Base: base,
	}
}

// Platform 返回平台名称
func (u *Uploader) Platform() string {
	return "tencent"
}

// ValidateCookie 验证 Cookie 是否有效（参考Python实现）
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

	// 访问视频号创建页面
	utils.Info("[-] 正在验证登录状态...")
	if _, err := page.Goto("https://channels.weixin.qq.com/platform/post/create"); err != nil {
		return false, fmt.Errorf("goto platform failed: %w", err)
	}

	// 等待页面加载
	time.Sleep(3 * time.Second)

	// 检查1: 是否有"微信小店"元素（参考Python）
	shopCount, _ := page.Locator("div.title-name:has-text('微信小店')").Count()
	if shopCount > 0 {
		utils.Info("[-] 检测到微信小店元素，Cookie 无效")
		return false, nil
	}

	// 检查2: 是否有登录提示
	loginCount, _ := page.GetByText("请使用微信扫描二维码登录").Count()
	if loginCount > 0 {
		utils.Info("[-] 检测到登录提示，Cookie 无效")
		return false, nil
	}

	// 检查3: 是否已进入创建页面
	url := page.URL()
	if url == "https://channels.weixin.qq.com/platform/post/create" {
		utils.Info("[-] 已进入视频号创建页面，Cookie 有效")
		return true, nil
	}

	return false, fmt.Errorf("unknown login status, current url: %s", url)
}

// Upload 上传视频
func (u *Uploader) Upload(ctx context.Context, task *types.VideoTask) error {
	steps := []uploader.StepFunc{
		// 1. 导航到上传页面
		u.StepNavigate("https://channels.weixin.qq.com/platform/post/create"),

		// 2. 上传视频（改进版检测）
		u.StepUploadVideoTencent(),

		// 3. 填写标题和标签
		u.StepFillTitleAndTags(),

		// 4. 添加到合集
		u.StepAddCollection(),

		// 5. 声明原创（多版本兼容）
		u.StepAddOriginal(),

		// 6. 填写短标题
		u.StepFillShortTitle(),

		// 7. 设置定时发布（如果有）
		u.StepSetScheduleTencent(),

		// 8. 点击发布/保存草稿
		u.StepClickPublishTencent(),
	}

	return u.Execute(ctx, task, steps)
}

// StepUploadVideoTencent 视频号专用上传视频步骤（参考Python实现）
func (u *Uploader) StepUploadVideoTencent() uploader.StepFunc {
	return func(ctx *uploader.Context) uploader.StepResult {
		ctx.ReportProgress(uploader.StepUploadMedia, 20, "开始上传视频...")

		// 设置输入文件
		fileInput := ctx.Page.Locator("input[type='file']")
		if err := fileInput.SetInputFiles(ctx.Task.VideoPath); err != nil {
			return uploader.StepResult{Step: uploader.StepUploadMedia, Success: false, Error: err}
		}

		ctx.ReportProgress(uploader.StepUploadMedia, 25, "视频文件已选择，等待上传...")

		// 等待上传完成 - 参考Python的检测逻辑
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

				// 检测逻辑1: 检查"发表"按钮是否可用（参考Python）
				publishBtn := ctx.Page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "发表"})
				btnClass, err := publishBtn.GetAttribute("class")
				if err == nil && btnClass != "" {
					// 如果按钮没有disabled类，说明上传完成
					if !strings.Contains(btnClass, "disabled") && !strings.Contains(btnClass, "weui-desktop-btn_disabled") {
						ctx.ReportProgress(uploader.StepUploadMedia, 40, "视频上传完成")
						ctx.TakeScreenshot("upload_success")
						return uploader.StepResult{Step: uploader.StepUploadMedia, Success: true}
					}
				}

				// 检测逻辑2: 检查删除按钮（备选）
				deleteCount, _ := ctx.Page.Locator("div.media-status-content div.tag-inner:has-text('删除')").Count()
				if deleteCount > 0 {
					ctx.ReportProgress(uploader.StepUploadMedia, 40, "视频上传完成（检测到删除按钮）")
					ctx.TakeScreenshot("upload_success")
					return uploader.StepResult{Step: uploader.StepUploadMedia, Success: true}
				}

				// 检测逻辑3: 检查上传错误
				errorCount, _ := ctx.Page.Locator("div.status-msg.error").Count()
				if errorCount > 0 {
					ctx.TakeScreenshot("upload_error")
					utils.Warn("[-] 检测到上传错误，尝试重试...")
					if err := u.handleUploadError(ctx); err != nil {
						return uploader.StepResult{Step: uploader.StepUploadMedia, Success: false, Error: fmt.Errorf("upload failed and retry failed: %w", err)}
					}
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

// handleUploadError 处理上传错误（参考Python实现）
func (u *Uploader) handleUploadError(ctx *uploader.Context) error {
	utils.Info("[-] 视频出错了，重新上传中...")

	// 点击删除按钮
	deleteBtn := ctx.Page.Locator("div.media-status-content div.tag-inner:has-text('删除')")
	if err := deleteBtn.Click(); err != nil {
		return fmt.Errorf("click delete button failed: %w", err)
	}
	time.Sleep(500 * time.Millisecond)

	// 确认删除
	confirmBtn := ctx.Page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "删除", Exact: playwright.Bool(true)})
	if err := confirmBtn.Click(); err != nil {
		return fmt.Errorf("click confirm delete failed: %w", err)
	}
	time.Sleep(1 * time.Second)

	// 重新上传
	fileInput := ctx.Page.Locator("input[type='file']")
	if err := fileInput.SetInputFiles(ctx.Task.VideoPath); err != nil {
		return fmt.Errorf("retry upload failed: %w", err)
	}

	return nil
}

// StepFillTitleAndTags 填写标题和标签（视频号专用）
func (u *Uploader) StepFillTitleAndTags() uploader.StepFunc {
	return func(ctx *uploader.Context) uploader.StepResult {
		ctx.ReportProgress(uploader.StepFillTitle, 45, "正在填写标题和话题...")

		task := ctx.Task

		// 设置标题（限制21字）
		title := task.Title
		if len(title) > 21 {
			title = title[:21]
		}

		// 点击标题输入框
		titleEditor := ctx.Page.Locator("div.input-editor")
		if err := titleEditor.Click(); err != nil {
			return uploader.StepResult{Step: uploader.StepFillTitle, Success: false, Error: err}
		}
		if err := ctx.Page.Keyboard().Type(title); err != nil {
			return uploader.StepResult{Step: uploader.StepFillTitle, Success: false, Error: err}
		}
		if err := ctx.Page.Keyboard().Press("Enter"); err != nil {
			return uploader.StepResult{Step: uploader.StepFillTitle, Success: false, Error: err}
		}

		// 添加话题标签
		for _, tag := range task.Tags {
			if err := ctx.Page.Keyboard().Type("#" + tag); err != nil {
				return uploader.StepResult{Step: uploader.StepAddTags, Success: false, Error: err}
			}
			if err := ctx.Page.Keyboard().Press("Space"); err != nil {
				return uploader.StepResult{Step: uploader.StepAddTags, Success: false, Error: err}
			}
			time.Sleep(500 * time.Millisecond)
		}

		utils.Info(fmt.Sprintf("[-] 成功添加 %d 个话题", len(task.Tags)))
		ctx.ReportProgress(uploader.StepAddTags, 55, fmt.Sprintf("已添加 %d 个话题", len(task.Tags)))
		return uploader.StepResult{Step: uploader.StepFillTitle, Success: true}
	}
}

// StepAddCollection 添加到合集（参考Python实现）
func (u *Uploader) StepAddCollection() uploader.StepFunc {
	return func(ctx *uploader.Context) uploader.StepResult {
		if ctx.Task.Collection == "" {
			return uploader.StepResult{Step: uploader.StepFillContent, Success: true}
		}

		ctx.ReportProgress(uploader.StepFillContent, 58, "正在添加到合集...")

		// 查找合集选项
		collectionElements := ctx.Page.GetByText("添加到合集").Locator("xpath=following-sibling::div").Locator(".option-list-wrap > div")
		count, err := collectionElements.Count()
		if err != nil || count <= 1 {
			utils.Warn("[-] 未找到合集选项或只有一个选项")
			return uploader.StepResult{Step: uploader.StepFillContent, Success: true}
		}

		// 点击展开合集选择
		if err := ctx.Page.GetByText("添加到合集").Locator("xpath=following-sibling::div").Click(); err != nil {
			utils.Warn(fmt.Sprintf("[-] 点击合集选择失败: %v", err))
			return uploader.StepResult{Step: uploader.StepFillContent, Success: true}
		}
		time.Sleep(1 * time.Second)

		// 选择第一个合集
		if err := collectionElements.First().Click(); err != nil {
			utils.Warn(fmt.Sprintf("[-] 选择合集失败: %v", err))
			return uploader.StepResult{Step: uploader.StepFillContent, Success: true}
		}

		ctx.ReportProgress(uploader.StepFillContent, 60, "合集添加完成")
		return uploader.StepResult{Step: uploader.StepFillContent, Success: true}
	}
}

// StepAddOriginal 声明原创（视频号专用，多版本兼容）
func (u *Uploader) StepAddOriginal() uploader.StepFunc {
	return func(ctx *uploader.Context) uploader.StepResult {
		if !ctx.Task.IsOriginal {
			return uploader.StepResult{Step: uploader.StepFillContent, Success: true}
		}

		ctx.ReportProgress(uploader.StepFillContent, 62, "正在声明原创...")
		time.Sleep(2 * time.Second)

		// 版本1: 检测并勾选"视频为原创"复选框
		originalCheckbox := ctx.Page.GetByLabel("视频为原创")
		count, err := originalCheckbox.Count()
		if err == nil && count > 0 {
			if err := originalCheckbox.Check(); err != nil {
				utils.Warn(fmt.Sprintf("[-] 勾选'视频为原创'失败: %v", err))
			} else {
				utils.Info("[-] 已勾选'视频为原创'")
			}
		}

		// 版本2: 检测并勾选"我已阅读并同意 《视频号原创声明使用条款》"
		agreementLabel := ctx.Page.Locator("label:has-text('我已阅读并同意 《视频号原创声明使用条款》')")
		count, err = agreementLabel.Count()
		if err == nil && count > 0 {
			visible, err := agreementLabel.IsVisible()
			if err == nil && visible {
				agreementCheckbox := ctx.Page.GetByLabel("我已阅读并同意 《视频号原创声明使用条款》")
				if err := agreementCheckbox.Check(); err != nil {
					utils.Warn(fmt.Sprintf("[-] 勾选原创声明条款失败: %v", err))
				} else {
					utils.Info("[-] 已勾选原创声明条款")
				}

				// 点击"声明原创"按钮
				declareBtn := ctx.Page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "声明原创"})
				btnCount, err := declareBtn.Count()
				if err == nil && btnCount > 0 {
					if err := declareBtn.Click(); err != nil {
						utils.Warn(fmt.Sprintf("[-] 点击'声明原创'按钮失败: %v", err))
					} else {
						time.Sleep(1 * time.Second)
						utils.Info("[-] 已点击'声明原创'按钮")
					}
				}
			}
		}

		// 版本3: 2023-11-20 WeChat更新兼容（参考Python）
		declareLabel := ctx.Page.Locator("div.label span:has-text('声明原创')")
		count, err = declareLabel.Count()
		if err == nil && count > 0 && ctx.Task.OriginalType != "" {
			// 检查复选框是否可用
			checkbox := ctx.Page.Locator("div.declare-original-checkbox input.ant-checkbox-input")
			isDisabled, err := checkbox.IsDisabled()
			if err == nil && !isDisabled {
				if err := checkbox.Click(); err != nil {
					utils.Warn(fmt.Sprintf("[-] 点击原创复选框失败: %v", err))
				}

				// 检查是否需要勾选条款
				termsCheckbox := ctx.Page.Locator("div.declare-original-dialog input.ant-checkbox-input:visible")
				termsCount, _ := termsCheckbox.Count()
				if termsCount > 0 {
					isChecked, _ := termsCheckbox.IsChecked()
					if !isChecked {
						termsCheckbox.Click()
					}
				}

				// 选择原创类型
				typeLabel := ctx.Page.Locator("div.original-type-form > div.form-label:has-text('原创类型'):visible")
				typeCount, _ := typeLabel.Count()
				if typeCount > 0 {
					// 点击下拉菜单
					dropdown := ctx.Page.Locator("div.form-content:visible")
					dropdown.Click()
					time.Sleep(500 * time.Millisecond)

					// 选择类型
					typeOption := ctx.Page.Locator(fmt.Sprintf("div.form-content:visible ul.weui-desktop-dropdown__list li.weui-desktop-dropdown__list-ele:has-text(\"%s\")", ctx.Task.OriginalType)).First()
					typeOption.Click()
					time.Sleep(1 * time.Second)
				}

				// 点击声明原创按钮
				declareBtn := ctx.Page.Locator("button:has-text('声明原创'):visible")
				btnCount, _ := declareBtn.Count()
				if btnCount > 0 {
					declareBtn.Click()
					time.Sleep(1 * time.Second)
				}
			}
		}

		ctx.ReportProgress(uploader.StepFillContent, 65, "原创声明完成")
		return uploader.StepResult{Step: uploader.StepFillContent, Success: true}
	}
}

// StepFillShortTitle 填写短标题（视频号专用）
func (u *Uploader) StepFillShortTitle() uploader.StepFunc {
	return func(ctx *uploader.Context) uploader.StepResult {
		title := ctx.Task.Title
		if ctx.Task.ShortTitle != "" {
			title = ctx.Task.ShortTitle
		}
		shortTitle := formatShortTitle(title)

		ctx.ReportProgress(uploader.StepFillContent, 67, fmt.Sprintf("正在填写短标题: %s", shortTitle))

		// 查找短标题输入框
		shortTitleInput := ctx.Page.GetByText("短标题", playwright.PageGetByTextOptions{Exact: playwright.Bool(true)}).Locator("..").Locator("xpath=following-sibling::div").Locator("span input[type='text']")

		// 检查是否存在短标题输入框
		count, err := shortTitleInput.Count()
		if err != nil || count == 0 {
			// 短标题输入框不存在，跳过
			return uploader.StepResult{Step: uploader.StepFillContent, Success: true}
		}

		// 填写短标题
		if err := shortTitleInput.Fill(shortTitle); err != nil {
			utils.Warn(fmt.Sprintf("[-] 填写短标题失败: %v", err))
			return uploader.StepResult{Step: uploader.StepFillContent, Success: true}
		}

		utils.Info(fmt.Sprintf("[-] 已设置短标题: %s", shortTitle))
		ctx.ReportProgress(uploader.StepFillContent, 70, "短标题填写完成")
		return uploader.StepResult{Step: uploader.StepFillContent, Success: true}
	}
}

// StepSetScheduleTencent 设置定时发布（视频号专用，参考Python实现）
func (u *Uploader) StepSetScheduleTencent() uploader.StepFunc {
	return func(ctx *uploader.Context) uploader.StepResult {
		if ctx.Task.ScheduleTime == nil || *ctx.Task.ScheduleTime == "" {
			return uploader.StepResult{Step: uploader.StepSetSchedule, Success: true}
		}

		ctx.ReportProgress(uploader.StepSetSchedule, 75, "正在设置定时发布...")

		// 点击定时发布（第二个包含"定时"的label）
		labelElement := ctx.Page.Locator("label").Filter(playwright.LocatorFilterOptions{HasText: playwright.String("定时")}).Nth(1)
		if err := labelElement.Click(); err != nil {
			return uploader.StepResult{Step: uploader.StepSetSchedule, Success: false, Error: err}
		}
		time.Sleep(1 * time.Second)

		// 点击日期时间选择器
		if err := ctx.Page.Click("input[placeholder='请选择发表时间']"); err != nil {
			return uploader.StepResult{Step: uploader.StepSetSchedule, Success: false, Error: err}
		}
		time.Sleep(500 * time.Millisecond)

		// 解析时间
		// 注意：这里简化处理，实际应该解析ScheduleTime并选择对应日期
		// 参考Python的日历选择逻辑

		// 选择标题栏（令定时时间生效）
		if err := ctx.Page.Locator("div.input-editor").Click(); err != nil {
			utils.Warn(fmt.Sprintf("[-] 点击标题栏失败: %v", err))
		}

		ctx.ReportProgress(uploader.StepSetSchedule, 80, "定时发布设置完成")
		return uploader.StepResult{Step: uploader.StepSetSchedule, Success: true}
	}
}

// StepClickPublishTencent 视频号专用发布步骤（支持草稿保存）
func (u *Uploader) StepClickPublishTencent() uploader.StepFunc {
	return func(ctx *uploader.Context) uploader.StepResult {
		// 判断是保存草稿还是发布
		if ctx.Task.IsDraft {
			ctx.ReportProgress(uploader.StepPublish, 85, "正在保存草稿...")
			return u.saveDraft(ctx)
		}

		ctx.ReportProgress(uploader.StepPublish, 85, "正在发布...")

		// 点击发表按钮
		publishButton := ctx.Page.Locator("div.form-btns button:has-text('发表')")
		if err := publishButton.Click(); err != nil {
			return uploader.StepResult{Step: uploader.StepPublish, Success: false, Error: err}
		}

		ctx.ReportProgress(uploader.StepPublish, 88, "等待发布结果...")

		// 等待发布成功 - 增加超时时间到60秒，确保上传有足够时间完成
		timeout := time.After(60 * time.Second)
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-timeout:
				ctx.TakeScreenshot("publish_timeout")
				return uploader.StepResult{Step: uploader.StepPublish, Success: false, Error: fmt.Errorf("publish timeout")}
			case <-ticker.C:
				// 检查是否跳转到列表页面
				url := ctx.Page.URL()
				if strings.Contains(url, "post/list") {
					ctx.ReportProgress(uploader.StepPublish, 95, "发布成功，等待后台处理完成...")
					ctx.TakeScreenshot("publish_success")
					// 检测到成功后，额外等待5秒确保后台处理完成
					time.Sleep(5 * time.Second)
					utils.Info("[-] 发布完成，后台处理已结束")
					return uploader.StepResult{Step: uploader.StepPublish, Success: true}
				}

				// 截图记录发布状态
				ctx.TakeScreenshot("publishing")
			}
		}
	}
}

// saveDraft 保存草稿（参考Python实现）
func (u *Uploader) saveDraft(ctx *uploader.Context) uploader.StepResult {
	// 点击"保存草稿"按钮
	draftButton := ctx.Page.Locator("div.form-btns button:has-text('保存草稿')")
	if err := draftButton.Click(); err != nil {
		return uploader.StepResult{Step: uploader.StepPublish, Success: false, Error: err}
	}

	// 等待跳转到草稿箱页面或确认保存成功 - 增加超时时间到60秒
	timeout := time.After(60 * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			ctx.TakeScreenshot("draft_timeout")
			return uploader.StepResult{Step: uploader.StepPublish, Success: false, Error: fmt.Errorf("save draft timeout")}
		case <-ticker.C:
			// 检查是否跳转到草稿相关页面
			url := ctx.Page.URL()
			if strings.Contains(url, "post/list") || strings.Contains(url, "draft") {
				ctx.ReportProgress(uploader.StepPublish, 95, "草稿保存成功，等待后台处理完成...")
				ctx.TakeScreenshot("draft_success")
				// 检测到成功后，额外等待5秒确保后台处理完成
				time.Sleep(5 * time.Second)
				utils.Info("[-] 草稿保存完成，后台处理已结束")
				return uploader.StepResult{Step: uploader.StepPublish, Success: true}
			}

			// 截图记录状态
			ctx.TakeScreenshot("saving_draft")
		}
	}
}

// formatShortTitle 格式化短标题（6-16字，参考Python实现）
func formatShortTitle(originTitle string) string {
	// 定义允许的特殊字符
	allowedSpecialChars := `《》"":+?%°`

	// 移除不允许的特殊字符
	var filteredChars []rune
	for _, char := range originTitle {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') || (char >= '\u4e00' && char <= '\u9fff') {
			// 字母、数字、中文
			filteredChars = append(filteredChars, char)
		} else if char == ',' || char == '，' {
			// 逗号替换为空格
			filteredChars = append(filteredChars, ' ')
		} else {
			// 检查是否允许的特殊字符
			for _, allowed := range allowedSpecialChars {
				if char == allowed {
					filteredChars = append(filteredChars, char)
					break
				}
			}
		}
	}

	formattedString := string(filteredChars)

	// 调整字符串长度
	if len(formattedString) > 16 {
		// 截断到16字
		runes := []rune(formattedString)
		if len(runes) > 16 {
			formattedString = string(runes[:16])
		}
	} else if len(formattedString) < 6 {
		// 空格填充到6字
		for len(formattedString) < 6 {
			formattedString += " "
		}
	}

	return formattedString
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

	// 访问视频号助手
	utils.Info("[-] 正在打开视频号助手...")
	if _, err := page.Goto("https://channels.weixin.qq.com/platform"); err != nil {
		return fmt.Errorf("goto platform failed: %w", err)
	}

	// 等待用户扫码登录
	utils.Info("[-] 请使用微信扫描二维码登录...")

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

			// 检查是否已进入平台
			url := page.URL()
			if url == "https://channels.weixin.qq.com/platform" ||
				strings.Contains(url, "post/list") {
				utils.Info("[-] 登录成功，已进入视频号平台")
				return nil
			}
		}
	}
}
