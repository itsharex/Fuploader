package baijiahao

import (
	"context"
	"fmt"
	"math/rand"
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

// Uploader 百家号上传器
type Uploader struct {
	*uploader.Base
}

// NewUploader 创建上传器
func NewUploader(cookiePath string) *Uploader {
	initBrowserPool()
	return &Uploader{
		Base: uploader.NewBase("baijiahao", cookiePath, browserPool),
	}
}

// Platform 返回平台名称
func (u *Uploader) Platform() string {
	return "baijiahao"
}

// ValidateCookie 验证 Cookie 是否有效
// 参照Python版本：访问首页，等待5秒，检测"注册/登录百家号"文字
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

	// 访问百家号创作者中心首页
	utils.Info("[-] 正在验证百家号登录状态...")
	if _, err := page.Goto("https://baijiahao.baidu.com/builder/rc/home"); err != nil {
		return false, fmt.Errorf("goto home page failed: %w", err)
	}

	// 等待5秒（与Python版本一致）
	time.Sleep(5 * time.Second)

	// 检测是否有"注册/登录百家号"文字（Python版本核心逻辑）
	loginTextCount, _ := page.GetByText("注册/登录百家号").Count()
	if loginTextCount > 0 {
		utils.Info("[-] 检测到'注册/登录百家号'，Cookie 失效")
		return false, nil
	}

	utils.Info("[-] Cookie 有效")
	return true, nil
}

// Upload 上传视频
func (u *Uploader) Upload(ctx context.Context, task *types.VideoTask) error {
	steps := []uploader.StepFunc{
		// 1. 导航到发布页面 - 参照Python版本直接访问edit页面
		u.StepNavigate("https://baijiahao.baidu.com/builder/rc/edit?type=videoV2"),

		// 2. 等待页面加载并上传视频
		u.StepUploadBaijiahaoVideo(task.VideoPath),

		// 3. 等待封面生成完成
		u.StepWaitCoverGenerated(),

		// 4. 填写标题、简介和标签
		u.StepFillBaijiahaoContent(task.Title, task.Description, task.Tags),

		// 5. 设置定时发布（如果有）
		u.StepSetScheduleBaijiahao(task.ScheduleTime),

		// 6. 检测安全验证
		u.StepCheckSecurityVerification(),

		// 7. 点击发布
		u.StepClickPublishBaijiahao(),
	}

	return u.Execute(ctx, task, steps)
}

// StepUploadBaijiahaoVideo 百家号上传视频步骤
func (u *Uploader) StepUploadBaijiahaoVideo(videoPath string) uploader.StepFunc {
	return func(ctx *uploader.Context) uploader.StepResult {
		// 等待页面完全加载
		utils.Info("[-] 等待发布页面加载...")
		time.Sleep(3 * time.Second)

		// 检查是否进入视频发布页面
		maxWait := 30
		for i := 0; i < maxWait; i++ {
			// 检查页面URL
			currentURL := ctx.Page.URL()
			if strings.Contains(currentURL, "edit?type=videoV2") {
				// 检查是否有视频上传输入框
				inputCount, _ := ctx.Page.Locator("div[class^='video-main-container'] input, input[type='file']").Count()
				if inputCount > 0 {
					utils.Info("[-] 已进入视频发布页面")
					break
				}
			}
			
			// 检查是否有"注册/登录百家号"文字（未登录）
			loginTextCount, _ := ctx.Page.GetByText("注册/登录百家号").Count()
			if loginTextCount > 0 {
				return uploader.StepResult{Step: uploader.StepUploadMedia, Success: false, Error: fmt.Errorf("未登录，请先登录百家号")}
			}
			
			utils.Info(fmt.Sprintf("[-] 正在等待进入视频发布页面... (%d/%d)", i+1, maxWait))
			time.Sleep(1 * time.Second)
		}

		// 上传视频文件 - 参照Python版本的选择器
		utils.Info("[-] 正在上传视频...")
		inputLocator := ctx.Page.Locator("div[class^='video-main-container'] input").First()
		if err := inputLocator.SetInputFiles(videoPath); err != nil {
			// 尝试备用选择器
			inputLocator = ctx.Page.Locator("input[type='file']").First()
			if err := inputLocator.SetInputFiles(videoPath); err != nil {
				return uploader.StepResult{Step: uploader.StepUploadMedia, Success: false, Error: fmt.Errorf("set input files failed: %w", err)}
			}
		}

		// 等待上传完成
		timeout := time.After(5 * time.Minute)
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-timeout:
				return uploader.StepResult{Step: uploader.StepUploadMedia, Success: false, Error: fmt.Errorf("upload timeout")}
			case <-ticker.C:
				// 检查上传失败
				failCount, _ := ctx.Page.Locator("div .cover-overlay:has-text('上传失败')").Count()
				if failCount > 0 {
					return uploader.StepResult{Step: uploader.StepUploadMedia, Success: false, Error: fmt.Errorf("upload failed")}
				}
				
				// 检查上传完成（Python版本逻辑）
				uploadingCount, _ := ctx.Page.Locator("div .cover-overlay:has-text('上传中')").Count()
				if uploadingCount == 0 {
					// 再检查是否有上传完成的标志
					formVisible, _ := ctx.Page.Locator("div#formMain:visible").Count()
					if formVisible > 0 {
						utils.Info("[-] 视频上传完成")
						return uploader.StepResult{Step: uploader.StepUploadMedia, Success: true}
					}
				}
				utils.Info("[-] 正在上传视频中...")
			}
		}
	}
}

