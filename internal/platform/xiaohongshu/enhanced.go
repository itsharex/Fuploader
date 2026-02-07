package xiaohongshu

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"Fuploader/internal/platform/browser"
	"Fuploader/internal/platform/uploader"
	"Fuploader/internal/utils"

	"github.com/playwright-community/playwright-go"
)

// EnhancedContextOptions 增强的浏览器上下文选项（带指纹随机化）
func (u *Uploader) getEnhancedContextOptions() *browser.ContextOptions {
	// 随机化 User-Agent 版本
	chromeVersions := []string{"120", "121", "122", "123", "124"}
	version := chromeVersions[rand.Intn(len(chromeVersions))]

	// 随机化视口大小（在合理范围内）
	width := 1920 + rand.Intn(100) - 50
	height := 1080 + rand.Intn(100) - 50

	return &browser.ContextOptions{
		UserAgent: fmt.Sprintf(
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s.0.0.0 Safari/537.36",
			version,
		),
		Viewport: &playwright.Size{
			Width:  width,
			Height: height,
		},
		Locale:     "zh-CN",
		TimezoneId: "Asia/Shanghai",
		Geolocation: &playwright.Geolocation{
			Latitude:  39.9042 + (rand.Float64()-0.5)*0.1,
			Longitude: 116.4074 + (rand.Float64()-0.5)*0.1,
		},
		ExtraHeaders: map[string]string{
			"Accept-Language":           "zh-CN,zh;q=0.9,en;q=0.8",
			"Sec-Ch-Ua":                 fmt.Sprintf(`"Not_A Brand";v="8", "Chromium";v="%s", "Google Chrome";v="%s"`, version, version),
			"Sec-Ch-Ua-Mobile":          "?0",
			"Sec-Ch-Ua-Platform":        `"Windows"`,
			"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
			"Accept-Encoding":           "gzip, deflate, br",
			"Upgrade-Insecure-Requests": "1",
		},
	}
}

// detectCaptcha 检测是否出现验证码/滑块
func (u *Uploader) detectCaptcha(page playwright.Page) (bool, string, error) {
	// 检测常见的验证码元素
	captchaSelectors := []struct {
		selector string
		type_    string
	}{
		{".captcha", "验证码"},
		{"[class*='captcha']", "验证码"},
		{"[class*='slider']", "滑块验证"},
		{"[class*='verify']", "验证"},
		{".geetest", "极验验证"},
		{"[class*='geetest']", "极验验证"},
		{"iframe[src*='captcha']", "验证码iframe"},
		{"iframe[src*='verify']", "验证iframe"},
		{"text=请完成验证", "文字验证"},
		{"text=拖动滑块", "滑块验证"},
		{"text=点击验证", "点击验证"},
	}

	for _, item := range captchaSelectors {
		count, err := page.Locator(item.selector).Count()
		if err == nil && count > 0 {
			// 检查是否可见
			visible, _ := page.Locator(item.selector).IsVisible()
			if visible {
				return true, item.type_, nil
			}
		}
	}

	// 检测页面文本中的验证提示
	verificationTexts := []string{
		"请完成安全验证",
		"请进行验证",
		"验证失败",
		"请点击",
		"请拖动",
	}

	for _, text := range verificationTexts {
		count, _ := page.GetByText(text).Count()
		if count > 0 {
			return true, "验证提示", nil
		}
	}

	return false, "", nil
}

// detectAntiBot 检测反爬虫标记
func (u *Uploader) detectAntiBot(page playwright.Page) (bool, string, error) {
	// 检测常见的反爬虫提示
	antiBotIndicators := []struct {
		selector string
		message  string
	}{
		{"text=访问过于频繁", "访问频繁"},
		{"text=操作过于频繁", "操作频繁"},
		{"text=请稍后再试", "限流提示"},
		{"text=系统繁忙", "系统繁忙"},
		{"text=网络异常", "网络异常"},
		{"text=账号异常", "账号异常"},
		{"text=登录异常", "登录异常"},
		{"text=自动程序", "自动程序检测"},
		{"text=机器人", "机器人检测"},
		{"[class*='ban']", "封禁提示"},
		{"[class*='block']", "拦截提示"},
	}

	for _, item := range antiBotIndicators {
		count, err := page.Locator(item.selector).Count()
		if err == nil && count > 0 {
			visible, _ := page.Locator(item.selector).IsVisible()
			if visible {
				return true, item.message, nil
			}
		}
	}

	return false, "", nil
}

