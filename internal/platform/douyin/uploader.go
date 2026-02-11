package douyin

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
		utils.InfoWithPlatform("douyin", fmt.Sprintf("[调试] "+format, args...))
	}
}

// browserPool 全局浏览器池实例
var browserPool *browser.Pool

func init() {
	browserPool = browser.NewPool(2, 5)
}

// Uploader 抖音上传器
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
		platform:   "douyin",
	}
	debugLog("创建上传器 - 地址: %p, cookiePath: '%s'", u, cookiePath)
	if cookiePath == "" {
		utils.WarnWithPlatform(u.platform, "失败: 创建上传器 - cookie路径为空")
	}
	return u
}

// NewUploaderWithAccount 创建带accountID的上传器（新接口）
func NewUploaderWithAccount(accountID uint) *Uploader {
	cookiePath := config.GetCookiePath("douyin", int(accountID))
	u := &Uploader{
		accountID:  accountID,
		cookiePath: cookiePath,
		platform:   "douyin",
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

	// 使用accountID获取上下文
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
	if _, err := page.Goto("https://creator.douyin.com/", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 验证Cookie - 打开页面失败: %v", err))
		return false, nil
	}

	time.Sleep(3 * time.Second)

	// 使用Cookie检测机制验证登录状态
	cookieConfig, ok := browser.GetCookieConfig("douyin")
	if !ok {
		return false, fmt.Errorf("失败: 验证Cookie - 获取抖音Cookie配置失败")
	}

	isValid, err := browserCtx.ValidateLoginCookies(cookieConfig)
	if err != nil {
		return false, fmt.Errorf("失败: 验证Cookie - 验证失败: %w", err)
	}

	if isValid {
		utils.SuccessWithPlatform(u.platform, "验证Cookie成功")
	} else {
		utils.WarnWithPlatform(u.platform, "失败: 验证Cookie - 未检测到sessionid Cookie")
	}

	return isValid, nil
}

// Upload 上传视频
func (u *Uploader) Upload(ctx context.Context, task *types.VideoTask) error {
	utils.InfoWithPlatform(u.platform, fmt.Sprintf("开始上传: %s", task.VideoPath))

	if _, err := os.Stat(task.VideoPath); err != nil {
		return fmt.Errorf("视频文件不存在: %w", err)
	}

	browserCtx, err := browserPool.GetContext(ctx, u.cookiePath, nil)
	if err != nil {
		return fmt.Errorf("获取浏览器失败: %w", err)
	}
	defer browserCtx.Release()

	page, err := browserCtx.GetPage()
	if err != nil {
		return fmt.Errorf("获取页面失败: %w", err)
	}

	// 导航到上传页面
	utils.InfoWithPlatform(u.platform, "正在打开发布页面...")
	if _, err := page.Goto("https://creator.douyin.com/creator-micro/content/upload", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		return fmt.Errorf("失败: 打开发布页面 - %w", err)
	}
	time.Sleep(3 * time.Second)

	// 上传视频
	utils.InfoWithPlatform(u.platform, "正在上传视频...")
	if err := u.uploadVideo(ctx, page, browserCtx, task.VideoPath); err != nil {
		return fmt.Errorf("失败: 上传视频 - %w", err)
	}

	time.Sleep(2 * time.Second)

	// 填写标题（限制30字符）
	if task.Title != "" {
		if err := u.fillTitle(page, task.Title); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 填写标题 - %v", err))
		}
	}

	// 填写描述和标签（抖音共用同一个输入框，先填描述，再填标签）
	if task.Description != "" || len(task.Tags) > 0 {
		// 先填描述
		if task.Description != "" {
			if err := u.fillDescription(page, task.Description); err != nil {
				utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 填写描述 - %v", err))
			}
		}
		// 再填标签
		if len(task.Tags) > 0 {
			if err := u.addTags(page, task.Tags); err != nil {
				utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 添加标签 - %v", err))
			}
		}
	}

	// 设置封面（始终执行，有自定义封面时上传，无则使用默认）
	if err := u.setCover(page, task.Thumbnail); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 设置封面 - %v", err))
	}

	// 添加商品链接
	if task.ProductLink != "" {
		if err := u.addProductLink(page, task.ProductLink, task.ProductTitle); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 添加商品链接 - %v", err))
		}
	}

	// 设置同步选项
	if task.SyncToutiao || task.SyncXigua {
		if err := u.setSyncOptions(page, task.SyncToutiao, task.SyncXigua); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 设置同步选项 - %v", err))
		}
	}

	// 设置权限选项
	if task.AllowDownload || task.AllowComment {
		if err := u.setPermissions(page, task.AllowDownload, task.AllowComment); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 设置权限选项 - %v", err))
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
		return fmt.Errorf("失败: 发布 - %w", err)
	}

	utils.SuccessWithPlatform(u.platform, "发布成功")
	return nil
}