// StepFillBaijiahaoContent 百家号填写内容步骤
func (u *Uploader) StepFillBaijiahaoContent(title, description string, tags []string) uploader.StepFunc {
	return func(ctx *uploader.Context) uploader.StepResult {
		// 标题优化：如果标题少于等于8个字，自动补充（参照Python版本）
		optimizedTitle := title
		if len(title) <= 8 {
			optimizedTitle = title + " 你不知道的"
			utils.Info(fmt.Sprintf("[-] 标题优化：原标题过短，已优化为: %s", optimizedTitle))
		}

		// 限制标题长度不超过30个字符（参照Python版本）
		if len(optimizedTitle) > 30 {
			optimizedTitle = optimizedTitle[:30]
		}

		// 设置标题 - 参照Python版本使用placeholder定位
		titleInput := ctx.Page.GetByPlaceholder("添加标题获得更多推荐")
		if err := titleInput.Fill(optimizedTitle); err != nil {
			// 尝试备用选择器
			titleInput = ctx.Page.Locator("input[placeholder*=\"标题\"], textarea[placeholder*=\"标题\"]").First()
			if err := titleInput.Fill(optimizedTitle); err != nil {
				return uploader.StepResult{Step: uploader.StepFillTitle, Success: false, Error: fmt.Errorf("fill title failed: %w", err)}
			}
		}

		// 填写简介（正文）
		if description != "" {
			descInput := ctx.Page.Locator("div[contenteditable=true], textarea[placeholder*=\"正文\"], textarea[placeholder*=\"内容\"]").First()
			if err := descInput.Fill(description); err != nil {
				utils.Warn(fmt.Sprintf("[-] 填写简介失败（可能是富文本编辑器）: %v", err))
			}
		}

		// 添加话题标签
		for _, tag := range tags {
			if err := ctx.Page.Keyboard().Type("#" + tag + " "); err != nil {
				return uploader.StepResult{Step: uploader.StepAddTags, Success: false, Error: fmt.Errorf("type tag failed: %w", err)}
			}
			time.Sleep(500 * time.Millisecond)
		}

		return uploader.StepResult{Step: uploader.StepFillContent, Success: true}
	}
}

// StepWaitCoverGenerated 等待封面生成完成
func (u *Uploader) StepWaitCoverGenerated() uploader.StepFunc {
	return func(ctx *uploader.Context) uploader.StepResult {
		ctx.ReportProgress(uploader.StepSetCover, 55, "等待封面生成...")

		timeout := time.After(2 * time.Minute)
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-timeout:
				return uploader.StepResult{Step: uploader.StepSetCover, Success: false, Error: fmt.Errorf("cover generation timeout")}
			case <-ticker.C:
				// 参照Python版本检测封面生成
				count, _ := ctx.Page.Locator("div.cheetah-spin-container img").Count()
				if count > 0 {
					ctx.ReportProgress(uploader.StepSetCover, 60, "封面生成完成")
					return uploader.StepResult{Step: uploader.StepSetCover, Success: true}
				}
				utils.Info("[-] 等待封面生成...")
			}
		}
	}
}