// humanLikeDelay 模拟人类操作的随机延迟
func humanLikeDelay(baseDelay time.Duration) {
	// 添加随机波动（±30%）
	variance := float64(baseDelay) * 0.3
	delay := baseDelay + time.Duration(rand.Float64()*variance*2-variance)
	time.Sleep(delay)
}

// humanLikeTyping 模拟人类输入（带随机延迟）
func humanLikeTyping(page playwright.Page, text string) error {
	for _, char := range text {
		if err := page.Keyboard().Type(string(char)); err != nil {
			return err
		}
		// 随机延迟 50-150ms
		time.Sleep(time.Duration(50+rand.Intn(100)) * time.Millisecond)
	}
	return nil
}

// StepUploadVideoXiaohongshuEnhanced 增强版上传步骤（带验证码检测）
func (u *Uploader) StepUploadVideoXiaohongshuEnhanced() uploader.StepFunc {
	return func(ctx *uploader.Context) uploader.StepResult {
		ctx.ReportProgress(uploader.StepUploadMedia, 20, "开始上传视频...")

		// 检测验证码
		hasCaptcha, captchaType, _ := u.detectCaptcha(ctx.Page)
		if hasCaptcha {
			ctx.TakeScreenshot("captcha_detected")
			utils.Warn(fmt.Sprintf("[-] 检测到%s，请手动完成", captchaType))
			ctx.ReportProgress(uploader.StepUploadMedia, 21, fmt.Sprintf("检测到%s，等待手动完成...", captchaType))

			// 等待用户完成验证（最多3分钟）
			timeout := time.After(3 * time.Minute)
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()

			captchaHandled := false
			for !captchaHandled {
				select {
				case <-timeout:
					return uploader.StepResult{
						Step:    uploader.StepUploadMedia,
						Success: false,
						Error:   fmt.Errorf("验证码处理超时"),
					}
				case <-ticker.C:
					// 检查验证码是否消失
					hasCaptchaNow, _, _ := u.detectCaptcha(ctx.Page)
					if !hasCaptchaNow {
						utils.Info("[-] 验证码已处理，继续上传")
						captchaHandled = true
					}
				}
			}
		}

		// 检测反爬虫
		hasAntiBot, antiBotMsg, _ := u.detectAntiBot(ctx.Page)
		if hasAntiBot {
			ctx.TakeScreenshot("antibot_detected")
			utils.Warn(fmt.Sprintf("[-] 检测到反爬虫: %s", antiBotMsg))
			return uploader.StepResult{
				Step:    uploader.StepUploadMedia,
				Success: false,
				Error:   fmt.Errorf("触发反爬虫检测: %s", antiBotMsg),
			}
		}

		// 执行原始上传逻辑
		return u.StepUploadVideoXiaohongshu()(ctx)
	}
}

// simulateHumanBehavior 模拟人类浏览行为
func simulateHumanBehavior(page playwright.Page) error {
	// 随机滚动
	scrollCount := 2 + rand.Intn(3)
	for i := 0; i < scrollCount; i++ {
		scrollY := rand.Intn(300) + 100
		_, err := page.Evaluate(fmt.Sprintf("window.scrollBy(0, %d)", scrollY))
		if err != nil {
			return err
		}
		time.Sleep(time.Duration(500+rand.Intn(500)) * time.Millisecond)
	}

	// 随机鼠标移动
	err := page.Mouse().Move(float64(rand.Intn(500)+100), float64(rand.Intn(300)+100))
	if err != nil {
		return err
	}

	return nil
}

// init 初始化随机数种子
func init() {
	rand.Seed(time.Now().UnixNano())
}

