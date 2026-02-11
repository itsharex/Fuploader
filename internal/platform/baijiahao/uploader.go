package baijiahao

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
		utils.InfoWithPlatform("baijiahao", fmt.Sprintf("[调试] "+format, args...))
	}
}

// browserPool 全局浏览器池实例
var browserPool *browser.Pool

func init() {
	browserPool = browser.NewPool(2, 5)
}

// Uploader 百家号上传器
type Uploader struct {
	accountID  uint
	cookiePath string
	platform   string
}

// NewUploader 创建上传器（兼容旧版，使用cookiePath）
func NewUploader(cookiePath string) *Uploader {
	u := &Uploader{
		accountID:  0,
		cookiePath: cookiePath,
		platform:   "baijiahao",
	}
	debugLog("创建上传器 - 地址: %p, cookiePath: '%s'", u, cookiePath)
	if cookiePath == "" {
		utils.Warn("[Baijiahao] NewUploader 收到空的cookiePath!")
	}
	return u
}

// NewUploaderWithAccount 创建带accountID的上传器（新接口）
func NewUploaderWithAccount(accountID uint) *Uploader {
	cookiePath := config.GetCookiePath("baijiahao", int(accountID))
	u := &Uploader{
		accountID:  accountID,
		cookiePath: cookiePath,
		platform:   "baijiahao",
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

	if _, err := page.Goto("https://baijiahao.baidu.com/builder/rc/home", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 验证Cookie - 打开页面失败: %v", err))
		return false, nil
	}

	time.Sleep(5 * time.Second)

	// 使用Cookie检测机制验证登录状态
	cookieConfig, ok := browser.GetCookieConfig("baijiahao")
	if !ok {
		return false, fmt.Errorf("失败: 验证Cookie - 获取Cookie配置失败")
	}

	isValid, err := browserCtx.ValidateLoginCookies(cookieConfig)
	if err != nil {
		return false, fmt.Errorf("失败: 验证Cookie - %w", err)
	}

	if isValid {
		utils.InfoWithPlatform(u.platform, "检测到PTOKEN和BAIDUID Cookie，验证通过")
	} else {
		utils.InfoWithPlatform(u.platform, "未检测到PTOKEN和BAIDUID Cookie，验证失败")
	}

	return isValid, nil
}

// Upload 上传视频
func (u *Uploader) Upload(ctx context.Context, task *types.VideoTask) error {
	utils.InfoWithPlatform(u.platform, fmt.Sprintf("开始上传: %s", task.VideoPath))

	if _, err := os.Stat(task.VideoPath); err != nil {
		return fmt.Errorf("视频文件不存在: %w", err)
	}

	// 使用accountID获取上下文（如果accountID为0则退化为旧逻辑）
	browserCtx, err := browserPool.GetContextByAccount(ctx, u.accountID, u.cookiePath, nil)
	if err != nil {
		return fmt.Errorf("获取浏览器失败: %w", err)
	}
	defer browserCtx.Release()

	page, err := browserCtx.GetPage()
	if err != nil {
		return fmt.Errorf("获取页面失败: %w", err)
	}

	// 导航到发布页面
	utils.InfoWithPlatform(u.platform, "正在打开发布页面...")
	if _, err := page.Goto("https://baijiahao.baidu.com/builder/rc/edit?type=videoV2", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		return fmt.Errorf("失败: 打开发布页面 - %w", err)
	}

	// 等待页面加载
	if err := page.Locator("div#formMain:visible").WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(10000),
	}); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 等待页面加载 - 超时: %v", err))
	}
	time.Sleep(3 * time.Second)

	// 上传视频
	utils.InfoWithPlatform(u.platform, "正在上传视频...")
	inputLocator := page.Locator("div[class^='video-main-container'] input[type='file']").First()
	if count, _ := inputLocator.Count(); count == 0 {
		inputLocator = page.Locator("input[type='file']").First()
	}

	if err := inputLocator.SetInputFiles(task.VideoPath); err != nil {
		return fmt.Errorf("失败: 上传视频 - %w", err)
	}

	// 等待上传完成
	if err := u.waitForUploadComplete(ctx, page, browserCtx); err != nil {
		return fmt.Errorf("失败: 等待视频上传 - %w", err)
	}

	// 填写标题
	if err := u.fillTitle(page, task.Title); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 填写标题 - %v", err))
	}

	// 填写描述
	if err := u.fillDescription(page, task.Description); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 填写描述 - %v", err))
	}

	// 添加标签
	if err := u.addTags(page, task.Tags); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 添加标签 - %v", err))
	}

	// 选择内容分类
	if err := u.selectCategory(page, task.Category); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 选择内容分类 - %v", err))
	}

	// 勾选AI创作声明
	if err := u.setAICreation(page); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 勾选AI创作声明 - %v", err))
	}

	// 勾选自动生成音频
	if err := u.setAutoAudio(page); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 勾选自动生成音频 - %v", err))
	}

	// 设置自定义封面
	if task.Thumbnail != "" {
		if err := u.setCustomCover(page, task.Thumbnail); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 设置自定义封面 - %v", err))
		}
	}

	// 设置定时发布或立即发布（互斥）
	if task.ScheduleTime != nil && *task.ScheduleTime != "" {
		// 执行定时发布
		if err := u.setScheduleTime(page, *task.ScheduleTime); err != nil {
			return fmt.Errorf("失败: 设置定时发布 - %w", err)
		}
	} else {
		// 执行立即发布
		utils.InfoWithPlatform(u.platform, "准备发布...")
		if err := u.publish(page, browserCtx); err != nil {
			return fmt.Errorf("失败: 发布 - %w", err)
		}
	}

	utils.SuccessWithPlatform(u.platform, "发布成功")
	return nil
}