// StepSetScheduleBaijiahao 百家号设置定时发布步骤
// 参照Python版本实现下拉选择方式
func (u *Uploader) StepSetScheduleBaijiahao(scheduleTime *string) uploader.StepFunc {
	return func(ctx *uploader.Context) uploader.StepResult {
		if scheduleTime == nil || *scheduleTime == "" {
			return uploader.StepResult{Step: uploader.StepSetSchedule, Success: true}
		}

		// 解析时间
		schedule, err := time.Parse("2006-01-02T15:04", *scheduleTime)
		if err != nil {
			return uploader.StepResult{Step: uploader.StepSetSchedule, Success: false, Error: fmt.Errorf("parse schedule time failed: %w", err)}
		}

		// 点击定时发布
		if err := ctx.Page.GetByText("定时发布").Click(); err != nil {
			return uploader.StepResult{Step: uploader.StepSetSchedule, Success: false, Error: fmt.Errorf("click schedule failed: %w", err)}
		}
		time.Sleep(2 * time.Second)

		// 参照Python版本：选择日期
		publishDateDay := fmt.Sprintf("%d月%d日", schedule.Month(), schedule.Day())
		if schedule.Day() < 10 {
			publishDateDay = fmt.Sprintf("%d月0%d日", schedule.Month(), schedule.Day())
		}

		// 点击日期选择框
		for retry := 0; retry < 3; retry++ {
			selectWrap := ctx.Page.Locator("div.select-wrap").Nth(0)
			if err := selectWrap.Click(); err != nil {
				if retry == 2 {
					return uploader.StepResult{Step: uploader.StepSetSchedule, Success: false, Error: fmt.Errorf("click date selector failed: %w", err)}
				}
				continue
			}
			time.Sleep(1 * time.Second)

			// 选择日期
			dateOption := ctx.Page.Locator(fmt.Sprintf("div.rc-virtual-list div.cheetah-select-item:has-text('%s')", publishDateDay))
			if err := dateOption.Click(); err != nil {
				if retry == 2 {
					utils.Warn(fmt.Sprintf("[-] 选择日期失败，使用默认日期: %v", err))
				} else {
					continue
				}
			}
			break
		}
		time.Sleep(1 * time.Second)

		// 参照Python版本：随机选择小时（因为Python版本注释说时间选择不准确，目前是随机）
		for retry := 0; retry < 3; retry++ {
			hourWrap := ctx.Page.Locator("div.select-wrap").Nth(1)
			if err := hourWrap.Click(); err != nil {
				if retry == 2 {
					return uploader.StepResult{Step: uploader.StepSetSchedule, Success: false, Error: fmt.Errorf("click hour selector failed: %w", err)}
				}
				continue
			}
			time.Sleep(1 * time.Second)

			// 获取可选小时数量
			hourOptions, _ := ctx.Page.Locator("div.rc-virtual-list:visible div.cheetah-select-item-option").Count()
			if hourOptions > 0 {
				// 随机选择一个（参照Python版本）
				randomHour := rand.Intn(hourOptions-3) + 1
				if randomHour >= hourOptions {
					randomHour = hourOptions - 1
				}
				hourOption := ctx.Page.Locator("div.rc-virtual-list:visible div.cheetah-select-item-option").Nth(randomHour)
				if err := hourOption.Click(); err != nil {
					utils.Warn(fmt.Sprintf("[-] 选择小时失败: %v", err))
				}
			}
			break
		}
		time.Sleep(1 * time.Second)

		// 点击确定按钮
		if err := ctx.Page.Locator("button:has-text('定时发布')").Click(); err != nil {
			utils.Warn(fmt.Sprintf("[-] 点击定时发布确认按钮失败: %v", err))
		}

		return uploader.StepResult{Step: uploader.StepSetSchedule, Success: true}
	}
}

// StepCheckSecurityVerification 检测百度安全验证
func (u *Uploader) StepCheckSecurityVerification() uploader.StepFunc {
	return func(ctx *uploader.Context) uploader.StepResult {
		time.Sleep(2 * time.Second)

		// 检测是否出现百度安全验证弹窗
		verifyLocator := ctx.Page.Locator("div.passMod_dialog-container:has-text('百度安全验证')")
		count, err := verifyLocator.Count()
		if err != nil {
			utils.Warn(fmt.Sprintf("[-] 检测安全验证时出错: %v", err))
			return uploader.StepResult{Step: uploader.StepPublish, Success: true}
		}

		if count > 0 {
			// 检查是否可见
			visible, _ := verifyLocator.IsVisible()
			if visible {
				utils.Error("[-] 出现百度安全验证，需要人工处理")
				return uploader.StepResult{Step: uploader.StepPublish, Success: false, Error: fmt.Errorf("百度安全验证出现，需要人工处理")}
			}
		}

		return uploader.StepResult{Step: uploader.StepPublish, Success: true}
	}
}

