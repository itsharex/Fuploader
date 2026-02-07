package douyin

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

// Uploader 抖音上传器
type Uploader struct {
	*uploader.Base
}

// NewUploader 创建上传器
func NewUploader(cookiePath string) *Uploader {
	initBrowserPool()
	return &Uploader{
		Base: uploader.NewBase("douyin", cookiePath, browserPool),
	}
}

// Platform 返回平台名称
func (u *Uploader) Platform() string {
	return "douyin"
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

	// 访问抖音创作者中心
	utils.Info("[-] 正在验证抖音登录状态...")
	if _, err := page.Goto("https://creator.douyin.com/"); err != nil {
		return false, fmt.Errorf("goto creator page failed: %w", err)
	}

	time.Sleep(3 * time.Second)

	// 检查当前URL
	currentURL := page.URL()
	utils.Info(fmt.Sprintf("[-] 当前URL: %s", currentURL))

	// 如果被重定向到登录页，说明Cookie无效
	if currentURL == "https://creator.douyin.com/login" ||
		currentURL == "https://www.douyin.com/login" {
		utils.Info("[-] 被重定向到登录页，Cookie 无效")
		return false, nil
	}

	// 检查是否有登录页特征
	phoneLoginCount, _ := page.GetByText("手机号登录").Count()
	qrLoginCount, _ := page.GetByText("扫码登录").Count()
	if phoneLoginCount > 0 || qrLoginCount > 0 {
		utils.Info("[-] 检测到登录页特征，Cookie 无效")
		return false, nil
	}

	// 检查是否有创作者中心特征元素（根据实际页面）
	// 左侧菜单
	homeMenu, _ := page.GetByText("首页").Count()
	contentManage, _ := page.GetByText("内容管理").Count()
	// 主要内容区
	publishVideo, _ := page.GetByText("发布视频").Count()

	if homeMenu > 0 || contentManage > 0 || publishVideo > 0 {
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
		u.StepNavigate("https://creator.douyin.com/creator-micro/content/upload"),

		// 2. 上传视频
		u.StepUploadVideo(
			"div[class^='container'] input",
			[]string{"[class^=\"long-card\"] div:has-text(\"重新上传\")"},
			[]string{"text=上传失败", ".upload-error"},
		),

		// 3. 填写标题
		u.StepFillTitleDouyin(task.Title),

		// 4. 填写简介
		u.StepFillDescription(task.Description),

		// 5. 添加标签
		u.StepAddTagsDouyin(task.Tags),

		// 6. 设置位置（改进版）
		u.StepSetLocationDouyin(task.Location),

		// 7. 设置第三方同步（头条/西瓜）
		u.StepSyncThirdParty(task.SyncToutiao, task.SyncXigua),

		// 8. 设置商品链接（如果有）
		u.StepSetProductLink(task.ProductLink, task.ProductTitle),

		// 9. 设置封面（改进版 - 优先使用自定义封面）
		u.StepSetCoverDouyin(task.Thumbnail),

		// 10. 处理自动封面选择（如果未设置自定义封面）
		u.StepHandleAutoCover(),

		// 11. 点击发布（带重试机制）
		u.StepClickPublishWithRetry(),
	}

	return u.Execute(ctx, task, steps)
}