// waitForUploadComplete 等待视频上传完成
func (u *Uploader) waitForUploadComplete(ctx context.Context, page playwright.Page, browserCtx *browser.PooledContext) error {
	utils.InfoWithPlatform(u.platform, "等待视频上传完成...")

	// 核心判定：等待封面预览区域出现（超时30秒）
	// 该元素出现 = 视频上传成功，是最稳定的判定依据
	coverImg := page.Locator(`div[class*="cover-container"] img[class*="coverImg"]`).First()
	if err := coverImg.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(30000),
	}); err != nil {
		return fmt.Errorf("失败: 等待视频上传 - 封面预览区域超时: %w", err)
	}

	utils.InfoWithPlatform(u.platform, "视频上传完成")
	return nil
}

// fillTitle 填写标题
func (u *Uploader) fillTitle(page playwright.Page, title string) error {
	if title == "" {
		return nil
	}

	utils.InfoWithPlatform(u.platform, "填写标题...")

	// 百家号标题少于8字符时自动补全
	if len(title) < 8 {
		title = title + " 你不知道的"
		utils.InfoWithPlatform(u.platform, fmt.Sprintf("标题少于8字符，自动补全为: %s", title))
	}

	// 限制30字符
	if len(title) > 30 {
		runes := []rune(title)
		if len(runes) > 30 {
			title = string(runes[:30])
		}
	}

	// 使用用户指定的选择器
	titleInput := page.Locator(`div[contenteditable="true"][aria-placeholder="添加标题获得更多推荐"]`).First()
	if err := titleInput.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		return fmt.Errorf("失败: 填写标题 - 未找到输入框: %w", err)
	}

	if err := titleInput.Fill(title); err != nil {
		return fmt.Errorf("失败: 填写标题 - %w", err)
	}

	utils.InfoWithPlatform(u.platform, fmt.Sprintf("标题已填写: %s", title))
	time.Sleep(500 * time.Millisecond)
	return nil
}

// fillDescription 填写描述
func (u *Uploader) fillDescription(page playwright.Page, description string) error {
	if description == "" {
		return nil
	}

	utils.InfoWithPlatform(u.platform, "填写描述...")

	// 使用用户指定的选择器
	descInput := page.Locator(`textarea#desc`).First()
	if err := descInput.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		return fmt.Errorf("失败: 填写描述 - 未找到输入框: %w", err)
	}

	if err := descInput.Fill(description); err != nil {
		return fmt.Errorf("失败: 填写描述 - %w", err)
	}

	utils.InfoWithPlatform(u.platform, "描述已填写")
	time.Sleep(500 * time.Millisecond)
	return nil
}

