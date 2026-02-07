package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter 平台限流器
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	limits   map[string]RateLimit
	mutex    sync.RWMutex
}

// RateLimit 限流配置
type RateLimit struct {
	Platform    string        `json:"platform"`     // 平台名称
	Requests    int           `json:"requests"`     // 时间窗口内允许的请求数
	Window      time.Duration `json:"window"`       // 时间窗口
	Burst       int           `json:"burst"`        // 突发请求数
	DailyLimit  int           `json:"daily_limit"`  // 每日上传限制
	HourlyLimit int           `json:"hourly_limit"` // 每小时上传限制
}

// DefaultLimits 默认限流配置
var DefaultLimits = map[string]RateLimit{
	"xiaohongshu": {
		Platform:    "xiaohongshu",
		Requests:    5,
		Window:      time.Hour,
		Burst:       2,
		DailyLimit:  20,
		HourlyLimit: 5,
	},
	"douyin": {
		Platform:    "douyin",
		Requests:    10,
		Window:      time.Hour,
		Burst:       3,
		DailyLimit:  50,
		HourlyLimit: 10,
	},
	"bilibili": {
		Platform:    "bilibili",
		Requests:    8,
		Window:      time.Hour,
		Burst:       2,
		DailyLimit:  30,
		HourlyLimit: 8,
	},
	"kuaishou": {
		Platform:    "kuaishou",
		Requests:    6,
		Window:      time.Hour,
		Burst:       2,
		DailyLimit:  25,
		HourlyLimit: 6,
	},
	"tiktok": {
		Platform:    "tiktok",
		Requests:    8,
		Window:      time.Hour,
		Burst:       2,
		DailyLimit:  30,
		HourlyLimit: 8,
	},
	"baijiahao": {
		Platform:    "baijiahao",
		Requests:    5,
		Window:      time.Hour,
		Burst:       2,
		DailyLimit:  20,
		HourlyLimit: 5,
	},
}

// NewRateLimiter 创建限流器
func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		limits:   make(map[string]RateLimit),
	}

	// 初始化默认限流配置
	for platform, limit := range DefaultLimits {
		rl.limits[platform] = limit
		rl.limiters[platform] = rate.NewLimiter(
			rate.Every(limit.Window/time.Duration(limit.Requests)),
			limit.Burst,
		)
	}

	return rl
}

// Allow 检查是否允许请求
func (rl *RateLimiter) Allow(platform string) bool {
	rl.mutex.RLock()
	limiter, exists := rl.limiters[platform]
	rl.mutex.RUnlock()

	if !exists {
		// 如果平台没有配置限流，默认允许
		return true
	}

	return limiter.Allow()
}

// AllowN 检查是否允许 N 个请求
func (rl *RateLimiter) AllowN(platform string, n int) bool {
	rl.mutex.RLock()
	limiter, exists := rl.limiters[platform]
	rl.mutex.RUnlock()

	if !exists {
		return true
	}

	return limiter.AllowN(time.Now(), n)
}

// Wait 等待直到允许请求
func (rl *RateLimiter) Wait(ctx context.Context, platform string) error {
	rl.mutex.RLock()
	limiter, exists := rl.limiters[platform]
	rl.mutex.RUnlock()

	if !exists {
		return nil
	}

	return limiter.Wait(ctx)
}

// GetLimit 获取平台限流配置
func (rl *RateLimiter) GetLimit(platform string) (RateLimit, bool) {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	limit, exists := rl.limits[platform]
	return limit, exists
}

// SetLimit 设置平台限流配置
func (rl *RateLimiter) SetLimit(limit RateLimit) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	rl.limits[limit.Platform] = limit
	rl.limiters[limit.Platform] = rate.NewLimiter(
		rate.Every(limit.Window/time.Duration(limit.Requests)),
		limit.Burst,
	)
}

// RemoveLimit 移除平台限流配置
func (rl *RateLimiter) RemoveLimit(platform string) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	delete(rl.limits, platform)
	delete(rl.limiters, platform)
}

// GetAllLimits 获取所有限流配置
func (rl *RateLimiter) GetAllLimits() map[string]RateLimit {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	result := make(map[string]RateLimit)
	for k, v := range rl.limits {
		result[k] = v
	}
	return result
}

// Reset 重置限流器
func (rl *RateLimiter) Reset(platform string) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	if limit, exists := rl.limits[platform]; exists {
		rl.limiters[platform] = rate.NewLimiter(
			rate.Every(limit.Window/time.Duration(limit.Requests)),
			limit.Burst,
		)
	}
}