// uploadVideo 上传视频
func (u *Uploader) uploadVideo(ctx context.Context, page playwright.Page, browserCtx *browser.PooledContext, videoPath string) error {
	// 定位文件输入框
	inputLocator := page.Locator("div[class^='container'] input[type='file']").First()
	if err := inputLocator.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(10000),
	}); err != nil {
		// 兜底：尝试通用选择器
		inputLocator = page.Locator("input[type='file']").First()
		if err := inputLocator.WaitFor(playwright.LocatorWaitForOptions{
			Timeout: playwright.Float(5000),
		}); err != nil {
			return fmt.Errorf("失败: 上传视频 - 未找到文件输入框: %w", err)
		}
	}

	if err := inputLocator.SetInputFiles(videoPath); err != nil {
		return fmt.Errorf("失败: 上传视频 - 设置视频文件失败: %w", err)
	}

	// 等待上传完成
	utils.InfoWithPlatform(u.platform, "等待视频上传完成...")
	if err := u.waitForUploadComplete(ctx, page, browserCtx); err != nil {
		return err
	}

	return nil
}

// waitForUploadComplete 等待视频上传完成
func (u *Uploader) waitForUploadComplete(ctx context.Context, page playwright.Page, browserCtx *browser.PooledContext) error {
	uploadTimeout := 10 * time.Minute
	uploadCheckInterval := 2 * time.Second
	uploadStartTime := time.Now()

	for time.Since(uploadStartTime) < uploadTimeout {
		select {
		case <-ctx.Done():
			return fmt.Errorf("上传已取消")
		default:
		}

		if browserCtx.IsPageClosed() {
			return fmt.Errorf("浏览器已关闭")
		}

		// 检测方式1：视频预览区域出现
		videoPreview := page.Locator("video, .video-preview, [class*='videoPreview'], div[class*='player']").First()
		if count, _ := videoPreview.Count(); count > 0 {
			if visible, _ := videoPreview.IsVisible(); visible {
				utils.InfoWithPlatform(u.platform, "视频上传完成")
				return nil
			}
		}

		// 检测方式2：上传进度条消失
		progressBar := page.Locator("div[class*='progress'], div[class*='uploading']").First()
		if count, _ := progressBar.Count(); count == 0 {
			// 检查是否有视频信息
			videoInfo := page.Locator("div[class*='video-info'], div[class*='mediaInfo']").First()
			if count, _ := videoInfo.Count(); count > 0 {
				utils.InfoWithPlatform(u.platform, "视频上传完成")
				return nil
			}
		}

		// 检测方式3："上传成功"文本
		successText := page.Locator("text=/上传成功|上传完成/").First()
		if count, _ := successText.Count(); count > 0 {
			if visible, _ := successText.IsVisible(); visible {
				utils.InfoWithPlatform(u.platform, "视频上传完成")
				return nil
			}
		}

		// 检测上传失败
		errorText := page.Locator("text=/上传失败|上传出错/").First()
		if count, _ := errorText.Count(); count > 0 {
			return fmt.Errorf("失败: 上传视频 - 检测到上传失败")
		}

		time.Sleep(uploadCheckInterval)
	}

	return fmt.Errorf("失败: 上传视频 - 上传超时")
}

// fillTitle 填写标题（限制30字符）
func (u *Uploader) fillTitle(page playwright.Page, title string) error {
	if title == "" {
		return nil
	}

	utils.InfoWithPlatform(u.platform, "填写标题...")

	// 限制30字符
	if len(title) > 30 {
		runes := []rune(title)
		if len(runes) > 30 {
			title = string(runes[:30])
		}
	}

	// 使用placeholder定位标题输入框
	titleInput := page.Locator(`input[placeholder="填写作品标题，为作品获得更多流量"]`).First()
	if err := titleInput.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		return fmt.Errorf("未找到标题输入框: %w", err)
	}

	if err := titleInput.Fill(title); err != nil {
		return fmt.Errorf("填写标题失败: %w", err)
	}

	utils.InfoWithPlatform(u.platform, fmt.Sprintf("标题已填写: %s", title))
	time.Sleep(500 * time.Millisecond)
	return nil
}