// addTags 添加标签
func (u *Uploader) addTags(page playwright.Page, tags []string) error {
	if len(tags) == 0 {
		return nil
	}

	utils.InfoWithPlatform(u.platform, "添加标签...")

	// 使用用户指定的选择器
	tagInput := page.Locator(`input.cheetah-ui-pro-tag-input-container-tag-input`).First()
	if err := tagInput.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		return fmt.Errorf("失败: 添加标签 - 未找到输入框: %w", err)
	}

	for i, tag := range tags {
		// 清理标签中的特殊字符
		cleanTag := strings.TrimSpace(tag)
		cleanTag = strings.ReplaceAll(cleanTag, "#", "")
		cleanTag = strings.ReplaceAll(cleanTag, " ", "")

		if cleanTag == "" {
			continue
		}

		if err := tagInput.Fill(cleanTag); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 添加标签 - 输入标签[%d]失败: %v", i, err))
			continue
		}
		time.Sleep(300 * time.Millisecond)

		// 按Enter确认标签
		if err := tagInput.Press("Enter"); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 添加标签 - 确认标签[%d]失败: %v", i, err))
			continue
		}
		time.Sleep(500 * time.Millisecond)
	}

	utils.InfoWithPlatform(u.platform, "标签添加完成")
	return nil
}

// selectCategory 选择内容分类
func (u *Uploader) selectCategory(page playwright.Page, category string) error {
	if category == "" {
		category = "科技"
	}

	utils.InfoWithPlatform(u.platform, fmt.Sprintf("选择内容分类: %s", category))

	// 点击分类选择器
	categoryInput := page.Locator(`input#rc_select_22`).First()
	if err := categoryInput.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		return fmt.Errorf("失败: 选择内容分类 - 未找到选择器: %w", err)
	}

	if err := categoryInput.Click(); err != nil {
		return fmt.Errorf("失败: 选择内容分类 - 点击选择器失败: %w", err)
	}
	time.Sleep(1 * time.Second)

	// 选择分类
	categoryOption := page.Locator(fmt.Sprintf(`span:has-text("%s")`, category)).First()
	if err := categoryOption.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		return fmt.Errorf("失败: 选择内容分类 - 未找到选项: %w", err)
	}

	if err := categoryOption.Click(); err != nil {
		return fmt.Errorf("失败: 选择内容分类 - 点击选项失败: %w", err)
	}

	utils.InfoWithPlatform(u.platform, fmt.Sprintf("分类已选择: %s", category))
	time.Sleep(500 * time.Millisecond)
	return nil
}

// setAICreation 勾选AI创作声明
func (u *Uploader) setAICreation(page playwright.Page) error {
	utils.InfoWithPlatform(u.platform, "勾选AI创作声明...")

	aiCheckbox := page.Locator(`span:has-text("AI创作声明")`).First()
	if err := aiCheckbox.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		return fmt.Errorf("失败: 勾选AI创作声明 - 未找到选项: %w", err)
	}

	if err := aiCheckbox.Click(); err != nil {
		return fmt.Errorf("失败: 勾选AI创作声明 - %w", err)
	}

	utils.InfoWithPlatform(u.platform, "AI创作声明已勾选")
	time.Sleep(500 * time.Millisecond)
	return nil
}

// setAutoAudio 勾选自动生成音频
func (u *Uploader) setAutoAudio(page playwright.Page) error {
	utils.InfoWithPlatform(u.platform, "勾选自动生成音频...")

	audioCheckbox := page.Locator(`span:has-text("自动生成音频")`).First()
	if err := audioCheckbox.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		return fmt.Errorf("失败: 勾选自动生成音频 - 未找到选项: %w", err)
	}

	if err := audioCheckbox.Click(); err != nil {
		return fmt.Errorf("失败: 勾选自动生成音频 - %w", err)
	}

	utils.InfoWithPlatform(u.platform, "自动生成音频已勾选")
	time.Sleep(500 * time.Millisecond)
	return nil
}