// StepNavigateToHomepage 步骤1: 访问小红书主页（模拟自然浏览路径）
func (u *Uploader) StepNavigateToHomepage() uploader.StepFunc {
	return func(ctx *uploader.Context) uploader.StepResult {
		ctx.ReportProgress(uploader.StepNavigate, 5, "正在访问小红书主页...")

		// 获取浏览器上下文
		browserCtx, err := browserPool.GetContext(ctx.Ctx, u.GetCookiePath(), u.getEnhancedContextOptions())
		if err != nil {
			return uploader.StepResult{Step: uploader.StepNavigate, Success: false, Error: err}
		}
		ctx.BrowserCtx = browserCtx

		page, err := browserCtx.GetPage()
		if err != nil {
			return uploader.StepResult{Step: uploader.StepNavigate, Success: false, Error: err}
		}
		ctx.Page = page

		// 访问小红书主页
		if _, err := page.Goto("https://www.xiaohongshu.com", playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateNetworkidle,
		}); err != nil {
			utils.Warn(fmt.Sprintf("[-] 访问主页失败，将直接访问创作者页面: %v", err))
			// 主页访问失败不阻断流程，继续下一步
			return uploader.StepResult{Step: uploader.StepNavigate, Success: true}
		}

		ctx.ReportProgress(uploader.StepNavigate, 8, "已加载小红书主页")

		// 模拟人类浏览行为
		utils.Info("[-] 模拟浏览主页...")
		if err := simulateHumanBehavior(page); err != nil {
			utils.Warn(fmt.Sprintf("[-] 模拟浏览行为失败: %v", err))
		}

		humanLikeDelay(2 * time.Second)

		// 截图记录
		ctx.TakeScreenshot("homepage_visited")

		return uploader.StepResult{Step: uploader.StepNavigate, Success: true}
	}
}

// StepNavigateToCreatorUpload 步骤2: 从首页进入创作者中心发布页面
func (u *Uploader) StepNavigateToCreatorUpload() uploader.StepFunc {
	return func(ctx *uploader.Context) uploader.StepResult {
		ctx.ReportProgress(uploader.StepNavigate, 10, "正在进入创作者中心...")

		page := ctx.Page
		if page == nil {
			return uploader.StepResult{Step: uploader.StepNavigate, Success: false, Error: fmt.Errorf("page not initialized")}
		}

		// 检测验证码
		hasCaptcha, captchaType, _ := u.detectCaptcha(page)
		if hasCaptcha {
			ctx.TakeScreenshot("captcha_before_creator")
			utils.Warn(fmt.Sprintf("[-] 检测到%s，请手动完成", captchaType))
		}

		// 尝试点击"发布"按钮或导航到创作者中心
		// 方案1: 尝试点击页面上的发布按钮（如果存在）
		publishBtnSelectors := []string{
			"text=发布",
			"text=发布笔记",
			"[class*='publish']",
			"[class*='upload']",
		}

		clicked := false
		for _, selector := range publishBtnSelectors {
			btn := page.Locator(selector)
			count, _ := btn.Count()
			if count > 0 {
				visible, _ := btn.IsVisible()
				if visible {
					if err := btn.Click(); err == nil {
						utils.Info(fmt.Sprintf("[-] 点击发布按钮: %s", selector))
						clicked = true
						break
					}
				}
			}
		}

		// 方案2: 如果没有找到按钮或点击失败，直接导航到创作者页面
		if !clicked {
			utils.Info("[-] 未找到发布按钮，直接导航到创作者页面...")
			if _, err := page.Goto("https://creator.xiaohongshu.com/publish/publish?from=homepage&target=video", playwright.PageGotoOptions{
				WaitUntil: playwright.WaitUntilStateNetworkidle,
			}); err != nil {
				return uploader.StepResult{Step: uploader.StepNavigate, Success: false, Error: fmt.Errorf("goto creator page failed: %w", err)}
			}
		} else {
			// 等待页面跳转
			time.Sleep(3 * time.Second)

			// 检查是否成功进入创作者页面
			url := page.URL()
			if !strings.Contains(url, "creator.xiaohongshu.com") {
				// 如果没有跳转成功，手动导航
				utils.Info("[-] 未自动跳转到创作者页面，手动导航...")
				if _, err := page.Goto("https://creator.xiaohongshu.com/publish/publish?from=homepage&target=video", playwright.PageGotoOptions{
					WaitUntil: playwright.WaitUntilStateNetworkidle,
				}); err != nil {
					return uploader.StepResult{Step: uploader.StepNavigate, Success: false, Error: fmt.Errorf("goto creator page failed: %w", err)}
				}
			}
		}

		// 等待页面完全加载
		humanLikeDelay(3 * time.Second)

		// 检测验证码（进入创作者页面后可能出现）
		hasCaptcha, captchaType, _ = u.detectCaptcha(page)
		if hasCaptcha {
			ctx.TakeScreenshot("captcha_in_creator")
			utils.Warn(fmt.Sprintf("[-] 进入创作者页面后检测到%s，请手动完成", captchaType))
			ctx.ReportProgress(uploader.StepNavigate, 12, fmt.Sprintf("检测到%s，等待手动完成...", captchaType))

			// 等待用户完成验证（最多2分钟）
			timeout := time.After(2 * time.Minute)
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-timeout:
					utils.Warn("[-] 验证码处理超时，继续尝试上传")
					break
				case <-ticker.C:
					hasCaptchaNow, _, _ := u.detectCaptcha(page)
					if !hasCaptchaNow {
						utils.Info("[-] 验证码已处理，继续上传流程")
						break
					}
				}
			}
		}

		// 检测反爬虫
		hasAntiBot, antiBotMsg, _ := u.detectAntiBot(page)
		if hasAntiBot {
			ctx.TakeScreenshot("antibot_in_creator")
			utils.Error(fmt.Sprintf("[-] 触发反爬虫检测: %s", antiBotMsg))
			return uploader.StepResult{
				Step:    uploader.StepNavigate,
				Success: false,
				Error:   fmt.Errorf("anti-bot detected: %s", antiBotMsg),
			}
		}

		ctx.ReportProgress(uploader.StepNavigate, 15, "已进入创作者中心发布页面")

		// 截图记录
		ctx.TakeScreenshot("creator_upload_page")

		return uploader.StepResult{Step: uploader.StepNavigate, Success: true}
	}
}

