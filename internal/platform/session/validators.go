package session

import (
	"context"
	"fmt"
	"time"

	"Fuploader/internal/platform/browser"
	"Fuploader/internal/types"
	"Fuploader/internal/utils"

	"github.com/playwright-community/playwright-go"
)

// ValidationOptions 验证选项
type ValidationOptions struct {
	Timeout       time.Duration
	RetryCount    int
	RetryInterval time.Duration
	CheckAsync    bool // 是否检查异步加载内容
}

// DefaultValidationOptions 默认验证选项
func DefaultValidationOptions() ValidationOptions {
	return ValidationOptions{
		Timeout:       10 * time.Second,
		RetryCount:    3,
		RetryInterval: 2 * time.Second,
		CheckAsync:    true,
	}
}

// BaseValidator 基础验证器
type BaseValidator struct {
	platform string
	pool     *browser.Pool
	options  ValidationOptions
}

// Platform 返回平台名称
func (v *BaseValidator) Platform() string {
	return v.platform
}

// SetOptions 设置验证选项
func (v *BaseValidator) SetOptions(options ValidationOptions) {
	v.options = options
}

// validateWithRetry 带重试的验证
func (v *BaseValidator) validateWithRetry(ctx context.Context, session *Session, validateFunc func(playwright.Page) (bool, error)) (bool, error) {
	for i := 0; i <= v.options.RetryCount; i++ {
		if i > 0 {
			utils.Info(fmt.Sprintf("[-] %s 验证失败，第%d次重试...", v.platform, i))
			time.Sleep(v.options.RetryInterval)
		}

		// 使用浏览器池获取上下文
		browserCtx, err := v.pool.GetContext(ctx, v.getCookiePath(session), v.getContextOptions())
		if err != nil {
			if i == v.options.RetryCount {
				return false, types.NewNetworkError("validate", fmt.Errorf("get browser context failed: %w", err))
			}
			continue
		}
		defer browserCtx.Release()

		page, err := browserCtx.GetPage()
		if err != nil {
			if i == v.options.RetryCount {
				return false, types.NewNetworkError("validate", fmt.Errorf("get page failed: %w", err))
			}
			continue
		}

		valid, err := validateFunc(page)
		if err != nil {
			if i == v.options.RetryCount {
				return false, err
			}
			continue
		}

		return valid, nil
	}

	return false, nil
}

func (v *BaseValidator) getCookiePath(session *Session) string {
	return fmt.Sprintf("cookies/%s_%d.json", session.Platform, session.AccountID)
}

func (v *BaseValidator) getContextOptions() *browser.ContextOptions {
	return &browser.ContextOptions{
		UserAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		Viewport:    &playwright.Size{Width: 1920, Height: 1080},
		Locale:      "zh-CN",
		TimezoneId:  "Asia/Shanghai",
		Geolocation: &playwright.Geolocation{Latitude: 39.9042, Longitude: 116.4074},
		ExtraHeaders: map[string]string{
			"Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8",
		},
	}
}

// XiaoHongShuValidator 小红书验证器
type XiaoHongShuValidator struct {
	BaseValidator
}

// NewXiaoHongShuValidator 创建小红书验证器
func NewXiaoHongShuValidator(pool *browser.Pool) *XiaoHongShuValidator {
	return &XiaoHongShuValidator{
		BaseValidator: BaseValidator{
			platform: "xiaohongshu",
			pool:     pool,
			options:  DefaultValidationOptions(),
		},
	}
}