// StepFillTitleDouyin 抖音填写标题步骤（参考Python项目实现）
func (u *Uploader) StepFillTitleDouyin(title string) uploader.StepFunc {
	return func(ctx *uploader.Context) uploader.StepResult {
		if title == "" {
			return uploader.StepResult{Step: uploader.StepFillTitle, Success: true}
		}

		// 限制标题长度（抖音限制30个字符）
		if len(title) > 30 {
			title = title[:30]
		}

		ctx.ReportProgress(uploader.StepFillTitle, 25, "正在填写标题...")

		// 方案1：使用相对定位（作品标题父级右侧第一个元素的input）
		titleContainer := ctx.Page.GetByText("作品标题").Locator("..").Locator("xpath=following-sibling::div[1]").Locator("input")
		count, _ := titleContainer.Count()

		if count > 0 {
			if err := titleContainer.Fill(title); err != nil {
				utils.Warn(fmt.Sprintf("[-] 使用相对定位填写标题失败: %v", err))
			} else {
				ctx.ReportProgress(uploader.StepFillTitle, 30, "标题填写完成")
				return uploader.StepResult{Step: uploader.StepFillTitle, Success: true}
			}
		}

		// 方案2：使用 .notranslate 类（参考Python项目）
		utils.Info("[-] 尝试使用备选方案填写标题...")
		titleInput := ctx.Page.Locator(".notranslate").First()
		if count, _ := titleInput.Count(); count > 0 {
			// 先清空输入框
			if err := titleInput.Click(); err != nil {
				return uploader.StepResult{Step: uploader.StepFillTitle, Success: false, Error: fmt.Errorf("click title input failed: %w", err)}
			}
			if err := ctx.Page.Keyboard().Press("Backspace"); err != nil {
				return uploader.StepResult{Step: uploader.StepFillTitle, Success: false, Error: fmt.Errorf("clear title failed: %w", err)}
			}
			if err := ctx.Page.Keyboard().Press("Control+KeyA"); err != nil {
				return uploader.StepResult{Step: uploader.StepFillTitle, Success: false, Error: fmt.Errorf("select all title failed: %w", err)}
			}
			if err := ctx.Page.Keyboard().Press("Delete"); err != nil {
				return uploader.StepResult{Step: uploader.StepFillTitle, Success: false, Error: fmt.Errorf("delete title failed: %w", err)}
			}
			// 输入标题
			if err := ctx.Page.Keyboard().Type(title); err != nil {
				return uploader.StepResult{Step: uploader.StepFillTitle, Success: false, Error: fmt.Errorf("type title failed: %w", err)}
			}
			if err := ctx.Page.Keyboard().Press("Enter"); err != nil {
				return uploader.StepResult{Step: uploader.StepFillTitle, Success: false, Error: fmt.Errorf("press enter failed: %w", err)}
			}

			ctx.ReportProgress(uploader.StepFillTitle, 30, "标题填写完成")
			return uploader.StepResult{Step: uploader.StepFillTitle, Success: true}
		}

		return uploader.StepResult{Step: uploader.StepFillTitle, Success: false, Error: fmt.Errorf("title input not found")}
	}
}

// StepFillDescription 填写简介步骤（参考Python项目，抖音描述和标题共用.notranslate）
func (u *Uploader) StepFillDescription(description string) uploader.StepFunc {
	return func(ctx *uploader.Context) uploader.StepResult {
		if description == "" {
			return uploader.StepResult{Step: uploader.StepFillContent, Success: true}
		}

		ctx.ReportProgress(uploader.StepFillContent, 31, "正在填写描述...")

		// 方案1：使用相对定位
		descContainer := ctx.Page.GetByText("作品描述").Locator("..").Locator("xpath=following-sibling::div[1]").Locator("textarea, .notranslate").First()
		count, _ := descContainer.Count()

		if count > 0 {
			if err := descContainer.Fill(description); err != nil {
				utils.Warn(fmt.Sprintf("[-] 使用相对定位填写描述失败: %v", err))
			} else {
				ctx.ReportProgress(uploader.StepFillContent, 35, "描述填写完成")
				return uploader.StepResult{Step: uploader.StepFillContent, Success: true}
			}
		}

		// 方案2：使用 .notranslate 类（参考Python项目，抖音描述和标题共用.notranslate）
		utils.Info("[-] 尝试使用备选方案填写描述...")
		// 获取第二个.notranslate（第一个是标题）
		descInput := ctx.Page.Locator(".notranslate").Nth(1)
		if count, _ := descInput.Count(); count > 0 {
			// 先清空输入框
			if err := descInput.Click(); err != nil {
				return uploader.StepResult{Step: uploader.StepFillContent, Success: false, Error: fmt.Errorf("click desc input failed: %w", err)}
			}
			if err := ctx.Page.Keyboard().Press("Control+KeyA"); err != nil {
				return uploader.StepResult{Step: uploader.StepFillContent, Success: false, Error: fmt.Errorf("select all desc failed: %w", err)}
			}
			if err := ctx.Page.Keyboard().Press("Delete"); err != nil {
				return uploader.StepResult{Step: uploader.StepFillContent, Success: false, Error: fmt.Errorf("delete desc failed: %w", err)}
			}
			// 输入描述
			if err := ctx.Page.Keyboard().Type(description); err != nil {
				return uploader.StepResult{Step: uploader.StepFillContent, Success: false, Error: fmt.Errorf("type desc failed: %w", err)}
			}

			ctx.ReportProgress(uploader.StepFillContent, 35, "描述填写完成")
			return uploader.StepResult{Step: uploader.StepFillContent, Success: true}
		}

		// 如果都失败，记录警告但不阻塞流程（描述不是必须的）
		utils.Warn("[-] 描述填写失败，继续执行")
		return uploader.StepResult{Step: uploader.StepFillContent, Success: true}
	}
}