// setCustomCover 设置自定义封面
func (u *Uploader) setCustomCover(page playwright.Page, coverPath string) error {
	if _, err := os.Stat(coverPath); err != nil {
		return fmt.Errorf("失败: 设置封面 - 文件不存在: %w", err)
	}

	utils.InfoWithPlatform(u.platform, "设置封面...")

	// 点击封面区域打开选择弹窗
	coverArea := page.Locator("div[class^='cover'], div.cover-area, div.cheetah-spin-container").First()
	if err := coverArea.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		return fmt.Errorf("失败: 设置封面 - 未找到封面区域: %w", err)
	}

	if err := coverArea.Click(); err != nil {
		return fmt.Errorf("失败: 设置封面 - 点击封面区域失败: %w", err)
	}
	time.Sleep(2 * time.Second)

	// 查找文件输入框
	coverInput := page.Locator("input[type='file'][accept*='image']").First()
	if err := coverInput.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		return fmt.Errorf("失败: 设置封面 - 未找到文件输入框: %w", err)
	}

	if err := coverInput.SetInputFiles(coverPath); err != nil {
		return fmt.Errorf("失败: 设置封面 - 上传失败: %w", err)
	}

	utils.InfoWithPlatform(u.platform, "封面上传中...")
	time.Sleep(3 * time.Second)

	// 点击确认或完成按钮
	confirmBtn := page.Locator("button:has-text('确认'), button:has-text('完成'), button:has-text('确定')").First()
	if count, _ := confirmBtn.Count(); count > 0 {
		if err := confirmBtn.Click(); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 设置封面 - 点击确认按钮失败: %v", err))
		}
		time.Sleep(1 * time.Second)
	}

	utils.InfoWithPlatform(u.platform, "封面设置完成")
	return nil
}

// setScheduleTime 设置定时发布
func (u *Uploader) setScheduleTime(page playwright.Page, scheduleTime string) error {
	utils.InfoWithPlatform(u.platform, fmt.Sprintf("设置定时发布: %s", scheduleTime))

	// 解析时间
	targetTime, err := time.Parse("2006-01-02 15:04", scheduleTime)
	if err != nil {
		return fmt.Errorf("失败: 设置定时发布 - 解析时间失败: %w", err)
	}

	// 1. 点击"定时发布"按钮，打开时间选择弹窗
	page.Locator(`span:has-text("定时发布")`).First().Click()
	time.Sleep(1 * time.Second)

	// 格式化日期和时间的显示文本
	monthDay := fmt.Sprintf("%d月%d日", targetTime.Month(), targetTime.Day())
	hour := fmt.Sprintf("%d", targetTime.Hour())
	minute := fmt.Sprintf("%d", targetTime.Minute())

	// 2. 选择日期
	page.Locator(fmt.Sprintf(`div:has-text("选择日期") + div span.cheetah-select-selection-item[title="%s"]`, monthDay)).First().Click()
	time.Sleep(500 * time.Millisecond)

	// 3. 选择小时：先定位"小时"标题的父容器，再找数字
	page.Locator(fmt.Sprintf(`div:has-text("小时") ~ div span:has-text("%s")`, hour)).First().Click()
	time.Sleep(300 * time.Millisecond)
	// 同样隔离后选"点"
	page.Locator(`div:has-text("小时") ~ div span:has-text("点")`).First().Click()
	time.Sleep(300 * time.Millisecond)

	// 4. 选择分钟：先定位"分钟"标题的父容器，再找数字
	page.Locator(fmt.Sprintf(`div:has-text("分钟") ~ div span:has-text("%s")`, minute)).First().Click()
	time.Sleep(300 * time.Millisecond)
	// 同样隔离后选"分"
	page.Locator(`div:has-text("分钟") ~ div span:has-text("分")`).First().Click()
	time.Sleep(300 * time.Millisecond)

	// 5. 确认定时发布
	page.Locator(`button:has-text("定时发布")`).First().Click()

	utils.InfoWithPlatform(u.platform, "定时发布设置完成")
	time.Sleep(1 * time.Second)
	return nil
}