// fillDescription 填写描述（抖音描述和标签共用同一个输入框）
func (u *Uploader) fillDescription(page playwright.Page, description string) error {
	utils.InfoWithPlatform(u.platform, "填写描述...")

	// 定位描述输入框（与标签共用.zone-container）
	descContainer := page.Locator(".zone-container").First()
	if err := descContainer.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		return fmt.Errorf("未找到描述输入框: %w", err)
	}

	if err := descContainer.Fill(description); err != nil {
		return fmt.Errorf("填写描述失败: %w", err)
	}

	utils.InfoWithPlatform(u.platform, "描述已填写")
	time.Sleep(500 * time.Millisecond)
	return nil
}

// addTags 添加话题标签
func (u *Uploader) addTags(page playwright.Page, tags []string) error {
	utils.InfoWithPlatform(u.platform, fmt.Sprintf("添加%d个标签...", len(tags)))

	// 定位标签输入区域
	tagContainer := page.Locator(".zone-container").First()
	if err := tagContainer.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(3000),
	}); err != nil {
		// 兜底：尝试其他选择器
		tagContainer = page.Locator("div[class*='tag'], div[class*='topic']").First()
		if err := tagContainer.WaitFor(playwright.LocatorWaitForOptions{
			Timeout: playwright.Float(3000),
		}); err != nil {
			return fmt.Errorf("未找到标签输入区域: %w", err)
		}
	}

	count, _ := tagContainer.Count()
	if count > 0 {
		for _, tag := range tags {
			cleanTag := strings.TrimSpace(tag)
			cleanTag = strings.ReplaceAll(cleanTag, "#", "")
			if cleanTag == "" {
				continue
			}

			tagContainer.Type("#"+cleanTag, playwright.LocatorTypeOptions{Delay: playwright.Float(100)})
			tagContainer.Press("Space")
			time.Sleep(300 * time.Millisecond)
		}
	}

	utils.InfoWithPlatform(u.platform, "标签添加完成")
	return nil
}

// setCover 设置封面
func (u *Uploader) setCover(page playwright.Page, coverPath string) error {
	utils.InfoWithPlatform(u.platform, "设置封面...")

	// 点击封面设置按钮
	coverBtn := page.GetByText("选择封面").First()
	if err := coverBtn.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		return fmt.Errorf("未找到封面设置按钮: %w", err)
	}

	if err := coverBtn.Click(); err != nil {
		return fmt.Errorf("点击封面设置按钮失败: %w", err)
	}
	time.Sleep(2 * time.Second)

	// 上传自定义封面
	if coverPath != "" {
		if _, err := os.Stat(coverPath); err == nil {
			utils.InfoWithPlatform(u.platform, "上传自定义封面...")

			// 直接定位封面上传的隐藏输入框
			coverInput := page.Locator(`input[type="file"][accept^="image/"].semi-upload-hidden-input`).First()
			if err := coverInput.WaitFor(playwright.LocatorWaitForOptions{
				Timeout: playwright.Float(5000),
			}); err != nil {
				utils.WarnWithPlatform(u.platform, fmt.Sprintf("未找到封面上传输入框: %v", err))
			} else {
				if err := coverInput.SetInputFiles(coverPath); err != nil {
					utils.WarnWithPlatform(u.platform, fmt.Sprintf("上传封面失败: %v", err))
				} else {
					utils.InfoWithPlatform(u.platform, "封面上传中...")
					time.Sleep(3 * time.Second)
				}
			}
		}
	}

	// 点击"设置竖封面"按钮切换到竖封面界面
	verticalBtn := page.Locator(`button:has-text("设置竖封面")`).First()
	if err := verticalBtn.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		utils.WarnWithPlatform(u.platform, "未找到设置竖封面按钮")
	} else {
		if err := verticalBtn.Click(); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("点击设置竖封面按钮失败: %v", err))
		} else {
			utils.InfoWithPlatform(u.platform, "已切换到竖封面")
		}
		time.Sleep(2 * time.Second)
	}

	// 点击"完成"按钮（使用span选择器）
	finishBtn := page.Locator(`span:has-text("完成")`).First()
	if err := finishBtn.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("未找到完成按钮: %v", err))
	} else {
		if err := finishBtn.Click(); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("点击完成按钮失败: %v", err))
		} else {
			utils.InfoWithPlatform(u.platform, "已点击完成按钮")
		}
	}
	time.Sleep(2 * time.Second)

	utils.InfoWithPlatform(u.platform, "封面设置完成")
	return nil
}