// StepAddTagsDouyin 抖音添加标签步骤（参考Python项目）
func (u *Uploader) StepAddTagsDouyin(tags []string) uploader.StepFunc {
	return func(ctx *uploader.Context) uploader.StepResult {
		if len(tags) == 0 {
			return uploader.StepResult{Step: uploader.StepAddTags, Success: true}
		}

		ctx.ReportProgress(uploader.StepAddTags, 36, "正在添加标签...")

		// 方案1：使用 .zone-container（参考Python项目）
		cssSelector := ".zone-container"
		tagContainer := ctx.Page.Locator(cssSelector).First()
		count, _ := tagContainer.Count()

		if count > 0 {
			utils.Info(fmt.Sprintf("[-] 正在添加 %d 个标签...", len(tags)))
			for i, tag := range tags {
				// 输入标签（带#号）
				if err := tagContainer.Type("#"+tag, playwright.LocatorTypeOptions{Delay: playwright.Float(100)}); err != nil {
					utils.Warn(fmt.Sprintf("[-] 输入标签 #%s 失败: %v", tag, err))
					continue
				}
				// 按空格确认
				if err := tagContainer.Press("Space"); err != nil {
					utils.Warn(fmt.Sprintf("[-] 确认标签 #%s 失败: %v", tag, err))
					continue
				}
				// 等待标签渲染
				time.Sleep(300 * time.Millisecond)
				ctx.ReportProgress(uploader.StepAddTags, 36+int((float64(i+1)/float64(len(tags)))*4), fmt.Sprintf("已添加 %d/%d 个标签", i+1, len(tags)))
			}
			utils.Info(fmt.Sprintf("[-] 完成添加 %d 个标签", len(tags)))
			ctx.ReportProgress(uploader.StepAddTags, 40, "标签添加完成")
			return uploader.StepResult{Step: uploader.StepAddTags, Success: true}
		}

		// 方案2：尝试使用 "添加话题" 按钮
		utils.Info("[-] 尝试使用备选方案添加标签...")
		addTopicBtn := ctx.Page.GetByText("添加话题").First()
		if count, _ := addTopicBtn.Count(); count > 0 {
			for i, tag := range tags {
				if err := addTopicBtn.Click(); err != nil {
					utils.Warn(fmt.Sprintf("[-] 点击添加话题按钮失败: %v", err))
					continue
				}
				time.Sleep(500 * time.Millisecond)

				// 输入标签
				if err := ctx.Page.Keyboard().Type("#" + tag); err != nil {
					utils.Warn(fmt.Sprintf("[-] 输入标签 #%s 失败: %v", tag, err))
					continue
				}
				if err := ctx.Page.Keyboard().Press("Enter"); err != nil {
					utils.Warn(fmt.Sprintf("[-] 确认标签 #%s 失败: %v", tag, err))
					continue
				}
				time.Sleep(300 * time.Millisecond)
				ctx.ReportProgress(uploader.StepAddTags, 36+int((float64(i+1)/float64(len(tags)))*4), fmt.Sprintf("已添加 %d/%d 个标签", i+1, len(tags)))
			}
			utils.Info(fmt.Sprintf("[-] 完成添加 %d 个标签", len(tags)))
			ctx.ReportProgress(uploader.StepAddTags, 40, "标签添加完成")
			return uploader.StepResult{Step: uploader.StepAddTags, Success: true}
		}

		// 如果都失败，记录警告但不阻塞流程（标签不是必须的）
		utils.Warn("[-] 标签添加失败，继续执行")
		return uploader.StepResult{Step: uploader.StepAddTags, Success: true}
	}
}

