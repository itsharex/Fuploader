package session

import (
	"fmt"
	"sync"
	"time"
)

// ValidationCacheEntry 验证缓存条目
type ValidationCacheEntry struct {
	Valid      bool
	CheckedAt  time.Time
	ExpiresAt  time.Time
}

// IsExpired 检查缓存是否过期
func (e *ValidationCacheEntry) IsExpired() bool {
	return time.Now().After(e.ExpiresAt)
}

// ValidationCache 验证缓存
type ValidationCache struct {
	cache map[string]*ValidationCacheEntry
	mu    sync.RWMutex
	ttl   time.Duration
}

// NewValidationCache 创建验证缓存
func NewValidationCache(ttl time.Duration) *ValidationCache {
	if ttl <= 0 {
		ttl = 5 * time.Minute // 默认5分钟
	}
	return &ValidationCache{
		cache: make(map[string]*ValidationCacheEntry),
		ttl:   ttl,
	}
}

// Get 获取缓存的验证结果
func (c *ValidationCache) Get(platform string, accountID int64) (*ValidationCacheEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := c.makeKey(platform, accountID)
	entry, exists := c.cache[key]
	if !exists || entry.IsExpired() {
		return nil, false
	}

	return entry, true
}

// Set 设置缓存的验证结果
func (c *ValidationCache) Set(platform string, accountID int64, valid bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := c.makeKey(platform, accountID)
	c.cache[key] = &ValidationCacheEntry{
		Valid:     valid,
		CheckedAt: time.Now(),
		ExpiresAt: time.Now().Add(c.ttl),
	}
}

// Invalidate 使缓存失效
func (c *ValidationCache) Invalidate(platform string, accountID int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := c.makeKey(platform, accountID)
	delete(c.cache, key)
}

// InvalidateAll 使所有缓存失效
func (c *ValidationCache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]*ValidationCacheEntry)
}

// InvalidatePlatform 使指定平台的所有缓存失效
func (c *ValidationCache) InvalidatePlatform(platform string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key := range c.cache {
		if len(key) > len(platform) && key[:len(platform)] == platform {
			delete(c.cache, key)
		}
	}
}

// Cleanup 清理过期缓存
func (c *ValidationCache) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, entry := range c.cache {
		if now.After(entry.ExpiresAt) {
			delete(c.cache, key)
		}
	}
}

// StartCleanupTask 启动定时清理任务
func (c *ValidationCache) StartCleanupTask(interval time.Duration) *time.Ticker {
	if interval <= 0 {
		interval = 10 * time.Minute // 默认10分钟
	}

	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			c.Cleanup()
		}
	}()

	return ticker
}

// makeKey 生成缓存键
func (c *ValidationCache) makeKey(platform string, accountID int64) string {
	return fmt.Sprintf("%s_%d", platform, accountID)
}

// GetStats 获取缓存统计
func (c *ValidationCache) GetStats() (total int, expired int) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	now := time.Now()
	for _, entry := range c.cache {
		total++
		if now.After(entry.ExpiresAt) {
			expired++
		}
	}

	return total, expired
}