// StepNavigateWithHumanBehavior 带人类行为的导航步骤（直接导航，保留用于其他场景）
func (u *Uploader) StepNavigateWithHumanBehavior(url string) uploader.StepFunc {
	return func(ctx *uploader.Context) uploader.StepResult {
		ctx.ReportProgress(uploader.StepNavigate, 10, "正在打开页面...")

		// 获取浏览器上下文
		browserCtx, err := u.GetBrowserPool().GetContext(ctx.Ctx, u.GetCookiePath(), u.getEnhancedContextOptions())
		if err != nil {
			return uploader.StepResult{Step: uploader.StepNavigate, Success: false, Error: err}
		}
		ctx.BrowserCtx = browserCtx

		page, err := browserCtx.GetPage()
		if err != nil {
			return uploader.StepResult{Step: uploader.StepNavigate, Success: false, Error: err}
		}
		ctx.Page = page

		ctx.ReportProgress(uploader.StepNavigate, 15, fmt.Sprintf("导航到: %s", url))
		if _, err := page.Goto(url, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateNetworkidle,
		}); err != nil {
			return uploader.StepResult{Step: uploader.StepNavigate, Success: false, Error: err}
		}

		// 等待页面加载
		humanLikeDelay(2 * time.Second)

		// 模拟人类浏览行为
		if err := simulateHumanBehavior(page); err != nil {
			utils.Warn(fmt.Sprintf("[-] 模拟浏览行为失败: %v", err))
		}

		// 截图记录
		ctx.TakeScreenshot("navigate_complete")

		return uploader.StepResult{Step: uploader.StepNavigate, Success: true}
	}
}