// StepSetCoverDouyin 抖音设置封面步骤（改进版）
func (u *Uploader) StepSetCoverDouyin(thumbnailPath string) uploader.StepFunc {
	return func(ctx *uploader.Context) uploader.StepResult {
		// 如果没有自定义封面，跳过
		if thumbnailPath == "" {
			ctx.ReportProgress(uploader.StepSetCover, 55, "未设置自定义封面，将使用自动封面")
			return uploader.StepResult{Step: uploader.StepSetCover, Success: true}
		}

		ctx.ReportProgress(uploader.StepSetCover, 50, "正在设置自定义封面...")
		utils.Info("[-] 正在设置自定义封面...")

		// 点击"选择封面"
		coverBtn := ctx.Page.GetByText("选择封面").First()
		if count, _ := coverBtn.Count(); count == 0 {
			utils.Warn("[-] 未找到选择封面按钮")
			return uploader.StepResult{Step: uploader.StepSetCover, Success: true}
		}

		if err := coverBtn.Click(); err != nil {
			utils.Warn(fmt.Sprintf("[-] 点击选择封面失败: %v", err))
			return uploader.StepResult{Step: uploader.StepSetCover, Success: true}
		}
		time.Sleep(2 * time.Second)

		// 点击"设置竖封面"
		verticalCoverBtn := ctx.Page.GetByText("设置竖封面").First()
		if count, _ := verticalCoverBtn.Count(); count > 0 {
			if err := verticalCoverBtn.Click(); err != nil {
				utils.Warn(fmt.Sprintf("[-] 点击设置竖封面失败: %v", err))
			}
			time.Sleep(2 * time.Second)
		}

		// 上传封面文件
		coverInput := ctx.Page.Locator("div[class^='semi-upload upload'] >> input.semi-upload-hidden-input")
		if err := coverInput.SetInputFiles(thumbnailPath); err != nil {
			// 尝试其他选择器
			coverInput = ctx.Page.Locator("input[type='file']").First()
			if err := coverInput.SetInputFiles(thumbnailPath); err != nil {
				utils.Warn(fmt.Sprintf("[-] 上传封面失败: %v", err))
				// 关闭弹窗
				ctx.Page.Keyboard().Press("Escape")
				return uploader.StepResult{Step: uploader.StepSetCover, Success: true}
			}
		}

		utils.Info("[-] 封面文件已上传，等待处理...")
		time.Sleep(3 * time.Second)

		// 点击完成按钮
		finishBtn := ctx.Page.Locator("div#tooltip-container button:visible:has-text('完成'), div[class^='extractFooter'] button:visible:has-text('完成'), button:has-text('完成'):visible").First()
		if count, _ := finishBtn.Count(); count > 0 {
			if err := finishBtn.Click(); err != nil {
				utils.Warn(fmt.Sprintf("[-] 点击完成按钮失败: %v", err))
			}
		}

		// 等待封面设置对话框关闭
		time.Sleep(2 * time.Second)

		ctx.ReportProgress(uploader.StepSetCover, 60, "自定义封面设置完成")
		utils.Info("[-] 自定义封面设置完成")
		return uploader.StepResult{Step: uploader.StepSetCover, Success: true}
	}
}

// StepHandleAutoCover 处理自动封面选择（当未设置自定义封面时）
func (u *Uploader) StepHandleAutoCover() uploader.StepFunc {
	return func(ctx *uploader.Context) uploader.StepResult {
		// 检测"请设置封面后再发布"提示
		coverPrompt := ctx.Page.GetByText("请设置封面后再发布").First()
		visible, err := coverPrompt.IsVisible()
		if err != nil || !visible {
			return uploader.StepResult{Step: uploader.StepSetCover, Success: true}
		}

		ctx.ReportProgress(uploader.StepSetCover, 55, "检测到需要设置封面，自动选择推荐封面...")
		utils.Info("[-] 检测到需要设置封面提示...")

		// 选择第一个推荐封面
		recommendCover := ctx.Page.Locator("[class^='recommendCover-']").First()
		count, err := recommendCover.Count()
		if err != nil || count == 0 {
			ctx.TakeScreenshot("cover_not_found")
			return uploader.StepResult{Step: uploader.StepSetCover, Success: false, Error: fmt.Errorf("推荐封面未找到")}
		}

		if err := recommendCover.Click(); err != nil {
			return uploader.StepResult{Step: uploader.StepSetCover, Success: false, Error: fmt.Errorf("点击推荐封面失败: %w", err)}
		}
		time.Sleep(1 * time.Second)

		// 处理确认弹窗 "是否确认应用此封面？"
		confirmText := ctx.Page.GetByText("是否确认应用此封面？")
		if count, _ := confirmText.Count(); count > 0 {
			utils.Info("[-] 检测到确认弹窗，点击确定...")
			confirmBtn := ctx.Page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "确定"})
			if err := confirmBtn.Click(); err != nil {
				utils.Warn(fmt.Sprintf("[-] 点击确认按钮失败: %v", err))
			}
			time.Sleep(1 * time.Second)
		}

		ctx.ReportProgress(uploader.StepSetCover, 62, "封面选择完成")
		utils.Info("[-] 已完成封面选择流程")
		return uploader.StepResult{Step: uploader.StepSetCover, Success: true}
	}
}

