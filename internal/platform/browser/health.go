package browser

import (
	"context"
	"fmt"
	"time"

	"Fuploader/internal/utils"
)

// HealthStatus 健康状态
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"   // 健康
	HealthStatusDegraded  HealthStatus = "degraded"  // 降级
	HealthStatusUnhealthy HealthStatus = "unhealthy" // 不健康
)

// HealthCheckResult 健康检查结果
type HealthCheckResult struct {
	Status         HealthStatus
	BrowserCount   int
	HealthyCount   int
	UnhealthyCount int
	Messages       []string
	CheckedAt      time.Time
}

// IsHealthy 是否健康
func (r *HealthCheckResult) IsHealthy() bool {
	return r.Status == HealthStatusHealthy
}

// HealthChecker 健康检查器
type HealthChecker struct {
	pool          *Pool
	checkInterval time.Duration
	maxFailures   int
	failureCounts map[*PooledBrowser]int
	lastCheckTime time.Time
	isRunning     bool
	stopChan      chan struct{}
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker(pool *Pool, checkInterval time.Duration, maxFailures int) *HealthChecker {
	if checkInterval <= 0 {
		checkInterval = 30 * time.Second // 默认30秒
	}
	if maxFailures <= 0 {
		maxFailures = 3 // 默认3次失败
	}

	return &HealthChecker{
		pool:          pool,
		checkInterval: checkInterval,
		maxFailures:   maxFailures,
		failureCounts: make(map[*PooledBrowser]int),
		stopChan:      make(chan struct{}),
	}
}

// Start 启动健康检查
func (h *HealthChecker) Start() {
	if h.isRunning {
		return
	}

	h.isRunning = true
	go h.run()
	utils.Info("[-] 浏览器池健康检查已启动")
}

// Stop 停止健康检查
func (h *HealthChecker) Stop() {
	if !h.isRunning {
		return
	}

	h.isRunning = false
	close(h.stopChan)
	utils.Info("[-] 浏览器池健康检查已停止")
}

// run 运行健康检查循环
func (h *HealthChecker) run() {
	ticker := time.NewTicker(h.checkInterval)
	defer ticker.Stop()

	// 立即执行一次检查
	h.check()

	for {
		select {
		case <-ticker.C:
			h.check()
		case <-h.stopChan:
			return
		}
	}
}

// check 执行健康检查
func (h *HealthChecker) check() {
	// 1. 检查浏览器实例健康
	result := h.Check()

	if result.Status == HealthStatusUnhealthy {
		utils.Error(fmt.Sprintf("[-] 浏览器池健康检查失败: %v", result.Messages))
	} else if result.Status == HealthStatusDegraded {
		utils.Warn(fmt.Sprintf("[-] 浏览器池降级: %v", result.Messages))
	}

	// 2. 检查上下文健康
	h.cleanupUnhealthyContexts()

	h.lastCheckTime = time.Now()
}

// Check 执行一次健康检查
func (h *HealthChecker) Check() *HealthCheckResult {
	result := &HealthCheckResult{
		Status:    HealthStatusHealthy,
		Messages:  make([]string, 0),
		CheckedAt: time.Now(),
	}

	h.pool.mutex.RLock()
	browsers := make([]*PooledBrowser, len(h.pool.browsers))
	copy(browsers, h.pool.browsers)
	h.pool.mutex.RUnlock()

	result.BrowserCount = len(browsers)

	for _, browser := range browsers {
		if h.isBrowserHealthy(browser) {
			result.HealthyCount++
			// 重置失败计数
			delete(h.failureCounts, browser)
		} else {
			result.UnhealthyCount++
			h.failureCounts[browser]++

			// 检查是否需要重启浏览器
			if h.failureCounts[browser] >= h.maxFailures {
				utils.Warn(fmt.Sprintf("[-] 浏览器实例连续%d次健康检查失败，尝试重启", h.maxFailures))
				if err := h.restartBrowser(browser); err != nil {
					result.Messages = append(result.Messages, fmt.Sprintf("重启浏览器失败: %v", err))
				} else {
					result.Messages = append(result.Messages, "浏览器已重启")
					delete(h.failureCounts, browser)
				}
			}
		}
	}

	// 确定整体状态
	if result.UnhealthyCount == result.BrowserCount && result.BrowserCount > 0 {
		result.Status = HealthStatusUnhealthy
	} else if result.UnhealthyCount > 0 {
		result.Status = HealthStatusDegraded
	}

	return result
}

// isBrowserHealthy 检查浏览器是否健康
func (h *HealthChecker) isBrowserHealthy(browser *PooledBrowser) bool {
	browser.mutex.Lock()
	defer browser.mutex.Unlock()

	// 检查浏览器是否已关闭
	if browser.browser == nil {
		return false
	}

	// 尝试执行一个简单的操作来验证浏览器是否响应
	// 创建临时上下文进行测试
	_, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 尝试创建新页面来验证浏览器是否正常工作
	testContext, err := browser.browser.NewContext()
	if err != nil {
		return false
	}
	defer testContext.Close()

	testPage, err := testContext.NewPage()
	if err != nil {
		return false
	}
	defer testPage.Close()

	// 尝试访问一个简单页面
	_, err = testPage.Goto("about:blank")
	if err != nil {
		return false
	}

	// 检查内存使用（如果可能）
	// 这里可以添加更多检查逻辑

	return true
}

// checkContextHealth 检查单个上下文的健康状态
func (h *HealthChecker) checkContextHealth(ctx *PooledContext) (bool, string) {
	// 1. 检查页面是否已关闭
	if ctx.IsPageClosed() {
		return false, "页面已关闭"
	}

	// 2. 尝试执行简单JS验证页面响应
	page, err := ctx.GetPage()
	if err != nil {
		return false, fmt.Sprintf("获取页面失败: %v", err)
	}

	_, err = page.Evaluate("1")
	if err != nil {
		return false, fmt.Sprintf("页面无响应: %v", err)
	}

	// 3. 检查登录态有效性（如果配置了验证器）
	if ctx.platform != "" && ctx.accountID > 0 {
		// 这里可以集成session validator进行登录态检查
		// 暂时跳过，避免频繁验证影响性能
	}

	return true, ""
}

// cleanupUnhealthyContexts 清理不健康的上下文
func (h *HealthChecker) cleanupUnhealthyContexts() {
	h.pool.mutex.RLock()
	browsers := make([]*PooledBrowser, len(h.pool.browsers))
	copy(browsers, h.pool.browsers)
	h.pool.mutex.RUnlock()

	for _, browser := range browsers {
		browser.mutex.Lock()
		contexts := make([]*PooledContext, len(browser.contexts))
		copy(contexts, browser.contexts)
		browser.mutex.Unlock()

		for _, ctx := range contexts {
			healthy, reason := h.checkContextHealth(ctx)
			if !healthy {
				utils.Warn(fmt.Sprintf("[-] 上下文不健康 [AccountID: %d, Platform: %s]: %s", 
					ctx.accountID, ctx.platform, reason))
				// 标记为需要清理，实际清理在Release时处理
			}
		}
	}
}

// restartBrowser 重启浏览器实例
func (h *HealthChecker) restartBrowser(oldBrowser *PooledBrowser) error {
	h.pool.mutex.Lock()
	defer h.pool.mutex.Unlock()

	// 关闭旧的浏览器实例
	for _, ctx := range oldBrowser.contexts {
		ctx.Close()
	}
	if err := oldBrowser.browser.Close(); err != nil {
		utils.Warn(fmt.Sprintf("[-] 关闭旧浏览器失败: %v", err))
	}

	// 从池中移除
	for i, b := range h.pool.browsers {
		if b == oldBrowser {
			h.pool.browsers = append(h.pool.browsers[:i], h.pool.browsers[i+1:]...)
			break
		}
	}

	// 创建新的浏览器实例
	newBrowser, err := h.pool.launchBrowser()
	if err != nil {
		return fmt.Errorf("启动新浏览器失败: %w", err)
	}

	pooled := &PooledBrowser{
		browser:  newBrowser,
		contexts: make([]*PooledContext, 0),
	}
	h.pool.browsers = append(h.pool.browsers, pooled)

	utils.Info("[-] 浏览器实例已重启")
	return nil
}

// GetLastCheckTime 获取上次检查时间
func (h *HealthChecker) GetLastCheckTime() time.Time {
	return h.lastCheckTime
}

// IsRunning 是否正在运行
func (h *HealthChecker) IsRunning() bool {
	return h.isRunning
}

// PoolHealthStats 池健康统计
type PoolHealthStats struct {
	TotalBrowsers     int           `json:"total_browsers"`
	HealthyBrowsers   int           `json:"healthy_browsers"`
	UnhealthyBrowsers int           `json:"unhealthy_browsers"`
	TotalContexts     int           `json:"total_contexts"`
	InUseContexts     int           `json:"in_use_contexts"`
	IdleContexts      int           `json:"idle_contexts"`
	LastCheckTime     time.Time     `json:"last_check_time"`
	Uptime            time.Duration `json:"uptime"`
}

// GetHealthStats 获取健康统计
func (h *HealthChecker) GetHealthStats() *PoolHealthStats {
	stats := h.pool.GetStats()
	result := h.Check()

	return &PoolHealthStats{
		TotalBrowsers:     result.BrowserCount,
		HealthyBrowsers:   result.HealthyCount,
		UnhealthyBrowsers: result.UnhealthyCount,
		TotalContexts:     stats.ContextCount,
		InUseContexts:     stats.InUseContextCount,
		IdleContexts:      stats.IdleContextCount,
		LastCheckTime:     h.lastCheckTime,
	}
}