// StepFillTitleEnhanced 增强版填写标题步骤
func (u *Uploader) StepFillTitleEnhanced(newSelector string, oldSelector string, maxLength int) uploader.StepFunc {
	return func(ctx *uploader.Context) uploader.StepResult {
		ctx.ReportProgress(uploader.StepFillTitle, 45, "正在填写标题...")

		title := ctx.Task.Title
		if len(title) > maxLength {
			title = title[:maxLength]
		}

		// 尝试新页面结构
		newInput := ctx.Page.Locator(newSelector)
		newCount, _ := newInput.Count()
		if newCount > 0 {
			// 先点击输入框
			if err := newInput.Click(); err != nil {
				return uploader.StepResult{Step: uploader.StepFillTitle, Success: false, Error: err}
			}
			humanLikeDelay(500 * time.Millisecond)

			// 模拟人类输入
			if err := humanLikeTyping(ctx.Page, title); err != nil {
				return uploader.StepResult{Step: uploader.StepFillTitle, Success: false, Error: err}
			}
			ctx.ReportProgress(uploader.StepFillTitle, 50, "标题填写完成（新页面结构）")
			return uploader.StepResult{Step: uploader.StepFillTitle, Success: true, Data: map[string]interface{}{"type": "new"}}
		}

		// 尝试旧页面结构
		oldInput := ctx.Page.Locator(oldSelector)
		oldCount, _ := oldInput.Count()
		if oldCount > 0 {
			if err := oldInput.Click(); err != nil {
				return uploader.StepResult{Step: uploader.StepFillTitle, Success: false, Error: err}
			}
			humanLikeDelay(500 * time.Millisecond)

			// 清空原有内容
			if err := ctx.Page.Keyboard().Press("Backspace"); err != nil {
				return uploader.StepResult{Step: uploader.StepFillTitle, Success: false, Error: err}
			}
			if err := ctx.Page.Keyboard().Press("Control+KeyA"); err != nil {
				return uploader.StepResult{Step: uploader.StepFillTitle, Success: false, Error: err}
			}
			if err := ctx.Page.Keyboard().Press("Delete"); err != nil {
				return uploader.StepResult{Step: uploader.StepFillTitle, Success: false, Error: err}
			}
			humanLikeDelay(300 * time.Millisecond)

			// 模拟人类输入
			if err := humanLikeTyping(ctx.Page, title); err != nil {
				return uploader.StepResult{Step: uploader.StepFillTitle, Success: false, Error: err}
			}
			if err := ctx.Page.Keyboard().Press("Enter"); err != nil {
				return uploader.StepResult{Step: uploader.StepFillTitle, Success: false, Error: err}
			}
			ctx.ReportProgress(uploader.StepFillTitle, 50, "标题填写完成（旧页面结构）")
			return uploader.StepResult{Step: uploader.StepFillTitle, Success: true, Data: map[string]interface{}{"type": "old"}}
		}

		return uploader.StepResult{Step: uploader.StepFillTitle, Success: false, Error: fmt.Errorf("title input not found, tried new: %s, old: %s", newSelector, oldSelector)}
	}
}

// StepAddTagsEnhanced 增强版添加标签步骤
func (u *Uploader) StepAddTagsEnhanced(contentEditableSelector string) uploader.StepFunc {
	return func(ctx *uploader.Context) uploader.StepResult {
		ctx.ReportProgress(uploader.StepAddTags, 52, "正在添加标签...")

		// 点击内容编辑区域
		editor := ctx.Page.Locator(contentEditableSelector)
		if err := editor.Click(); err != nil {
			return uploader.StepResult{Step: uploader.StepAddTags, Success: false, Error: err}
		}
		humanLikeDelay(300 * time.Millisecond)

		for i, tag := range ctx.Task.Tags {
			// 添加标签前随机延迟
			if i > 0 {
				humanLikeDelay(800 * time.Millisecond)
			}

			// 模拟人类输入标签
			if err := humanLikeTyping(ctx.Page, "#"+tag); err != nil {
				return uploader.StepResult{Step: uploader.StepAddTags, Success: false, Error: err}
			}
			humanLikeDelay(200 * time.Millisecond)

			// 按空格
			if err := ctx.Page.Keyboard().Press("Space"); err != nil {
				return uploader.StepResult{Step: uploader.StepAddTags, Success: false, Error: err}
			}
		}

		ctx.ReportProgress(uploader.StepAddTags, 55, fmt.Sprintf("已添加 %d 个标签", len(ctx.Task.Tags)))
		return uploader.StepResult{Step: uploader.StepAddTags, Success: true}
	}
}