// Validate 验证小红书会话（参考 Python 实现）
func (v *XiaoHongShuValidator) Validate(ctx context.Context, session *Session) (bool, error) {
	return v.validateWithRetry(ctx, session, func(page playwright.Page) (bool, error) {
		// 访问发布页面验证登录状态
		utils.Info("[-] 正在验证小红书登录状态...")
		if _, err := page.Goto("https://creator.xiaohongshu.com/creator-micro/content/upload",
			playwright.PageGotoOptions{
				WaitUntil: playwright.WaitUntilStateNetworkidle,
				Timeout:   playwright.Float(10000),
			}); err != nil {
			return false, types.NewTimeoutError("validate", err)
		}

		// 等待页面加载
		time.Sleep(2 * time.Second)

		// 检查当前URL
		currentURL := page.URL()
		utils.Info(fmt.Sprintf("[-] 小红书验证 - 当前URL: %s", currentURL))

		// 如果被重定向到登录页，说明Cookie无效
		if currentURL == "https://creator.xiaohongshu.com/login" ||
			currentURL == "https://www.xiaohongshu.com/login" ||
			currentURL == "https://creator.xiaohongshu.com/login" {
			return false, nil
		}

		// 检查是否存在登录按钮（参考 Python 实现）
		phoneLoginCount, _ := page.GetByText("手机号登录").Count()
		qrLoginCount, _ := page.GetByText("扫码登录").Count()
		if phoneLoginCount > 0 || qrLoginCount > 0 {
			utils.Info("[-] 小红书检测到登录按钮，Cookie 已失效")
			return false, nil
		}

		// 检查是否有创作者中心特征元素
		publishBtn, _ := page.GetByText("发布笔记").Count()
		contentManage, _ := page.GetByText("内容管理").Count()
		dataCenter, _ := page.GetByText("数据中心").Count()

		if publishBtn > 0 || contentManage > 0 || dataCenter > 0 {
			utils.Info("[-] 小红书 Cookie 验证通过")
			return true, nil
		}

		return false, nil
	})
}

// DouyinValidator 抖音验证器
type DouyinValidator struct {
	BaseValidator
}

// NewDouyinValidator 创建抖音验证器
func NewDouyinValidator(pool *browser.Pool) *DouyinValidator {
	return &DouyinValidator{
		BaseValidator: BaseValidator{
			platform: "douyin",
			pool:     pool,
			options:  DefaultValidationOptions(),
		},
	}
}

// Validate 验证抖音会话（参考 Python 实现）
func (v *DouyinValidator) Validate(ctx context.Context, session *Session) (bool, error) {
	return v.validateWithRetry(ctx, session, func(page playwright.Page) (bool, error) {
		// 访问上传页面验证登录状态（参考 Python 实现）
		utils.Info("[-] 正在验证抖音登录状态...")
		if _, err := page.Goto("https://creator.douyin.com/creator-micro/content/upload",
			playwright.PageGotoOptions{
				WaitUntil: playwright.WaitUntilStateNetworkidle,
				Timeout:   playwright.Float(10000),
			}); err != nil {
			return false, types.NewTimeoutError("validate", err)
		}

		// 等待页面加载
		time.Sleep(2 * time.Second)

		// 检查当前URL
		currentURL := page.URL()
		utils.Info(fmt.Sprintf("[-] 抖音验证 - 当前URL: %s", currentURL))

		// 如果被重定向到登录页
		if currentURL == "https://creator.douyin.com/login" ||
			currentURL == "https://www.douyin.com/login" {
			return false, nil
		}

		// 检查是否存在登录按钮（参考 Python 实现）
		phoneLoginCount, _ := page.GetByText("手机号登录").Count()
		qrLoginCount, _ := page.GetByText("扫码登录").Count()
		if phoneLoginCount > 0 || qrLoginCount > 0 {
			utils.Info("[-] 抖音检测到登录按钮，Cookie 已失效")
			return false, nil
		}

		// 检查是否有创作者中心特征元素
		uploadBtn, _ := page.GetByText("上传视频").Count()
		contentManage, _ := page.GetByText("内容管理").Count()
		dataStats, _ := page.GetByText("作品数据").Count()

		if uploadBtn > 0 || contentManage > 0 || dataStats > 0 {
			utils.Info("[-] 抖音 Cookie 验证通过")
			return true, nil
		}

		return false, nil
	})
}

// KuaishouValidator 快手验证器
type KuaishouValidator struct {
	BaseValidator
}

// NewKuaishouValidator 创建快手验证器
func NewKuaishouValidator(pool *browser.Pool) *KuaishouValidator {
	return &KuaishouValidator{
		BaseValidator: BaseValidator{
			platform: "kuaishou",
			pool:     pool,
			options:  DefaultValidationOptions(),
		},
	}
}

