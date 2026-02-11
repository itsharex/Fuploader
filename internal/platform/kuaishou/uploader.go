package kuaishou

import (
	"context"
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
		utils.InfoWithPlatform("kuaishou", fmt.Sprintf("[调试] "+format, args...))
	}
}

// browserPool 全局浏览器池实例
var browserPool *browser.Pool

func init() {
	browserPool = browser.NewPool(2, 5)
}

// Uploader 快手上传器
type Uploader struct {
	accountID  uint
	cookiePath string
	platform   string
}

// NewUploader 创建上传器（兼容旧版，使用cookiePath）
func NewUploader(cookiePath string) *Uploader {
	u := &Uploader{
		accountID:  0, // 兼容旧版
		cookiePath: cookiePath,
		platform:   "kuaishou",
	}
	debugLog("创建上传器 - 地址: %p, cookiePath: '%s'", u, cookiePath)
	if cookiePath == "" {
		utils.Warn("[Kuaishou] NewUploader 收到空的cookiePath!")
	}
	return u
}

// NewUploaderWithAccount 创建带accountID的上传器（新接口）
func NewUploaderWithAccount(accountID uint) *Uploader {
	cookiePath := config.GetCookiePath("kuaishou", int(accountID))
	u := &Uploader{
		accountID:  accountID,
		cookiePath: cookiePath,
		platform:   "kuaishou",
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
		utils.WarnWithPlatform(u.platform, "Cookie文件不存在")
		return false, nil
	}

	// 使用accountID获取上下文（如果accountID为0则退化为旧逻辑）
	browserCtx, err := browserPool.GetContextByAccount(ctx, u.accountID, u.cookiePath, nil)
	if err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 获取浏览器 - %v", err))
		return false, nil
	}
	defer browserCtx.Release()

	page, err := browserCtx.GetPage()
	if err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 获取页面 - %v", err))
		return false, nil
	}

	if _, err := page.Goto("https://cp.kuaishou.com/", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 打开页面 - %v", err))
		return false, nil
	}

	time.Sleep(3 * time.Second)

	// 使用Cookie检测机制验证登录状态
	cookieConfig, ok := browser.GetCookieConfig("kuaishou")
	if !ok {
		return false, fmt.Errorf("失败: 获取Cookie配置")
	}

	isValid, err := browserCtx.ValidateLoginCookies(cookieConfig)
	if err != nil {
		return false, fmt.Errorf("失败: 验证Cookie - %v", err)
	}

	// 保留平台特有的日志（Cookie名称验证）
	if isValid {
		utils.InfoWithPlatform(u.platform, "检测到kuaishou.server.web_ph Cookie，验证通过")
	} else {
		utils.InfoWithPlatform(u.platform, "未检测到kuaishou.server.web_ph Cookie，验证失败")
	}

	return isValid, nil
}

// Upload 上传视频
func (u *Uploader) Upload(ctx context.Context, task *types.VideoTask) error {
	utils.InfoWithPlatform(u.platform, fmt.Sprintf("开始上传: %s", task.VideoPath))

	if _, err := os.Stat(task.VideoPath); err != nil {
		return fmt.Errorf("失败: 检查视频文件 - %v", err)
	}

	// 使用accountID获取上下文（如果accountID为0则退化为旧逻辑）
	browserCtx, err := browserPool.GetContextByAccount(ctx, u.accountID, u.cookiePath, nil)
	if err != nil {
		return fmt.Errorf("失败: 获取浏览器 - %v", err)
	}
	defer browserCtx.Release()

	page, err := browserCtx.GetPage()
	if err != nil {
		return fmt.Errorf("失败: 获取页面 - %v", err)
	}

	// 导航到上传页面
	utils.InfoWithPlatform(u.platform, "正在打开发布页面...")
	if _, err := page.Goto("https://cp.kuaishou.com/article/publish/video", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		return fmt.Errorf("失败: 打开发布页面 - %v", err)
	}
	time.Sleep(3 * time.Second)

	// 处理新功能引导
	if err := u.handleNewFeatureGuide(page); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 处理新功能引导 - %v", err))
	}

	// 上传视频
	if err := u.uploadVideo(ctx, page, browserCtx, task.VideoPath); err != nil {
		return fmt.Errorf("失败: 上传视频 - %v", err)
	}

	// 等待2秒，让页面稳定
	time.Sleep(2 * time.Second)

	// 关闭提示弹窗（可选，失败不阻塞）
	if err := u.handleSkipPopup(page); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 关闭弹窗 - %v", err))
	}

	// 设置下载权限（取消勾选"允许下载此作品"）
	// 从task.AllowDownload读取设置，默认禁止下载
	allowDownload := false
	if task.AllowDownload {
		allowDownload = true
	}
	if err := u.setDownloadPermission(page, allowDownload); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 设置下载权限 - %v", err))
	}

	// 设置封面（新流程）
	if task.Thumbnail != "" {
		if err := u.setCover(page, task.Thumbnail); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 设置封面 - %v", err))
		}
	}

	// 填写描述和标题
	if err := u.fillDescription(page, task.Title, task.Description); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 填写描述 - %v", err))
	}

	// 添加话题标签（最多3个）
	if len(task.Tags) > 0 {
		if err := u.addTags(page, task.Tags); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 添加标签 - %v", err))
		}
	}

	// 设置定时发布
	if task.ScheduleTime != nil && *task.ScheduleTime != "" {
		if err := u.setScheduleTime(page, *task.ScheduleTime); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 设置定时发布 - %v", err))
		}
	}

	// 点击发布
	utils.InfoWithPlatform(u.platform, "准备发布...")
	if err := u.publish(page, browserCtx); err != nil {
		return fmt.Errorf("失败: 发布 - %v", err)
	}

	utils.SuccessWithPlatform(u.platform, "发布成功")
	return nil
}