// StepSetThumbnailEnhanced 增强版设置封面步骤
func (u *Uploader) StepSetThumbnailEnhanced() uploader.StepFunc {
	return func(ctx *uploader.Context) uploader.StepResult {
		if ctx.Task.Thumbnail == "" {
			return uploader.StepResult{Step: uploader.StepSetCover, Success: true}
		}

		ctx.ReportProgress(uploader.StepSetCover, 58, "正在设置封面...")

		// 点击"选择封面"
		if err := ctx.Page.GetByText("选择封面").Click(); err != nil {
			utils.Warn(fmt.Sprintf("[-] 点击选择封面失败: %v", err))
			return uploader.StepResult{Step: uploader.StepSetCover, Success: true}
		}
		humanLikeDelay(2 * time.Second)

		// 点击"设置竖封面"
		if err := ctx.Page.GetByText("设置竖封面").Click(); err != nil {
			utils.Warn(fmt.Sprintf("[-] 点击设置竖封面失败: %v", err))
			return uploader.StepResult{Step: uploader.StepSetCover, Success: true}
		}
		humanLikeDelay(2 * time.Second)

		// 上传封面文件
		coverInput := ctx.Page.Locator("div[class^='semi-upload upload'] >> input.semi-upload-hidden-input")
		if err := coverInput.SetInputFiles(ctx.Task.Thumbnail); err != nil {
			utils.Warn(fmt.Sprintf("[-] 上传封面失败: %v", err))
			return uploader.StepResult{Step: uploader.StepSetCover, Success: true}
		}
		humanLikeDelay(2 * time.Second)

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

// StepSetLocationEnhanced 增强版设置位置步骤
func (u *Uploader) StepSetLocationEnhanced() uploader.StepFunc {
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
		humanLikeDelay(1 * time.Second)

		// 模拟人类输入位置
		if err := humanLikeTyping(ctx.Page, ctx.Task.Location); err != nil {
			utils.Warn(fmt.Sprintf("[-] 输入位置失败: %v", err))
			return uploader.StepResult{Step: uploader.StepFillContent, Success: true}
		}
		humanLikeDelay(3 * time.Second)

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

// StepSetScheduleXiaohongshuEnhanced 增强版小红书专用定时发布步骤
func (u *Uploader) StepSetScheduleXiaohongshuEnhanced() uploader.StepFunc {
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
		humanLikeDelay(1 * time.Second)

		// 设置日期时间
		scheduleInput := ctx.Page.Locator(".el-input__inner[placeholder=\"选择日期和时间\"]")
		if err := scheduleInput.Click(); err != nil {
			return uploader.StepResult{Step: uploader.StepSetSchedule, Success: false, Error: err}
		}
		humanLikeDelay(300 * time.Millisecond)

		if err := ctx.Page.Keyboard().Press("Control+KeyA"); err != nil {
			return uploader.StepResult{Step: uploader.StepSetSchedule, Success: false, Error: err}
		}
		if err := humanLikeTyping(ctx.Page, *ctx.Task.ScheduleTime); err != nil {
			return uploader.StepResult{Step: uploader.StepSetSchedule, Success: false, Error: err}
		}
		if err := ctx.Page.Keyboard().Press("Enter"); err != nil {
			return uploader.StepResult{Step: uploader.StepSetSchedule, Success: false, Error: err}
		}

		ctx.ReportProgress(uploader.StepSetSchedule, 80, "定时发布设置完成")
		return uploader.StepResult{Step: uploader.StepSetSchedule, Success: true}
	}
}

// StepClickPublishXiaohongshuEnhanced 增强版小红书专用发布步骤
func (u *Uploader) StepClickPublishXiaohongshuEnhanced() uploader.StepFunc {
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

		// 使用 WaitForURL 等待跳转到成功页面（参考Python实现）
		// 创建一个channel来接收WaitForURL的结果
		resultChan := make(chan error, 1)
		go func() {
			err := ctx.Page.WaitForURL("**/publish/success**", playwright.PageWaitForURLOptions{
				Timeout: playwright.Float(30000), // 30秒超时
			})
			resultChan <- err
		}()

		// 同时检测验证码和反爬虫
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		timeout := time.After(35 * time.Second)

		for {
			select {
			case err := <-resultChan:
				// WaitForURL 返回结果
				if err == nil {
					// 发布成功，等待页面稳定（参考Python的sleep(2)）
					utils.Info("[-] 发布成功，等待页面稳定...")
					humanLikeDelay(2 * time.Second)

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
				// WaitForURL 超时或失败，继续检测
				utils.Warn(fmt.Sprintf("[-] WaitForURL返回: %v", err))

			case <-timeout:
				ctx.TakeScreenshot("publish_timeout")
				return uploader.StepResult{Step: uploader.StepPublish, Success: false, Error: fmt.Errorf("publish timeout after 30s")}

			case <-ticker.C:
				// 检测验证码
				hasCaptcha, captchaType, _ := u.detectCaptcha(ctx.Page)
				if hasCaptcha {
					ctx.TakeScreenshot("publish_captcha")
					utils.Warn(fmt.Sprintf("[-] 发布时检测到%s", captchaType))
				}

				// 检测反爬虫
				hasAntiBot, antiBotMsg, _ := u.detectAntiBot(ctx.Page)
				if hasAntiBot {
					ctx.TakeScreenshot("publish_antibot")
					return uploader.StepResult{
						Step:    uploader.StepPublish,
						Success: false,
						Error:   fmt.Errorf("anti-bot detected during publish: %s", antiBotMsg),
					}
				}

				// 截图记录发布状态
				ctx.TakeScreenshot("publishing")
			}
		}
	}
}