// StepClickPublishBaijiahao 百家号发布步骤（参照Python版本）
func (u *Uploader) StepClickPublishBaijiahao() uploader.StepFunc {
	return func(ctx *uploader.Context) uploader.StepResult {
		ctx.ReportProgress(uploader.StepPublish, 85, "正在发布...")

		// 点击发布按钮
		publishBtn := ctx.Page.Locator("button:has-text('发布')")
		if err := publishBtn.Click(); err != nil {
			return uploader.StepResult{Step: uploader.StepPublish, Success: false, Error: fmt.Errorf("click publish failed: %w", err)}
		}

		ctx.ReportProgress(uploader.StepPublish, 88, "等待发布结果...")

		// 等待发布成功（参照Python版本：等待跳转到clue页面）
		timeout := time.After(30 * time.Second)
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-timeout:
				ctx.TakeScreenshot("publish_timeout")
				return uploader.StepResult{Step: uploader.StepPublish, Success: false, Error: fmt.Errorf("publish timeout")}
			case <-ticker.C:
				// 检查是否跳转到线索页面（Python版本逻辑）
				url := ctx.Page.URL()
				if strings.Contains(url, "baijiahao.baidu.com/builder/rc/clue") {
					// 保存cookie（参照Python版本）
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

				// 检测安全验证
				verifyCount, _ := ctx.Page.Locator("div.passMod_dialog-container:has-text('百度安全验证')").Count()
				if verifyCount > 0 {
					visible, _ := ctx.Page.Locator("div.passMod_dialog-container:has-text('百度安全验证')").IsVisible()
					if visible {
						return uploader.StepResult{Step: uploader.StepPublish, Success: false, Error: fmt.Errorf("百度安全验证出现，需要人工处理")}
					}
				}

				// 截图记录发布状态
				ctx.TakeScreenshot("publishing")
			}
		}
	}
}

// Login 登录
// 参照Python版本：使用page.pause()等待用户完成登录，然后保存cookie
func (u *Uploader) Login() error {
	ctx := context.Background()

	// 创建新的浏览器上下文（不使用现有cookie）
	browserCtx, err := browserPool.GetContext(ctx, "", u.GetContextOptions())
	if err != nil {
		return fmt.Errorf("get browser context failed: %w", err)
	}
	// 注意：这里不defer Release，因为登录成功后需要保持上下文

	page, err := browserCtx.GetPage()
	if err != nil {
		browserCtx.Release()
		return fmt.Errorf("get page failed: %w", err)
	}

	// 参照Python版本：直接打开登录页面
	utils.Info("[-] 正在打开百家号登录页面...")
	if _, err := page.Goto("https://baijiahao.baidu.com/builder/theme/bjh/login", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	}); err != nil {
		browserCtx.Release()
		return fmt.Errorf("goto login page failed: %w", err)
	}

	utils.Info("[-] 请在浏览器窗口中完成登录")
	utils.Info("[-] 提示：如需要手机验证码，请保持手机畅通")
	utils.Info("[-] 登录完成后，请在开发者工具中点击继续（Resume）按钮")

	// 参照Python版本：使用page.pause()暂停，等待用户完成登录
	// 这样用户可以在浏览器中完成所有登录步骤（包括验证码、短信验证等）
	if err := page.Pause(); err != nil {
		utils.Warn(fmt.Sprintf("[-] page.pause()返回: %v", err))
	}

	// 用户点击继续后，检查是否登录成功
	utils.Info("[-] 正在检查登录状态...")
	time.Sleep(2 * time.Second)

	// 检查当前URL
	currentURL := page.URL()
	utils.Info(fmt.Sprintf("[-] 当前URL: %s", currentURL))

	// 检查是否已进入创作者中心
	if strings.Contains(currentURL, "baijiahao.baidu.com/builder") {
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