// Validate 验证快手会话（参考 Python 实现）
func (v *KuaishouValidator) Validate(ctx context.Context, session *Session) (bool, error) {
	return v.validateWithRetry(ctx, session, func(page playwright.Page) (bool, error) {
		// 访问发布页面验证登录状态（参考 Python 实现）
		utils.Info("[-] 正在验证快手登录状态...")
		if _, err := page.Goto("https://cp.kuaishou.com/article/publish/video",
			playwright.PageGotoOptions{
				WaitUntil: playwright.WaitUntilStateNetworkidle,
				Timeout:   playwright.Float(10000),
			}); err != nil {
			return false, types.NewTimeoutError("validate", err)
		}

		// 等待页面加载
		time.Sleep(2 * time.Second)

		// 检查当前URL
		currentURL := page.URL()
		utils.Info(fmt.Sprintf("[-] 快手验证 - 当前URL: %s", currentURL))

		// 如果被重定向到登录页
		if currentURL == "https://cp.kuaishou.com/login" ||
			currentURL == "https://passport.kuaishou.com/" {
			return false, nil
		}

		// 检查"机构服务"元素（参考 Python 实现）
		orgService, _ := page.Locator("div.names div.container div.name:text('机构服务')").Count()
		if orgService > 0 {
			utils.Info("[-] 快手检测到'机构服务'元素，Cookie 已失效")
			return false, nil
		}

		// 检查是否有创作者中心特征元素
		publishBtn, _ := page.GetByText("发布视频").Count()
		contentManage, _ := page.GetByText("内容管理").Count()
		dataCenter, _ := page.GetByText("数据中心").Count()

		if publishBtn > 0 || contentManage > 0 || dataCenter > 0 {
			utils.Info("[-] 快手 Cookie 验证通过")
			return true, nil
		}

		return false, nil
	})
}

// TencentValidator 视频号验证器
type TencentValidator struct {
	BaseValidator
}

// NewTencentValidator 创建视频号验证器
func NewTencentValidator(pool *browser.Pool) *TencentValidator {
	return &TencentValidator{
		BaseValidator: BaseValidator{
			platform: "tencent",
			pool:     pool,
			options:  DefaultValidationOptions(),
		},
	}
}

// Validate 验证视频号会话（参考 Python 实现）
func (v *TencentValidator) Validate(ctx context.Context, session *Session) (bool, error) {
	return v.validateWithRetry(ctx, session, func(page playwright.Page) (bool, error) {
		// 访问发布页面验证登录状态（参考 Python 实现）
		utils.Info("[-] 正在验证视频号登录状态...")
		if _, err := page.Goto("https://channels.weixin.qq.com/platform/post/create",
			playwright.PageGotoOptions{
				WaitUntil: playwright.WaitUntilStateNetworkidle,
				Timeout:   playwright.Float(10000),
			}); err != nil {
			return false, types.NewTimeoutError("validate", err)
		}

		// 等待页面加载
		time.Sleep(2 * time.Second)

		// 检查当前URL
		currentURL := page.URL()
		utils.Info(fmt.Sprintf("[-] 视频号验证 - 当前URL: %s", currentURL))

		// 检查"微信小店"元素（参考 Python 实现）
		shopElement, _ := page.Locator("div.title-name:has-text('微信小店')").Count()
		if shopElement > 0 {
			utils.Info("[-] 视频号检测到'微信小店'元素，Cookie 已失效")
			return false, nil
		}

		// 检查是否有创作者中心特征元素
		publishBtn, _ := page.GetByText("发表视频").Count()
		contentManage, _ := page.GetByText("内容管理").Count()
		dataCenter, _ := page.GetByText("数据中心").Count()

		if publishBtn > 0 || contentManage > 0 || dataCenter > 0 {
			utils.Info("[-] 视频号 Cookie 验证通过")
			return true, nil
		}

		return false, nil
	})
}

// TikTokValidator TikTok验证器
type TikTokValidator struct {
	BaseValidator
}

// NewTikTokValidator 创建TikTok验证器
func NewTikTokValidator(pool *browser.Pool) *TikTokValidator {
	return &TikTokValidator{
		BaseValidator: BaseValidator{
			platform: "tiktok",
			pool:     pool,
			options:  DefaultValidationOptions(),
		},
	}
}