// publish 点击发布并检测结果
func (u *Uploader) publish(page playwright.Page, browserCtx *browser.PooledContext) error {
	// 定位发布按钮
	publishBtn := page.Locator("button:has-text('发布'), button:has-text('立即发布')").First()
	if err := publishBtn.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		return fmt.Errorf("失败: 发布 - 未找到发布按钮: %w", err)
	}

	// 滚动到按钮可见
	if err := publishBtn.ScrollIntoViewIfNeeded(); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 发布 - 滚动到按钮失败: %v", err))
	}

	urlBeforePublish := page.URL()
	maxAttempts := 3

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if browserCtx.IsPageClosed() {
			return fmt.Errorf("失败: 发布 - 浏览器已关闭")
		}

		utils.InfoWithPlatform(u.platform, fmt.Sprintf("第%d次尝试发布...", attempt+1))

		// 点击发布按钮
		if err := publishBtn.Click(playwright.LocatorClickOptions{
			Force: playwright.Bool(true),
		}); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 发布 - 点击按钮失败: %v", err))
			time.Sleep(2 * time.Second)
			continue
		}

		utils.InfoWithPlatform(u.platform, "已点击发布按钮")
		time.Sleep(3 * time.Second)

		// 处理确认弹窗
		confirmDialog := page.Locator("button:has-text('确定'), button:has-text('确认')").First()
		if count, _ := confirmDialog.Count(); count > 0 {
			if visible, _ := confirmDialog.IsVisible(); visible {
				utils.InfoWithPlatform(u.platform, "处理确认弹窗...")
				confirmDialog.Click()
				time.Sleep(2 * time.Second)
			}
		}

		// 检测发布结果
		if err := u.checkPublishResult(page, browserCtx, urlBeforePublish); err == nil {
			return nil
		} else {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 发布 - 检测未通过: %v", err))
		}

		if attempt < maxAttempts-1 {
			time.Sleep(3 * time.Second)
		}
	}

	return fmt.Errorf("失败: 发布 - 已重试%d次", maxAttempts)
}

// checkPublishResult 检测发布结果
func (u *Uploader) checkPublishResult(page playwright.Page, browserCtx *browser.PooledContext, urlBefore string) error {
	checkTimeout := 60 * time.Second
	checkInterval := 2 * time.Second
	checkStart := time.Now()

	for time.Since(checkStart) < checkTimeout {
		if browserCtx.IsPageClosed() {
			return fmt.Errorf("失败: 检测发布结果 - 浏览器已关闭")
		}

		currentURL := page.URL()

		// 成功标志1：URL跳转到管理页
		if strings.Contains(currentURL, "baijiahao.baidu.com/builder/rc/clue") ||
			strings.Contains(currentURL, "baijiahao.baidu.com/builder/rc/manage") {
			return nil
		}

		// 成功标志2：URL变化且不再包含edit
		if currentURL != urlBefore && !strings.Contains(currentURL, "edit") {
			return nil
		}

		// 成功标志3：成功提示文本
		successIndicators := []string{"发布成功", "提交成功", "审核中", "稿件已提交"}
		for _, indicator := range successIndicators {
			successText := page.Locator(fmt.Sprintf("text=%s", indicator)).First()
			if count, _ := successText.Count(); count > 0 {
				if visible, _ := successText.IsVisible(); visible {
					return nil
				}
			}
		}

		// 失败标志
		errorIndicators := []string{"发布失败", "提交失败", "错误", "请完善"}
		for _, indicator := range errorIndicators {
			errorText := page.Locator(fmt.Sprintf("text=%s", indicator)).First()
			if count, _ := errorText.Count(); count > 0 {
				if visible, _ := errorText.IsVisible(); visible {
					text, _ := errorText.TextContent()
					return fmt.Errorf("失败: 检测发布结果 - 页面错误: %s", text)
				}
			}
		}

		time.Sleep(checkInterval)
	}

	return fmt.Errorf("失败: 检测发布结果 - 超时")
}

// Login 登录
func (u *Uploader) Login() error {
	debugLog("Login开始 - cookiePath: '%s'", u.cookiePath)
	if u.cookiePath == "" {
		return fmt.Errorf("失败: 登录 - cookie路径为空")
	}

	ctx := context.Background()
	utils.InfoWithPlatform(u.platform, "正在打开登录页面...")

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

	if _, err := page.Goto("https://baijiahao.baidu.com/builder/theme/bjh/login", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		return fmt.Errorf("失败: 登录 - 打开登录页面失败: %w", err)
	}

	time.Sleep(3 * time.Second)

	utils.InfoWithPlatform(u.platform, "请在浏览器窗口中完成登录...")

	// 使用Cookie检测机制等待登录成功
	cookieConfig, ok := browser.GetCookieConfig("baijiahao")
	if !ok {
		return fmt.Errorf("失败: 登录 - 获取Cookie配置失败")
	}

	if err := browserCtx.WaitForLoginCookies(cookieConfig); err != nil {
		return fmt.Errorf("失败: 登录 - 等待Cookie失败: %w", err)
	}

	utils.SuccessWithPlatform(u.platform, "登录成功")
	if err := browserCtx.SaveCookiesTo(u.cookiePath); err != nil {
		return fmt.Errorf("失败: 登录 - 保存Cookie失败: %w", err)
	}
	return nil
}