// StepClickPublishWithRetry 抖音发布步骤（带重试机制，参考Python实现）
func (u *Uploader) StepClickPublishWithRetry() uploader.StepFunc {
	return func(ctx *uploader.Context) uploader.StepResult {
		ctx.ReportProgress(uploader.StepPublish, 85, "正在发布...")

		maxRetries := 20                 // 最多重试20次
		retryInterval := 5 * time.Second // 每次重试间隔5秒

		for retryCount := 0; retryCount < maxRetries; retryCount++ {
			// 检查页面是否已关闭
			if ctx.BrowserCtx != nil && ctx.BrowserCtx.IsPageClosed() {
				utils.Error("[-] 页面已被关闭，发布中断")
				return uploader.StepResult{Step: uploader.StepPublish, Success: false, Error: fmt.Errorf("page closed by user")}
			}

			// 尝试点击发布按钮
			publishBtn := ctx.Page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "发布", Exact: playwright.Bool(true)})
			if count, _ := publishBtn.Count(); count > 0 {
				if err := publishBtn.Click(); err != nil {
					utils.Warn(fmt.Sprintf("[-] 点击发布按钮失败: %v", err))
				}
			}

			// 等待一下
			time.Sleep(retryInterval)

			// 检查是否发布成功（跳转到管理页面）
			url := ctx.Page.URL()
			if strings.Contains(url, "creator.douyin.com/creator-micro/content/manage") {
				ctx.ReportProgress(uploader.StepPublish, 95, "发布成功")
				ctx.TakeScreenshot("publish_success")
				utils.Info("[-] 视频发布成功")
				return uploader.StepResult{Step: uploader.StepPublish, Success: true}
			}

			// 尝试处理封面问题
			coverPrompt := ctx.Page.GetByText("请设置封面后再发布").First()
			if visible, _ := coverPrompt.IsVisible(); visible {
				utils.Info("[-] 检测到需要设置封面，尝试自动选择...")
				// 调用自动封面选择
				recommendCover := ctx.Page.Locator("[class^='recommendCover-']").First()
				if count, _ := recommendCover.Count(); count > 0 {
					if err := recommendCover.Click(); err != nil {
						utils.Warn(fmt.Sprintf("[-] 点击推荐封面失败: %v", err))
					}
					time.Sleep(1 * time.Second)
					// 处理确认弹窗
					confirmBtn := ctx.Page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "确定"})
					if count, _ := confirmBtn.Count(); count > 0 {
						confirmBtn.Click()
						time.Sleep(1 * time.Second)
					}
				}
			}

			// 截图记录发布状态
			if retryCount%5 == 0 {
				ctx.TakeScreenshot(fmt.Sprintf("publishing_retry_%d", retryCount))
			}

			utils.Info(fmt.Sprintf("[-] 视频正在发布中... (尝试 %d/%d)", retryCount+1, maxRetries))
		}

		// 超过最大重试次数
		ctx.TakeScreenshot("publish_max_retry")
		return uploader.StepResult{Step: uploader.StepPublish, Success: false, Error: fmt.Errorf("发布失败，已重试%d次", maxRetries)}
	}
}