// addProductLink 添加商品链接
func (u *Uploader) addProductLink(page playwright.Page, productLink, productTitle string) error {
	utils.InfoWithPlatform(u.platform, "添加商品链接...")

	// 点击"添加标签"
	addTagBtn := page.GetByText("添加标签").First()
	if err := addTagBtn.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(3000),
	}); err != nil {
		return fmt.Errorf("未找到添加标签按钮: %w", err)
	}

	if err := addTagBtn.Click(); err != nil {
		return fmt.Errorf("点击添加标签失败: %w", err)
	}
	time.Sleep(1 * time.Second)

	// 点击"购物车"
	cartBtn := page.GetByText("购物车").First()
	if count, _ := cartBtn.Count(); count > 0 {
		cartBtn.Click()
		time.Sleep(1 * time.Second)
	}

	// 填写商品链接
	linkInput := page.Locator("input[placeholder='添加商品链接']").First()
	if err := linkInput.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(3000),
	}); err != nil {
		return fmt.Errorf("未找到商品链接输入框: %w", err)
	}

	if err := linkInput.Fill(productLink); err != nil {
		return fmt.Errorf("填写商品链接失败: %w", err)
	}

	// 填写商品短标题（限制10字符）
	if productTitle != "" {
		shortTitle := productTitle
		if len(shortTitle) > 10 {
			runes := []rune(shortTitle)
			if len(runes) > 10 {
				shortTitle = string(runes[:10])
			}
		}

		titleInput := page.Locator("input[placeholder*='短标题']").First()
		if count, _ := titleInput.Count(); count > 0 {
			titleInput.Fill(shortTitle)
		}
	}

	utils.InfoWithPlatform(u.platform, "商品链接已添加")
	time.Sleep(1 * time.Second)
	return nil
}

// setSyncOptions 设置同步选项
func (u *Uploader) setSyncOptions(page playwright.Page, syncToutiao, syncXigua bool) error {
	// 同步到今日头条
	if syncToutiao {
		toutiaoCheckbox := page.Locator("text=同步到今日头条").Locator("xpath=../input[type='checkbox']").First()
		if count, _ := toutiaoCheckbox.Count(); count > 0 {
			isChecked, _ := toutiaoCheckbox.IsChecked()
			if !isChecked {
				toutiaoCheckbox.Check()
			}
		}
	}

	// 同步到西瓜视频
	if syncXigua {
		xiguaCheckbox := page.Locator("text=同步到西瓜视频").Locator("xpath=../input[type='checkbox']").First()
		if count, _ := xiguaCheckbox.Count(); count > 0 {
			isChecked, _ := xiguaCheckbox.IsChecked()
			if !isChecked {
				xiguaCheckbox.Check()
			}
		}
	}

	time.Sleep(500 * time.Millisecond)
	return nil
}

// setPermissions 设置权限选项
func (u *Uploader) setPermissions(page playwright.Page, allowDownload, allowComment bool) error {
	// 设置是否允许下载
	if !allowDownload {
		// 选择"不允许"
		disallowBtn := page.Locator(`span:has-text("不允许")`).First()
		if err := disallowBtn.WaitFor(playwright.LocatorWaitForOptions{
			Timeout: playwright.Float(5000),
		}); err != nil {
			return fmt.Errorf("未找到不允许选项: %w", err)
		}
		if err := disallowBtn.Click(); err != nil {
			return fmt.Errorf("点击不允许失败: %w", err)
		}
	} else {
		// 选择"允许"
		allowBtn := page.Locator(`span:has-text("允许")`).First()
		if err := allowBtn.WaitFor(playwright.LocatorWaitForOptions{
			Timeout: playwright.Float(5000),
		}); err != nil {
			return fmt.Errorf("未找到允许选项: %w", err)
		}
		if err := allowBtn.Click(); err != nil {
			return fmt.Errorf("点击允许失败: %w", err)
		}
	}

	time.Sleep(500 * time.Millisecond)
	return nil
}