// Validate 验证TikTok会话（参考 Python 实现）
func (v *TikTokValidator) Validate(ctx context.Context, session *Session) (bool, error) {
	return v.validateWithRetry(ctx, session, func(page playwright.Page) (bool, error) {
		// 访问 TikTok Studio（参考 Python 实现）
		utils.Info("[-] 正在验证 TikTok 登录状态...")
		if _, err := page.Goto("https://www.tiktok.com/tiktokstudio/upload",
			playwright.PageGotoOptions{
				WaitUntil: playwright.WaitUntilStateNetworkidle,
				Timeout:   playwright.Float(10000),
			}); err != nil {
			return false, types.NewTimeoutError("validate", err)
		}

		// 等待页面加载
		time.Sleep(2 * time.Second)

		// 检查当前URL
		currentURL := page.URL()
		utils.Info(fmt.Sprintf("[-] TikTok 验证 - 当前URL: %s", currentURL))

		// 如果被重定向到登录页
		if currentURL == "https://www.tiktok.com/login" ||
			currentURL == "https://www.tiktok.com/login/" {
			return false, nil
		}

		// 检查 Select 元素 class（参考 Python 实现）
		selectElements, _ := page.QuerySelectorAll("select")
		for _, element := range selectElements {
			className, _ := element.GetAttribute("class")
			if className != "" && len(className) > 7 && className[:7] == "tiktok-" {
				utils.Info("[-] TikTok 检测到未登录特征元素")
				return false, nil
			}
		}

		// 检查是否有创作者中心特征元素
		uploadBtn, _ := page.GetByText("Upload").Count()
		dashboardMenu, _ := page.GetByText("Dashboard").Count()
		contentMenu, _ := page.GetByText("Content").Count()

		if uploadBtn > 0 || dashboardMenu > 0 || contentMenu > 0 {
			utils.Info("[-] TikTok Cookie 验证通过")
			return true, nil
		}

		return false, nil
	})
}

func (v *TikTokValidator) getContextOptions() *browser.ContextOptions {
	return &browser.ContextOptions{
		UserAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		Viewport:    &playwright.Size{Width: 1920, Height: 1080},
		Locale:      "en-US",
		TimezoneId:  "America/New_York",
		Geolocation: &playwright.Geolocation{Latitude: 40.7128, Longitude: -74.0060},
		ExtraHeaders: map[string]string{
			"Accept-Language": "en-US,en;q=0.9",
		},
	}
}

// BaijiahaoValidator 百家号验证器
type BaijiahaoValidator struct {
	BaseValidator
}

// NewBaijiahaoValidator 创建百家号验证器
func NewBaijiahaoValidator(pool *browser.Pool) *BaijiahaoValidator {
	return &BaijiahaoValidator{
		BaseValidator: BaseValidator{
			platform: "baijiahao",
			pool:     pool,
			options:  DefaultValidationOptions(),
		},
	}
}

// Validate 验证百家号会话（参考 Python 实现）
func (v *BaijiahaoValidator) Validate(ctx context.Context, session *Session) (bool, error) {
	return v.validateWithRetry(ctx, session, func(page playwright.Page) (bool, error) {
		// 访问百家号首页（参考 Python 实现）
		utils.Info("[-] 正在验证百家号登录状态...")
		if _, err := page.Goto("https://baijiahao.baidu.com/builder/rc/home",
			playwright.PageGotoOptions{
				WaitUntil: playwright.WaitUntilStateNetworkidle,
				Timeout:   playwright.Float(10000),
			}); err != nil {
			return false, types.NewTimeoutError("validate", err)
		}

		// 等待页面加载
		time.Sleep(2 * time.Second)

		// 检查当前URL
		currentURL := page.URL()
		utils.Info(fmt.Sprintf("[-] 百家号验证 - 当前URL: %s", currentURL))

		// 如果被重定向到登录页
		if currentURL == "https://baijiahao.baidu.com/builder/theme/bjh/login" ||
			currentURL == "https://passport.baidu.com/" {
			return false, nil
		}

		// 检查"注册/登录百家号"按钮（参考 Python 实现）
		loginBtn, _ := page.GetByText("注册/登录百家号").Count()
		if loginBtn > 0 {
			utils.Info("[-] 百家号检测到登录按钮，Cookie 已失效")
			return false, nil
		}

		// 检查是否有创作者中心特征元素
		publishBtn, _ := page.GetByText("发布").Count()
		contentManage, _ := page.GetByText("内容管理").Count()
		homeMenu, _ := page.GetByText("首页").Count()

		if publishBtn > 0 || contentManage > 0 || homeMenu > 0 {
			utils.Info("[-] 百家号 Cookie 验证通过")
			return true, nil
		}

		return false, nil
	})
}

// RegisterAllValidators 注册所有平台验证器
func RegisterAllValidators(manager *Manager, pool *browser.Pool) {
	manager.RegisterValidator(NewXiaoHongShuValidator(pool))
	manager.RegisterValidator(NewDouyinValidator(pool))
	manager.RegisterValidator(NewKuaishouValidator(pool))
	manager.RegisterValidator(NewTencentValidator(pool))
	manager.RegisterValidator(NewTikTokValidator(pool))
	manager.RegisterValidator(NewBaijiahaoValidator(pool))
}