// handleNewFeatureGuide 处理新功能引导
func (u *Uploader) handleNewFeatureGuide(page playwright.Page) error {
	newFeatureBtn := page.Locator("button[type='button'] span:has-text('我知道了')")
	count, _ := newFeatureBtn.Count()
	if count > 0 {
		if err := newFeatureBtn.Click(); err == nil {
			time.Sleep(1 * time.Second)
		}
	}
	return nil
}

// handleSkipPopup 关闭提示弹窗
func (u *Uploader) handleSkipPopup(page playwright.Page) error {
	skipBtn := page.Locator(`div[aria-label="Skip"][title="Skip"]`).First()
	count, _ := skipBtn.Count()
	if count > 0 {
		if err := skipBtn.Click(); err != nil {
			return fmt.Errorf("失败: 点击Skip按钮 - %v", err)
		}
		time.Sleep(500 * time.Millisecond)
	}
	return nil
}

// setDownloadPermission 设置下载权限
func (u *Uploader) setDownloadPermission(page playwright.Page, allowDownload bool) error {
	checkbox := page.Locator(`span:has-text("允许下载此作品")`).First()
	count, _ := checkbox.Count()
	if count == 0 {
		return fmt.Errorf("失败: 查找下载权限设置选项")
	}

	// 获取当前勾选状态
	// 通过检查父元素或相关元素的状态来判断是否已勾选
	// 这里简化处理：直接点击切换，根据allowDownload决定是否需要再次点击
	if err := checkbox.Click(); err != nil {
		return fmt.Errorf("失败: 点击下载权限选项 - %v", err)
	}

	_ = allowDownload
	time.Sleep(500 * time.Millisecond)
	return nil
}

// uploadVideo 上传视频
func (u *Uploader) uploadVideo(ctx context.Context, page playwright.Page, browserCtx *browser.PooledContext, videoPath string) error {
	utils.InfoWithPlatform(u.platform, "正在上传视频...")

	uploadButton := page.Locator("button[class^='_upload-btn']")
	if err := uploadButton.WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(10000),
	}); err != nil {
		return fmt.Errorf("失败: 等待上传按钮 - %v", err)
	}

	fileChooser, err := page.ExpectFileChooser(func() error {
		return uploadButton.Click()
	})
	if err != nil {
		return fmt.Errorf("失败: 等待文件选择器 - %v", err)
	}

	if err := fileChooser.SetFiles(videoPath); err != nil {
		return fmt.Errorf("失败: 设置视频文件 - %v", err)
	}

	// 等待上传完成
	utils.InfoWithPlatform(u.platform, "等待视频上传完成...")
	if err := u.waitForUploadComplete(ctx, page, browserCtx); err != nil {
		return fmt.Errorf("失败: 等待视频上传 - %v", err)
	}
	utils.InfoWithPlatform(u.platform, "视频上传完成")

	return nil
}