// setScheduleTime 设置定时发布
func (u *Uploader) setScheduleTime(page playwright.Page, scheduleTime string) error {
	// 点击"定时发布"选项
	scheduleBtn := page.Locator(`span:has-text("定时发布")`).First()
	if err := scheduleBtn.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		return fmt.Errorf("未找到定时发布选项: %w", err)
	}
	if err := scheduleBtn.Click(); err != nil {
		return fmt.Errorf("点击定时发布失败: %w", err)
	}
	time.Sleep(1 * time.Second)

	// 填写定时发布时间
	timeInput := page.Locator(`input[format="yyyy-MM-dd HH:mm"]`).First()
	if err := timeInput.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		return fmt.Errorf("未找到时间输入框: %w", err)
	}
	if err := timeInput.Fill(scheduleTime); err != nil {
		return fmt.Errorf("填写定时发布时间失败: %w", err)
	}

	time.Sleep(1 * time.Second)
	return nil
}

// publish 点击发布并检测结果
func (u *Uploader) publish(page playwright.Page, browserCtx *browser.PooledContext) error {
	maxRetries := 20

	for retryCount := 0; retryCount < maxRetries; retryCount++ {
		if browserCtx.IsPageClosed() {
			return fmt.Errorf("失败: 发布 - 浏览器已关闭")
		}

		// 定位发布按钮
		publishBtn := page.GetByRole("button", playwright.PageGetByRoleOptions{
			Name:  "发布",
			Exact: playwright.Bool(true),
		})
		if count, _ := publishBtn.Count(); count > 0 {
			if err := publishBtn.Click(); err != nil {
				utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 发布 - 点击发布按钮失败: %v", err))
			}
		}

		time.Sleep(5 * time.Second)

		// 检测发布结果
		url := page.URL()
		if strings.Contains(url, "creator.douyin.com/creator-micro/content/manage") {
			return nil
		}

		// 处理封面未设置提示
		coverPrompt := page.GetByText("请设置封面后再发布").First()
		if visible, _ := coverPrompt.IsVisible(); visible {
			recommendCover := page.Locator("[class^='recommendCover-']").First()
			if count, _ := recommendCover.Count(); count > 0 {
				recommendCover.Click()
				time.Sleep(1 * time.Second)
				confirmBtn := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "确定"})
				if count, _ := confirmBtn.Count(); count > 0 {
					confirmBtn.Click()
					time.Sleep(1 * time.Second)
				}
			}
		}

		// 检测成功提示
		successText := page.Locator("text=/发布成功|提交成功/").First()
		if count, _ := successText.Count(); count > 0 {
			if visible, _ := successText.IsVisible(); visible {
				return nil
			}
		}
	}

	return fmt.Errorf("失败: 发布 - 发布超时，已重试%d次", maxRetries)
}

// Login 登录
func (u *Uploader) Login() error {
	debugLog("Login开始 - cookiePath: '%s'", u.cookiePath)
	if u.cookiePath == "" {
		return fmt.Errorf("失败: 登录 - cookie路径为空")
	}

	ctx := context.Background()

	browserCtx, err := browserPool.GetContext(ctx, "", nil)
	if err != nil {
		return fmt.Errorf("失败: 登录 - 获取浏览器失败: %w", err)
	}
	defer browserCtx.Release()

	page, err := browserCtx.GetPage()
	if err != nil {
		return fmt.Errorf("失败: 登录 - 获取页面失败: %w", err)
	}

	utils.InfoWithPlatform(u.platform, "正在打开发布页面...")
	if _, err := page.Goto("https://creator.douyin.com/", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	}); err != nil {
		return fmt.Errorf("失败: 登录 - 打开登录页面失败: %w", err)
	}

	time.Sleep(3 * time.Second)

	// 使用Cookie检测机制等待登录成功
	cookieConfig, ok := browser.GetCookieConfig("douyin")
	if !ok {
		return fmt.Errorf("失败: 登录 - 获取抖音Cookie配置失败")
	}

	if err := browserCtx.WaitForLoginCookies(cookieConfig); err != nil {
		return fmt.Errorf("失败: 登录 - 等待登录Cookie失败: %w", err)
	}

	utils.SuccessWithPlatform(u.platform, "登录成功")
	if err := browserCtx.SaveCookiesTo(u.cookiePath); err != nil {
		return fmt.Errorf("失败: 登录 - 保存Cookie失败: %w", err)
	}
	return nil
}
