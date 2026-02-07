package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"Fuploader/internal/platform/platformutils"
	"Fuploader/internal/utils"

	"github.com/playwright-community/playwright-go"
)

// PoolStats 浏览器池统计信息
type PoolStats struct {
	BrowserCount      int       `json:"browser_count"`        // 当前浏览器实例数
	ContextCount      int       `json:"context_count"`        // 当前上下文总数
	IdleContextCount  int       `json:"idle_context_count"`   // 空闲上下文数
	InUseContextCount int       `json:"in_use_context_count"` // 使用中上下文数
	WaitQueueLength   int       `json:"wait_queue_length"`    // 等待队列长度
	MaxBrowsers       int       `json:"max_browsers"`         // 最大浏览器数
	MaxContexts       int       `json:"max_contexts"`         // 每个浏览器的最大上下文数
	Timestamp         time.Time `json:"timestamp"`            // 统计时间戳
}

// Pool 浏览器池
type Pool struct {
	maxBrowsers int
	maxContexts int
	browsers    []*PooledBrowser
	mutex       sync.RWMutex
	waitQueue   chan struct{} // 等待队列，用于限制并发获取上下文
	stats       PoolStats
	statsMutex  sync.RWMutex
}

// PooledBrowser 池化浏览器
type PooledBrowser struct {
	browser  playwright.Browser
	contexts []*PooledContext
	lastUsed time.Time
	inUse    int
	mutex    sync.Mutex
}

// PooledContext 封装的浏览器上下文
type PooledContext struct {
	context    playwright.BrowserContext
	page       playwright.Page
	cookiePath string
	createdAt  time.Time
	lastUsed   time.Time
	parent     *PooledBrowser
}

// ContextOptions 上下文选项
type ContextOptions struct {
	UserAgent    string
	Viewport     *playwright.Size
	Locale       string
	TimezoneId   string
	Geolocation  *playwright.Geolocation
	ExtraHeaders map[string]string
}

// NewPool 创建浏览器池
func NewPool(maxBrowsers, maxContexts int) *Pool {
	return &Pool{
		maxBrowsers: maxBrowsers,
		maxContexts: maxContexts,
		browsers:    make([]*PooledBrowser, 0),
		waitQueue:   make(chan struct{}, maxBrowsers*maxContexts), // 限制并发数
	}
}

// NewPoolFromConfig 从配置创建浏览器池
func NewPoolFromConfig() *Pool {
	config := LoadPoolConfig()
	return NewPool(config.MaxBrowsers, config.MaxContextsPerBrowser)
}

// GetContext 获取浏览器上下文
func (p *Pool) GetContext(ctx context.Context, cookiePath string, options *ContextOptions) (*PooledContext, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// 1. 尝试复用现有上下文
	for _, browser := range p.browsers {
		if pooledCtx := browser.getIdleContext(cookiePath); pooledCtx != nil {
			p.updateStats()
			return pooledCtx, nil
		}
	}

	// 2. 创建新上下文
	browser, err := p.getOrCreateBrowser()
	if err != nil {
		return nil, err
	}

	pooledCtx, err := browser.createContext(cookiePath, options)
	if err != nil {
		return nil, err
	}

	p.updateStats()
	return pooledCtx, nil
}

// GetStats 获取浏览器池统计信息
func (p *Pool) GetStats() PoolStats {
	p.statsMutex.RLock()
	defer p.statsMutex.RUnlock()
	return p.stats
}

// updateStats 更新统计信息
func (p *Pool) updateStats() {
	p.statsMutex.Lock()
	defer p.statsMutex.Unlock()

	p.stats = PoolStats{
		BrowserCount: len(p.browsers),
		MaxBrowsers:  p.maxBrowsers,
		MaxContexts:  p.maxContexts,
		Timestamp:    time.Now(),
	}

	for _, browser := range p.browsers {
		browser.mutex.Lock()
		p.stats.ContextCount += len(browser.contexts)
		p.stats.InUseContextCount += browser.inUse
		for _, ctx := range browser.contexts {
			if time.Since(ctx.lastUsed) > 30*time.Second {
				p.stats.IdleContextCount++
			}
		}
		browser.mutex.Unlock()
	}

	p.stats.WaitQueueLength = len(p.waitQueue)
}

// Close 关闭浏览器池
func (p *Pool) Close() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	for _, browser := range p.browsers {
		for _, ctx := range browser.contexts {
			ctx.Close()
		}
		if err := browser.browser.Close(); err != nil {
			utils.Warn(fmt.Sprintf("[-] 关闭浏览器失败: %v", err))
		}
	}

	p.browsers = make([]*PooledBrowser, 0)
	p.updateStats()
	return nil
}