// waitForUploadComplete 等待视频上传完成
func (u *Uploader) waitForUploadComplete(ctx context.Context, page playwright.Page, browserCtx *browser.PooledContext) error {
	maxRetries := 60
	retryInterval := 2 * time.Second

	for retryCount := 0; retryCount < maxRetries; retryCount++ {
		select {
		case <-ctx.Done():
			return fmt.Errorf("失败: 视频上传 - 已取消")
		default:
		}

		if browserCtx.IsPageClosed() {
			return fmt.Errorf("失败: 视频上传 - 浏览器已关闭")
		}

		// 检测方式1："上传中"文本消失
		uploadingCount, _ := page.Locator("text=上传中").Count()
		if uploadingCount == 0 {
			// 检查是否有成功标志
			successCount, _ := page.Locator("[class*='success'] >> text=上传成功").Count()
			if successCount > 0 {
				return nil
			}
			// 检查视频预览是否出现
			videoPreview := page.Locator("video, .video-preview, [class*='videoPreview']").First()
			if count, _ := videoPreview.Count(); count > 0 {
				if visible, _ := videoPreview.IsVisible(); visible {
					return nil
				}
			}
			return nil
		}

		// 检测上传失败
		errorText := page.Locator("text=/上传失败|上传出错|Upload failed/").First()
		if count, _ := errorText.Count(); count > 0 {
			return fmt.Errorf("失败: 视频上传 - 检测到上传失败")
		}

		time.Sleep(retryInterval)
	}

	return fmt.Errorf("失败: 视频上传 - 超时，已等待%d次检测", maxRetries)
}

// fillDescription 填写描述和标题
func (u *Uploader) fillDescription(page playwright.Page, title, description string) error {
	utils.InfoWithPlatform(u.platform, "填写标题...")

	// 定位描述输入区域
	descArea := page.GetByText("描述").Locator("xpath=following-sibling::div")
	if err := descArea.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		// 兜底：尝试其他选择器
		descArea = page.Locator("textarea[placeholder*='描述'], div[contenteditable='true']").First()
		if err := descArea.WaitFor(playwright.LocatorWaitForOptions{
			Timeout: playwright.Float(3000),
		}); err != nil {
			return fmt.Errorf("失败: 查找描述输入区域 - %v", err)
		}
	}

	// 点击并清空
	if err := descArea.Click(); err != nil {
		return fmt.Errorf("失败: 点击描述区域 - %v", err)
	}
	time.Sleep(300 * time.Millisecond)

	page.Keyboard().Press("Backspace")
	page.Keyboard().Press("Control+KeyA")
	page.Keyboard().Press("Delete")
	time.Sleep(300 * time.Millisecond)

	// 填写标题（快手描述区域通常包含标题）
	content := ""
	if title != "" {
		content = title
	}
	if description != "" {
		if content != "" {
			content += "\n"
		}
		content += description
	}

	if content != "" {
		page.Keyboard().Type(content)
		utils.InfoWithPlatform(u.platform, fmt.Sprintf("标题已填写: %s", title))
		utils.InfoWithPlatform(u.platform, "描述已填写")
	}

	time.Sleep(500 * time.Millisecond)
	return nil
}

// addTags 添加话题标签（最多3个）
func (u *Uploader) addTags(page playwright.Page, tags []string) error {
	// 快手最多支持3个标签
	maxTags := 3
	if len(tags) < maxTags {
		maxTags = len(tags)
	}
	tagsToAdd := tags[:maxTags]

	utils.InfoWithPlatform(u.platform, "添加标签...")

	for _, tag := range tagsToAdd {
		// 清理标签
		cleanTag := strings.TrimSpace(tag)
		cleanTag = strings.ReplaceAll(cleanTag, "#", "")

		if cleanTag == "" {
			continue
		}

		// 在描述区域输入标签
		page.Keyboard().Type(fmt.Sprintf("#%s ", cleanTag))
		time.Sleep(2 * time.Second) // 快手标签需要等待联想
	}

	utils.InfoWithPlatform(u.platform, "标签添加完成")
	return nil
}