// Stats 限流统计
type Stats struct {
	Platform string    `json:"platform"`
	Allowed  int64     `json:"allowed"`
	Rejected int64     `json:"rejected"`
	Limit    RateLimit `json:"limit"`
	Tokens   float64   `json:"tokens"`
}

// PlatformStats 平台限流统计（用于监控）
type PlatformStats struct {
	mu    sync.RWMutex
	stats map[string]*Stats
}

// NewPlatformStats 创建平台统计
func NewPlatformStats() *PlatformStats {
	return &PlatformStats{
		stats: make(map[string]*Stats),
	}
}

// RecordAllowed 记录允许的请求
func (ps *PlatformStats) RecordAllowed(platform string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if _, exists := ps.stats[platform]; !exists {
		ps.stats[platform] = &Stats{Platform: platform}
	}
	ps.stats[platform].Allowed++
}

// RecordRejected 记录拒绝的请求
func (ps *PlatformStats) RecordRejected(platform string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if _, exists := ps.stats[platform]; !exists {
		ps.stats[platform] = &Stats{Platform: platform}
	}
	ps.stats[platform].Rejected++
}

// GetStats 获取统计信息
func (ps *PlatformStats) GetStats(platform string) (*Stats, bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	stats, exists := ps.stats[platform]
	if !exists {
		return nil, false
	}

	// 复制一份返回
	result := *stats
	return &result, true
}

// GetAllStats 获取所有统计信息
func (ps *PlatformStats) GetAllStats() map[string]Stats {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	result := make(map[string]Stats)
	for k, v := range ps.stats {
		result[k] = *v
	}
	return result
}

// ResetStats 重置统计信息
func (ps *PlatformStats) ResetStats(platform string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	delete(ps.stats, platform)
}

// ResetAllStats 重置所有统计信息
func (ps *PlatformStats) ResetAllStats() {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ps.stats = make(map[string]*Stats)
}

// LimiterWithStats 带统计的限流器
type LimiterWithStats struct {
	limiter *RateLimiter
	stats   *PlatformStats
}

// NewLimiterWithStats 创建带统计的限流器
func NewLimiterWithStats() *LimiterWithStats {
	return &LimiterWithStats{
		limiter: NewRateLimiter(),
		stats:   NewPlatformStats(),
	}
}

// Allow 检查是否允许请求并记录统计
func (lws *LimiterWithStats) Allow(platform string) bool {
	allowed := lws.limiter.Allow(platform)
	if allowed {
		lws.stats.RecordAllowed(platform)
	} else {
		lws.stats.RecordRejected(platform)
	}
	return allowed
}

// Wait 等待直到允许请求
func (lws *LimiterWithStats) Wait(ctx context.Context, platform string) error {
	err := lws.limiter.Wait(ctx, platform)
	if err == nil {
		lws.stats.RecordAllowed(platform)
	}
	return err
}

// GetLimit 获取平台限流配置
func (lws *LimiterWithStats) GetLimit(platform string) (RateLimit, bool) {
	return lws.limiter.GetLimit(platform)
}

// SetLimit 设置平台限流配置
func (lws *LimiterWithStats) SetLimit(limit RateLimit) {
	lws.limiter.SetLimit(limit)
}

// GetStats 获取平台统计信息
func (lws *LimiterWithStats) GetStats(platform string) (*Stats, bool) {
	return lws.stats.GetStats(platform)
}

// GetAllStats 获取所有统计信息
func (lws *LimiterWithStats) GetAllStats() map[string]Stats {
	return lws.stats.GetAllStats()
}

// CheckUploadLimit 检查上传限制
func (lws *LimiterWithStats) CheckUploadLimit(platform string, dailyCount, hourlyCount int) error {
	limit, exists := lws.limiter.GetLimit(platform)
	if !exists {
		return nil
	}

	if limit.DailyLimit > 0 && dailyCount >= limit.DailyLimit {
		return fmt.Errorf("平台 %s 已达到每日上传限制 (%d)", platform, limit.DailyLimit)
	}

	if limit.HourlyLimit > 0 && hourlyCount >= limit.HourlyLimit {
		return fmt.Errorf("平台 %s 已达到每小时上传限制 (%d)", platform, limit.HourlyLimit)
	}

	return nil
}