// getOrCreateBrowser 获取或创建浏览器实例
func (p *Pool) getOrCreateBrowser() (*PooledBrowser, error) {
	// 查找有可用容量的浏览器
	for _, b := range p.browsers {
		if b.canCreateContext(p.maxContexts) {
			return b, nil
		}
	}

	// 创建新浏览器
	if len(p.browsers) < p.maxBrowsers {
		browser, err := p.launchBrowser()
		if err != nil {
			return nil, err
		}

		pooled := &PooledBrowser{
			browser:  browser,
			contexts: make([]*PooledContext, 0),
		}
		p.browsers = append(p.browsers, pooled)
		return pooled, nil
	}

	return nil, fmt.Errorf("max browsers reached")
}

// launchBrowser 启动浏览器
func (p *Pool) launchBrowser() (playwright.Browser, error) {
	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("start playwright failed: %w", err)
	}

	// 查找本地 Chrome
	chromePath := findLocalChrome()

	launchOptions := playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(false),
		Args: []string{
			"--disable-blink-features=AutomationControlled",
			"--disable-web-security",
			"--no-sandbox",
			"--disable-setuid-sandbox",
			"--disable-dev-shm-usage",
			"--window-size=1920,1080",
			"--window-position=0,0",
			"--start-maximized",
			"--disable-infobars",
			"--disable-extensions",
			"--disable-default-apps",
			"--disable-background-networking",
			"--disable-sync",
			"--disable-translate",
			"--disable-popup-blocking",
			"--disable-features=IsolateOrigins,site-per-process",
			"--disable-site-isolation-trials",
		},
	}

	if chromePath != "" {
		launchOptions.ExecutablePath = playwright.String(chromePath)
		utils.Info("[-] 浏览器池使用本地 Chrome")
	}

	browser, err := pw.Chromium.Launch(launchOptions)
	if err != nil {
		return nil, fmt.Errorf("launch browser failed: %w", err)
	}

	return browser, nil
}

// canCreateContext 检查是否可以创建新上下文
func (b *PooledBrowser) canCreateContext(maxContexts int) bool {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return len(b.contexts) < maxContexts
}

// getIdleContext 获取空闲上下文
func (b *PooledBrowser) getIdleContext(cookiePath string) *PooledContext {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	for _, ctx := range b.contexts {
		if ctx.cookiePath == cookiePath && time.Since(ctx.lastUsed) > 30*time.Second {
			ctx.lastUsed = time.Now()
			b.inUse++
			return ctx
		}
	}
	return nil
}

// createContext 创建浏览器上下文
func (b *PooledBrowser) createContext(cookiePath string, options *ContextOptions) (*PooledContext, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	contextOptions := playwright.BrowserNewContextOptions{
		Locale:           playwright.String(options.Locale),
		TimezoneId:       playwright.String(options.TimezoneId),
		Permissions:      []string{"geolocation"},
		ColorScheme:      playwright.ColorSchemeLight,
		ExtraHttpHeaders: options.ExtraHeaders,
		// 不设置 Viewport，让浏览器使用 --window-size 参数决定窗口大小
		// 这样可以避免 Playwright 视口设置与 Chrome 窗口大小冲突
	}

	if options.UserAgent != "" {
		contextOptions.UserAgent = playwright.String(options.UserAgent)
	}
	if options.Geolocation != nil {
		contextOptions.Geolocation = options.Geolocation
	}

	// 加载 Cookie
	if _, err := os.Stat(cookiePath); err == nil {
		contextOptions.StorageStatePath = playwright.String(cookiePath)
	}

	context, err := b.browser.NewContext(contextOptions)
	if err != nil {
		return nil, fmt.Errorf("create context failed: %w", err)
	}

	// 注入反检测脚本
	if err := platformutils.InjectStealthScript(context); err != nil {
		return nil, fmt.Errorf("inject stealth script failed: %w", err)
	}

	ctx := &PooledContext{
		context:    context,
		cookiePath: cookiePath,
		createdAt:  time.Now(),
		lastUsed:   time.Now(),
		parent:     b,
	}

	b.contexts = append(b.contexts, ctx)
	b.inUse++

	return ctx, nil
}

// Release 释放上下文
func (c *PooledContext) Release() error {
	c.parent.mutex.Lock()
	defer c.parent.mutex.Unlock()

	// 检查页面是否已关闭
	if c.IsPageClosed() {
		utils.Info("[-] 页面已关闭，清理上下文")
		// 关闭整个上下文
		c.context.Close()
		// 从父浏览器的上下文中移除
		c.removeFromParent()
		c.parent.inUse--
		return fmt.Errorf("page was closed by user")
	}

	// 保存 Cookie
	if err := c.saveCookie(); err != nil {
		utils.Warn(fmt.Sprintf("[-] 保存 cookie 失败: %v", err))
	}

	// 关闭页面
	if c.page != nil {
		c.page.Close()
		c.page = nil
	}

	c.parent.inUse--
	c.lastUsed = time.Now()

	return nil
}