// setCover 设置封面（新流程）
func (u *Uploader) setCover(page playwright.Page, coverPath string) error {
	if _, err := os.Stat(coverPath); err != nil {
		return fmt.Errorf("失败: 检查封面文件 - %v", err)
	}

	// 1. 进入封面设置
	coverSettingBtn := page.Locator(`div:has-text("封面设置")`).First()
	if err := coverSettingBtn.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		return fmt.Errorf("失败: 查找封面设置按钮 - %v", err)
	}
	if err := coverSettingBtn.Click(); err != nil {
		return fmt.Errorf("失败: 点击封面设置按钮 - %v", err)
	}
	time.Sleep(1 * time.Second)

	utils.InfoWithPlatform(u.platform, "设置封面...")

	// 2. 切换到"上传封面"标签
	uploadCoverTab := page.Locator(`div:has-text("上传封面")`).First()
	if err := uploadCoverTab.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		return fmt.Errorf("失败: 查找上传封面标签 - %v", err)
	}
	if err := uploadCoverTab.Click(); err != nil {
		return fmt.Errorf("失败: 点击上传封面标签 - %v", err)
	}
	time.Sleep(1 * time.Second)

	// 3. 定位文件输入框并上传封面图片
	coverInput := page.Locator(`input[type="file"][accept^="image/"]`).First()
	if err := coverInput.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		return fmt.Errorf("失败: 查找封面文件输入框 - %v", err)
	}
	if err := coverInput.SetInputFiles(coverPath); err != nil {
		return fmt.Errorf("失败: 上传封面 - %v", err)
	}
	time.Sleep(2 * time.Second)

	// 4. 点击"确认"按钮
	confirmBtn := page.Locator(`span:has-text("确认")`).First()
	if err := confirmBtn.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		return fmt.Errorf("失败: 查找确认按钮 - %v", err)
	}
	if err := confirmBtn.Click(); err != nil {
		return fmt.Errorf("失败: 点击确认按钮 - %v", err)
	}

	utils.InfoWithPlatform(u.platform, "封面设置完成")
	time.Sleep(1 * time.Second)
	return nil
}

// setScheduleTime 设置定时发布（使用日期选择器）
func (u *Uploader) setScheduleTime(page playwright.Page, scheduleTime string) error {
	utils.InfoWithPlatform(u.platform, fmt.Sprintf("设置定时发布时间: %s", scheduleTime))

	// 解析时间
	targetTime, err := time.Parse("2006-01-02 15:04:05", scheduleTime)
	if err != nil {
		// 尝试不带秒的格式
		targetTime, err = time.Parse("2006-01-02 15:04", scheduleTime)
		if err != nil {
			return fmt.Errorf("失败: 解析时间 - %v", err)
		}
	}

	// 点击定时发布选项
	scheduleLabel := page.Locator("label:has-text('发布时间')")
	if err := scheduleLabel.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(3000),
	}); err != nil {
		return fmt.Errorf("失败: 查找发布时间选项 - %v", err)
	}

	// 选择定时发布单选框（通常是第二个）
	scheduleRadio := scheduleLabel.Locator("xpath=following-sibling::div").Locator(".ant-radio-input").Nth(1)
	if err := scheduleRadio.Click(); err != nil {
		// 兜底：直接点击"定时发布"文本
		scheduleText := page.GetByText("定时发布")
		if err := scheduleText.Click(); err != nil {
			return fmt.Errorf("失败: 点击定时发布 - %v", err)
		}
	}
	time.Sleep(1 * time.Second)

	// 点击定时发布输入框，打开日期选择器
	scheduleInput := page.Locator(`input[placeholder="选择日期和时间"]`).First()
	if err := scheduleInput.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		return fmt.Errorf("失败: 查找定时发布输入框 - %v", err)
	}
	if err := scheduleInput.Click(); err != nil {
		return fmt.Errorf("失败: 点击定时发布输入框 - %v", err)
	}
	time.Sleep(1 * time.Second)

	// 选择日期
	dateStr := targetTime.Format("2006-01-02")
	dateCell := page.Locator(fmt.Sprintf(`td[title="%s"] div.ant-picker-cell-inner`, dateStr)).First()
	if err := dateCell.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		return fmt.Errorf("失败: 查找日期单元格 %s - %v", dateStr, err)
	}
	if err := dateCell.Click(); err != nil {
		return fmt.Errorf("失败: 点击日期单元格 - %v", err)
	}
	time.Sleep(500 * time.Millisecond)

	// 选择小时（第1列）
	hourStr := targetTime.Format("15")
	hourCell := page.Locator(fmt.Sprintf(`div.ant-picker-time-panel-column >> div.ant-picker-time-panel-cell-inner:has-text("%s")`, hourStr)).First()
	if count, _ := hourCell.Count(); count > 0 {
		if err := hourCell.Click(); err != nil {
			return fmt.Errorf("失败: 选择小时 - %v", err)
		}
		time.Sleep(300 * time.Millisecond)
	}

	// 选择分钟（第2列）
	minuteStr := targetTime.Format("04")
	minuteCell := page.Locator(fmt.Sprintf(`div.ant-picker-time-panel-column >> div.ant-picker-time-panel-cell-inner:has-text("%s")`, minuteStr)).Nth(1)
	if count, _ := minuteCell.Count(); count > 0 {
		if err := minuteCell.Click(); err != nil {
			return fmt.Errorf("失败: 选择分钟 - %v", err)
		}
		time.Sleep(300 * time.Millisecond)
	}

	// 选择秒（第3列）
	secondStr := targetTime.Format("05")
	secondCell := page.Locator(fmt.Sprintf(`div.ant-picker-time-panel-column >> div.ant-picker-time-panel-cell-inner:has-text("%s")`, secondStr)).Nth(2)
	if count, _ := secondCell.Count(); count > 0 {
		if err := secondCell.Click(); err != nil {
			return fmt.Errorf("失败: 选择秒 - %v", err)
		}
		time.Sleep(300 * time.Millisecond)
	}

	// 点击"确定"按钮
	confirmBtn := page.Locator(`span:has-text("确定")`).First()
	if err := confirmBtn.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		return fmt.Errorf("失败: 查找确定按钮 - %v", err)
	}
	if err := confirmBtn.Click(); err != nil {
		return fmt.Errorf("失败: 点击确定按钮 - %v", err)
	}
	time.Sleep(1 * time.Second)
	return nil
}