// StepSyncThirdParty 第三方平台同步（头条/西瓜）（中优先级改进）
func (u *Uploader) StepSyncThirdParty(syncToutiao, syncXigua bool) uploader.StepFunc {
	return func(ctx *uploader.Context) uploader.StepResult {
		if !syncToutiao && !syncXigua {
			return uploader.StepResult{Step: uploader.StepFillContent, Success: true}
		}

		ctx.ReportProgress(uploader.StepFillContent, 64, "正在设置第三方同步...")

		// 查找第三方同步开关
		thirdPartySwitch := ctx.Page.Locator("[class^='info'] > [class^='first-part'] div div.semi-switch")
		count, err := thirdPartySwitch.Count()
		if err != nil || count == 0 {
			utils.Info("[-] 未找到第三方同步开关，跳过")
			return uploader.StepResult{Step: uploader.StepFillContent, Success: true}
		}

		// 检查是否已选中
		className, _ := thirdPartySwitch.GetAttribute("class")
		if !strings.Contains(className, "semi-switch-checked") {
			// 点击开启同步
			switchInput := thirdPartySwitch.Locator("input.semi-switch-native-control")
			if err := switchInput.Click(); err != nil {
				utils.Warn(fmt.Sprintf("[-] 开启第三方同步失败: %v", err))
			} else {
				utils.Info("[-] 已开启第三方同步")
			}
		} else {
			utils.Info("[-] 第三方同步已开启")
		}

		ctx.ReportProgress(uploader.StepFillContent, 66, "第三方同步设置完成")
		return uploader.StepResult{Step: uploader.StepFillContent, Success: true}
	}
}

// StepSetProductLink 设置商品链接（中优先级改进）
func (u *Uploader) StepSetProductLink(productLink, productTitle string) uploader.StepFunc {
	return func(ctx *uploader.Context) uploader.StepResult {
		if productLink == "" {
			return uploader.StepResult{Step: uploader.StepFillContent, Success: true}
		}

		ctx.ReportProgress(uploader.StepFillContent, 67, "正在设置商品链接...")

		// 定位"添加标签"文本，然后向上导航到容器，再找到下拉框
		addTagText := ctx.Page.GetByText("添加标签")
		if count, _ := addTagText.Count(); count == 0 {
			utils.Info("[-] 未找到添加标签按钮，跳过商品链接设置")
			return uploader.StepResult{Step: uploader.StepFillContent, Success: true}
		}

		// 点击添加标签区域
		dropdown := addTagText.Locator("xpath=../../..").Locator(".semi-select").First()
		if err := dropdown.Click(); err != nil {
			utils.Warn(fmt.Sprintf("[-] 点击添加标签下拉框失败: %v", err))
			return uploader.StepResult{Step: uploader.StepFillContent, Success: true}
		}

		// 等待下拉选项出现
		shoppingCartOption := ctx.Page.Locator("[role='option']:has-text('购物车')")
		if err := shoppingCartOption.WaitFor(playwright.LocatorWaitForOptions{Timeout: playwright.Float(5000)}); err != nil {
			utils.Warn(fmt.Sprintf("[-] 等待购物车选项失败: %v", err))
			return uploader.StepResult{Step: uploader.StepFillContent, Success: true}
		}

		// 选择"购物车"选项
		if err := shoppingCartOption.Click(); err != nil {
			utils.Warn(fmt.Sprintf("[-] 选择购物车选项失败: %v", err))
			return uploader.StepResult{Step: uploader.StepFillContent, Success: true}
		}

		// 等待商品链接输入框出现
		linkInput := ctx.Page.Locator("input[placeholder='粘贴商品链接']")
		if err := linkInput.WaitFor(playwright.LocatorWaitForOptions{Timeout: playwright.Float(5000)}); err != nil {
			utils.Warn(fmt.Sprintf("[-] 等待商品链接输入框失败: %v", err))
			return uploader.StepResult{Step: uploader.StepFillContent, Success: true}
		}

		// 输入商品链接
		if err := linkInput.Fill(productLink); err != nil {
			utils.Warn(fmt.Sprintf("[-] 填写商品链接失败: %v", err))
			return uploader.StepResult{Step: uploader.StepFillContent, Success: true}
		}

		// 点击"添加链接"按钮
		addButton := ctx.Page.GetByText("添加链接")
		buttonClass, _ := addButton.GetAttribute("class")
		if strings.Contains(buttonClass, "disable") {
			utils.Warn("[-] 添加链接按钮不可用，可能是链接格式不正确")
			return uploader.StepResult{Step: uploader.StepFillContent, Success: true}
		}

		if err := addButton.Click(); err != nil {
			utils.Warn(fmt.Sprintf("[-] 点击添加链接按钮失败: %v", err))
			return uploader.StepResult{Step: uploader.StepFillContent, Success: true}
		}

		// 处理商品短标题弹窗（如果有）
		if productTitle != "" {
			u.handleProductDialog(ctx, productTitle)
		}

		ctx.ReportProgress(uploader.StepFillContent, 70, "商品链接设置完成")
		return uploader.StepResult{Step: uploader.StepFillContent, Success: true}
	}
}