// removeFromParent 从父浏览器中移除上下文
func (c *PooledContext) removeFromParent() {
	for i, ctx := range c.parent.contexts {
		if ctx == c {
			// 从切片中移除
			c.parent.contexts = append(c.parent.contexts[:i], c.parent.contexts[i+1:]...)
			break
		}
	}
}

// saveCookie 保存 Cookie（私有方法）
func (c *PooledContext) saveCookie() error {
	return c.SaveCookies()
}

// SaveCookies 保存 Cookie（公共方法，供外部调用）
func (c *PooledContext) SaveCookies() error {
	if c.cookiePath == "" {
		return fmt.Errorf("cookie path is empty")
	}
	return c.SaveCookiesTo(c.cookiePath)
}

// SaveCookiesTo 保存 Cookie 到指定路径
func (c *PooledContext) SaveCookiesTo(cookiePath string) error {
	storage, err := c.context.StorageState()
	if err != nil {
		return err
	}

	data, err := json.Marshal(storage)
	if err != nil {
		return err
	}

	// 确保目录存在
	dir := filepath.Dir(cookiePath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create cookie directory failed: %w", err)
		}
	}

	return os.WriteFile(cookiePath, data, 0644)
}

// GetPage 获取或创建页面
func (c *PooledContext) GetPage() (playwright.Page, error) {
	if c.page != nil {
		return c.page, nil
	}

	page, err := c.context.NewPage()
	if err != nil {
		return nil, err
	}

	// 设置默认超时
	page.SetDefaultTimeout(30000) // 30秒
	page.SetDefaultNavigationTimeout(30000)

	// 视口大小已在创建 context 时设置，这里不再重复设置
	// 避免与 context 的视口设置冲突

	c.page = page
	return page, nil
}

// WaitForPageLoad 等待页面完全加载
func (c *PooledContext) WaitForPageLoad() error {
	if c.page == nil {
		return fmt.Errorf("page not created")
	}
	// 等待网络空闲，确保所有资源加载完成
	return c.page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State: playwright.LoadStateNetworkidle,
	})
}

// IsPageClosed 检查页面是否已关闭
// 增加重试机制，避免页面导航或短暂中断导致的误判
func (c *PooledContext) IsPageClosed() bool {
	if c.page == nil {
		return true
	}

	// 重试3次，每次间隔500ms
	maxRetries := 3
	retryDelay := 500 * time.Millisecond

	for i := 0; i < maxRetries; i++ {
		// 如果本次检测通过，直接返回未关闭
		if c.checkPageAlive() {
			return false
		}

		// 如果不是最后一次，等待后重试
		if i < maxRetries-1 {
			time.Sleep(retryDelay)
		}
	}

	// 连续3次检测失败，判定为页面已关闭
	return true
}

// checkPageAlive 检查页面是否存活（单次检测）
func (c *PooledContext) checkPageAlive() bool {
	// 方法1: 尝试执行简单的 JS
	_, err := c.page.Evaluate("1")
	if err != nil {
		return false
	}

	// 方法2: 检查页面 URL
	_, err = c.page.Evaluate(`window.location.href`)
	if err != nil {
		return false
	}

	// 方法3: 检查页面标题
	_, err = c.page.Evaluate(`document.title`)
	if err != nil {
		return false
	}

	return true
}

// Close 关闭上下文
func (c *PooledContext) Close() error {
	if c.page != nil {
		c.page.Close()
	}
	return c.context.Close()
}

// ClosePage 关闭页面（上传成功后调用）
func (c *PooledContext) ClosePage() error {
	if c.page != nil {
		utils.Info("[-] 关闭浏览器页面")
		if err := c.page.Close(); err != nil {
			utils.Warn(fmt.Sprintf("[-] 关闭页面失败: %v", err))
			return err
		}
		c.page = nil
		utils.Info("[-] 浏览器页面已关闭")
	}
	return nil
}

// findLocalChrome 查找本地 Chrome
func findLocalChrome() string {
	paths := []string{
		`C:\Program Files\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		os.Getenv("LOCALAPPDATA") + `\Google\Chrome\Application\chrome.exe`,
		os.Getenv("PROGRAMFILES") + `\Google\Chrome\Application\chrome.exe`,
		os.Getenv("PROGRAMFILES(X86)") + `\Google\Chrome\Application\chrome.exe`,
	}

	for _, path := range paths {
		if path != "" {
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
	}
	return ""
}