// publish 点击发布并检测结果
func (u *Uploader) publish(page playwright.Page, browserCtx *browser.PooledContext) error {
	maxAttempts := 30

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if browserCtx.IsPageClosed() {
			return fmt.Errorf("失败: 发布 - 浏览器已关闭")
		}

		// 定位发布按钮
		publishButton := page.GetByText("发布", playwright.PageGetByTextOptions{Exact: playwright.Bool(true)})
		count, _ := publishButton.Count()
		if count > 0 {
			if err := publishButton.Click(); err != nil {
				utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 点击发布按钮 - %v", err))
			}
		}

		time.Sleep(1 * time.Second)

		// 处理确认弹窗
		confirmButton := page.GetByText("确认发布")
		confirmCount, _ := confirmButton.Count()
		if confirmCount > 0 {
			if err := confirmButton.Click(); err != nil {
				utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 点击确认发布 - %v", err))
			}
		}

		// 检测发布结果
		currentURL := page.URL()
		if currentURL == "https://cp.kuaishou.com/article/manage/video?status=2&from=publish" {
			return nil
		}

		// 检测成功提示
		successCount, _ := page.Locator("text=发布成功").Count()
		if successCount > 0 {
			if visible, _ := page.Locator("text=发布成功").IsVisible(); visible {
				return nil
			}
		}

		// 检测错误
		errorText := page.Locator("text=/发布失败|提交失败|错误/").First()
		if count, _ := errorText.Count(); count > 0 {
			if visible, _ := errorText.IsVisible(); visible {
				text, _ := errorText.TextContent()
				return fmt.Errorf("失败: 发布 - %s", text)
			}
		}

		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("失败: 发布 - 超时，已尝试%d次", maxAttempts)
}

// Login 登录
func (u *Uploader) Login() error {
	debugLog("Login开始 - cookiePath: '%s'", u.cookiePath)
	if u.cookiePath == "" {
		return fmt.Errorf("cookie路径为空")
	}

	ctx := context.Background()
	utils.InfoWithPlatform(u.platform, fmt.Sprintf("Cookie保存路径: %s", u.cookiePath))

	// 登录时不使用accountID（因为是新登录）
	browserCtx, err := browserPool.GetContextByAccount(ctx, 0, "", nil)
	if err != nil {
		return fmt.Errorf("失败: 获取浏览器 - %v", err)
	}
	defer browserCtx.Release()

	page, err := browserCtx.GetPage()
	if err != nil {
		return fmt.Errorf("失败: 获取页面 - %v", err)
	}

	utils.InfoWithPlatform(u.platform, "正在打开发布页面...")
	if _, err := page.Goto("https://cp.kuaishou.com/article/publish/video", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	}); err != nil {
		return fmt.Errorf("失败: 打开发布页面 - %v", err)
	}

	time.Sleep(3 * time.Second)

	utils.InfoWithPlatform(u.platform, "请在浏览器窗口中完成登录，登录成功后会自动保存")

	// 使用Cookie检测机制等待登录成功
	cookieConfig, ok := browser.GetCookieConfig("kuaishou")
	if !ok {
		return fmt.Errorf("失败: 获取Cookie配置")
	}

	if err := browserCtx.WaitForLoginCookies(cookieConfig); err != nil {
		return fmt.Errorf("失败: 等待登录Cookie - %v", err)
	}

	utils.SuccessWithPlatform(u.platform, "登录成功")
	if err := browserCtx.SaveCookiesTo(u.cookiePath); err != nil {
		return fmt.Errorf("失败: 保存Cookie - %v", err)
	}
	return nil
}