// handleProductDialog 处理商品短标题弹窗
func (u *Uploader) handleProductDialog(ctx *uploader.Context, productTitle string) {
	// 等待短标题输入框出现
	titleInput := ctx.Page.Locator("input[placeholder*='短标题'], input[placeholder*='标题']").First()
	if err := titleInput.WaitFor(playwright.LocatorWaitForOptions{Timeout: playwright.Float(3000)}); err != nil {
		utils.Warn("[-] 未找到商品短标题输入框")
		return
	}

	if err := titleInput.Fill(productTitle); err != nil {
		utils.Warn(fmt.Sprintf("[-] 填写商品短标题失败: %v", err))
		return
	}

	// 点击确定按钮
	confirmBtn := ctx.Page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "确定"})
	if err := confirmBtn.Click(); err != nil {
		utils.Warn(fmt.Sprintf("[-] 点击确定按钮失败: %v", err))
	}
}

// StepSetLocationDouyin 抖音位置设置（低优先级改进）
func (u *Uploader) StepSetLocationDouyin(location string) uploader.StepFunc {
	return func(ctx *uploader.Context) uploader.StepResult {
		if location == "" {
			return uploader.StepResult{Step: uploader.StepFillContent, Success: true}
		}

		ctx.ReportProgress(uploader.StepFillContent, 71, "正在设置位置...")

		// 定位位置输入框
		locationInput := ctx.Page.Locator("div.semi-select span:has-text('输入地理位置')")
		if count, _ := locationInput.Count(); count == 0 {
			utils.Info("[-] 未找到位置输入框，跳过位置设置")
			return uploader.StepResult{Step: uploader.StepFillContent, Success: true}
		}

		if err := locationInput.Click(); err != nil {
			utils.Warn(fmt.Sprintf("[-] 点击位置输入框失败: %v", err))
			return uploader.StepResult{Step: uploader.StepFillContent, Success: true}
		}

		// 清空原有内容
		if err := ctx.Page.Keyboard().Press("Backspace"); err != nil {
			utils.Warn(fmt.Sprintf("[-] 清空位置输入框失败: %v", err))
		}

		time.Sleep(2 * time.Second)

		// 输入位置
		if err := ctx.Page.Keyboard().Type(location); err != nil {
			utils.Warn(fmt.Sprintf("[-] 输入位置失败: %v", err))
			return uploader.StepResult{Step: uploader.StepFillContent, Success: true}
		}

		time.Sleep(2 * time.Second)

		// 选择第一个选项
		firstOption := ctx.Page.Locator("div[role='listbox'] [role='option']").First()
		if err := firstOption.Click(); err != nil {
			utils.Warn(fmt.Sprintf("[-] 选择位置选项失败: %v", err))
			return uploader.StepResult{Step: uploader.StepFillContent, Success: true}
		}

		ctx.ReportProgress(uploader.StepFillContent, 73, "位置设置完成")
		return uploader.StepResult{Step: uploader.StepFillContent, Success: true}
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

	// 访问抖音创作者中心
	utils.Info("[-] 正在打开抖音创作者中心...")
	if _, err := page.Goto("https://creator.douyin.com/", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	}); err != nil {
		return fmt.Errorf("goto creator center failed: %w", err)
	}

	// 等待页面完全加载
	if err := browserCtx.WaitForPageLoad(); err != nil {
		utils.Warn(fmt.Sprintf("[-] 等待页面加载警告: %v", err))
	}
	time.Sleep(3 * time.Second)

	// 等待用户登录
	utils.Info("[-] 请在浏览器窗口中完成登录...")

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

			// 检查是否已进入创作者中心
			url := page.URL()
			utils.Info(fmt.Sprintf("[-] 当前URL: %s", url))

			// 使用 strings.Contains 匹配URL
			if strings.Contains(url, "creator.douyin.com") {
				// 等待一下确保页面元素加载完成
				time.Sleep(1 * time.Second)

				// 检查创作者中心特征元素（根据实际页面）
				homeMenu, _ := page.GetByText("首页").Count()
				contentManage, _ := page.GetByText("内容管理").Count()
				publishVideo, _ := page.GetByText("发布视频").Count()

				if homeMenu > 0 || contentManage > 0 || publishVideo > 0 {
					utils.Info("[-] 登录成功，已进入创作者中心")
					return nil
				}
			}
		}
	}
}
