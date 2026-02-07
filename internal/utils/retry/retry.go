// Package retry 提供全面的重试机制实现
package retry

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"Fuploader/internal/utils"
)

// RetryStrategy 重试策略类型
type RetryStrategy string

const (
	// ExponentialBackoff 指数退避策略
	ExponentialBackoff RetryStrategy = "exponential_backoff"
	// FixedInterval 固定间隔策略
	FixedInterval RetryStrategy = "fixed_interval"
	// RandomDelay 随机延迟策略
	RandomDelay RetryStrategy = "random_delay"
	// LinearBackoff 线性退避策略
	LinearBackoff RetryStrategy = "linear_backoff"
)

// RetryCondition 重试条件函数
type RetryCondition func(error) bool

// RetryCallback 重试回调函数
type RetryCallback func(attempt int, delay time.Duration, err error)

// Config 重试配置
type Config struct {
	// 基础配置
	MaxRetries      int           // 最大重试次数
	InitialDelay    time.Duration // 初始延迟
	MaxDelay        time.Duration // 最大延迟
	TotalTimeout    time.Duration // 总超时时间
	
	// 策略配置
	Strategy        RetryStrategy // 重试策略
	BackoffFactor   float64       // 退避因子（用于指数退避）
	Jitter          bool          // 是否启用抖动
	JitterFactor    float64       // 抖动因子 (0.0 - 1.0)
	
	// 条件配置
	RetryableErrors []string      // 可重试的错误类型
	RetryCondition  RetryCondition // 自定义重试条件
	
	// 回调配置
	OnRetry         RetryCallback // 重试时回调
	OnSuccess       func()        // 成功时回调
	OnFailure       func(error)   // 最终失败时回调
	
	// 限流配置
	RateLimit       *RateLimitConfig // 限流配置
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	RequestsPerSecond float64       // 每秒请求数
	BurstSize         int           // 突发请求数
	CooldownPeriod    time.Duration // 冷却期
}

// DefaultConfig 默认重试配置
func DefaultConfig() *Config {
	return &Config{
		MaxRetries:      3,
		InitialDelay:    2 * time.Second,
		MaxDelay:        30 * time.Second,
		TotalTimeout:    5 * time.Minute,
		Strategy:        ExponentialBackoff,
		BackoffFactor:   2.0,
		Jitter:          true,
		JitterFactor: