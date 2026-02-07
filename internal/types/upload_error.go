package types

import (
	"errors"
	"fmt"
)

// UploadErrorType 上传错误类型
type UploadErrorType string

const (
	// 可重试错误
	UploadErrorTypeNetwork     UploadErrorType = "network"      // 网络错误
	UploadErrorTypeTimeout     UploadErrorType = "timeout"      // 超时错误
	UploadErrorTypeSelector    UploadErrorType = "selector"     // 选择器错误（页面可能未加载完成）
	UploadErrorTypeUpload      UploadErrorType = "upload"       // 上传错误
	UploadErrorTypeRateLimited UploadErrorType = "rate_limited" // 限流错误

	// 不可重试错误
	UploadErrorTypePlatform      UploadErrorType = "platform"      // 平台错误（如封禁）
	UploadErrorTypeValidation    UploadErrorType = "validation"    // 验证错误
	UploadErrorTypeAuth          UploadErrorType = "auth"          // 认证错误
	UploadErrorTypeUnrecoverable UploadErrorType = "unrecoverable" // 不可恢复错误
)

// IsRetryable 判断错误类型是否可重试
func (t UploadErrorType) IsRetryable() bool {
	switch t {
	case UploadErrorTypeNetwork, UploadErrorTypeTimeout, UploadErrorTypeSelector,
		UploadErrorTypeUpload, UploadErrorTypeRateLimited:
		return true
	default:
		return false
	}
}

// UploadError 上传错误
type UploadError struct {
	Type      UploadErrorType
	Step      string
	Message   string
	Cause     error
	Retryable bool
	Code      ErrorCode
}

// Error 实现 error 接口
func (e *UploadError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s][%s] %s: %v", e.Type, e.Step, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s][%s] %s", e.Type, e.Step, e.Message)
}

// Unwrap 实现错误链
func (e *UploadError) Unwrap() error {
	return e.Cause
}

// NewUploadError 创建上传错误
func NewUploadError(errType UploadErrorType, step string, message string, cause error) *UploadError {
	return &UploadError{
		Type:      errType,
		Step:      step,
		Message:   message,
		Cause:     cause,
		Retryable: errType.IsRetryable(),
	}
}

// NewNetworkError 创建网络错误
func NewNetworkError(step string, cause error) *UploadError {
	return NewUploadError(UploadErrorTypeNetwork, step, "网络错误", cause)
}

// NewTimeoutError 创建超时错误
func NewTimeoutError(step string, cause error) *UploadError {
	return NewUploadError(UploadErrorTypeTimeout, step, "操作超时", cause)
}

// NewSelectorError 创建选择器错误
func NewSelectorError(step string, selector string, cause error) *UploadError {
	return NewUploadError(UploadErrorTypeSelector, step, fmt.Sprintf("元素选择失败: %s", selector), cause)
}

// NewUploadError_ 创建上传错误（带下划线避免与类型名冲突）
func NewUploadError_(step string, cause error) *UploadError {
	return NewUploadError(UploadErrorTypeUpload, step, "上传失败", cause)
}

// NewRateLimitedError 创建限流错误
func NewRateLimitedError(step string, cause error) *UploadError {
	return NewUploadError(UploadErrorTypeRateLimited, step, "请求过于频繁", cause)
}

// NewPlatformError 创建平台错误
func NewPlatformError(step string, message string, cause error) *UploadError {
	return NewUploadError(UploadErrorTypePlatform, step, message, cause)
}

// NewValidationError 创建验证错误
func NewValidationError(step string, message string, cause error) *UploadError {
	return NewUploadError(UploadErrorTypeValidation, step, message, cause)
}

// NewAuthError 创建认证错误
func NewAuthError(step string, message string, cause error) *UploadError {
	return NewUploadError(UploadErrorTypeAuth, step, message, cause)
}

// NewUnrecoverableError 创建不可恢复错误
func NewUnrecoverableError(step string, message string, cause error) *UploadError {
	return NewUploadError(UploadErrorTypeUnrecoverable, step, message, cause)
}

// IsUploadError 判断是否为上传错误
func IsUploadError(err error) (*UploadError, bool) {
	var uploadErr *UploadError
	if errors.As(err, &uploadErr) {
		return uploadErr, true
	}
	return nil, false
}

// IsRetryableError 判断错误是否可重试
func IsRetryableError(err error) bool {
	if uploadErr, ok := IsUploadError(err); ok {
		return uploadErr.Retryable
	}
	// 默认情况下，未知错误不可重试
	return false
}

// ClassifyError 根据错误内容分类错误
func ClassifyError(err error, step string) *UploadError {
	if err == nil {
		return nil
	}

	// 如果已经是 UploadError，直接返回
	if uploadErr, ok := IsUploadError(err); ok {
		return uploadErr
	}

	errStr := err.Error()

	// 网络错误
	if containsAny(errStr, []string{
		"net::", "network", "connection", "timeout", "deadline exceeded",
		"connection refused", "no such host", "i/o timeout",
	}) {
		return NewNetworkError(step, err)
	}

	// 超时错误
	if containsAny(errStr, []string{
		"timeout", "timed out", "context deadline exceeded", "waiting for",
	}) {
		return NewTimeoutError(step, err)
	}

	// 选择器错误
	if containsAny(errStr, []string{
		"selector", "element not found", "locator", "count",
	}) {
		return NewSelectorError(step, "", err)
	}

	// 限流错误
	if containsAny(errStr, []string{
		"rate limit", "too many requests", "429", "frequency",
		"请求过于频繁", "操作太频繁",
	}) {
		return NewRateLimitedError(step, err)
	}

	// 认证错误
	if containsAny(errStr, []string{
		"auth", "unauthorized", "401", "cookie", "login", "session",
		"未登录", "登录过期", "认证失败",
	}) {
		return NewAuthError(step, "认证失败", err)
	}

	// 平台错误
	if containsAny(errStr, []string{
		"platform", "publish error", "upload failed", "检测失败",
		"审核不通过", "发布失败",
	}) {
		return NewPlatformError(step, "平台错误", err)
	}

	// 默认归类为不可恢复错误
	return NewUnrecoverableError(step, "未知错误", err)
}

// containsAny 检查字符串是否包含任意一个子串
func containsAny(s string, substrs []string) bool {
	lowerS := toLower(s)
	for _, substr := range substrs {
		if contains(lowerS, toLower(substr)) {
			return true
		}
	}
	return false
}

// toLower 转换为小写（简单实现）
func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c = c + ('a' - 'A')
		}
		result[i] = c
	}
	return string(result)
}

// contains 检查是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		findSubstr(s, substr) >= 0)
}

// findSubstr 查找子串位置
func findSubstr(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// RetryPolicy 重试策略
type RetryPolicy struct {
	MaxRetries    int
	InitialDelay  int64 // 毫秒
	MaxDelay      int64 // 毫秒
	BackoffFactor float64
	Jitter        bool // 是否添加随机抖动
}

// DefaultRetryPolicy 默认重试策略
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxRetries:    3,
		InitialDelay:  2000,  // 2秒
		MaxDelay:      30000, // 30秒
		BackoffFactor: 2.0,
		Jitter:        true,
	}
}

// AggressiveRetryPolicy 激进重试策略（用于网络不稳定情况）
func AggressiveRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxRetries:    5,
		InitialDelay:  1000,  // 1秒
		MaxDelay:      60000, // 60秒
		BackoffFactor: 1.5,
		Jitter:        true,
	}
}

// ConservativeRetryPolicy 保守重试策略（用于限流情况）
func ConservativeRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxRetries:    3,
		InitialDelay:  5000,   // 5秒
		MaxDelay:      120000, // 2分钟
		BackoffFactor: 2.0,
		Jitter:        true,
	}
}

// CalculateRetryDelay 计算重试延迟
func (p RetryPolicy) CalculateRetryDelay(retryCount int) int64 {
	if retryCount <= 0 {
		return 0
	}

	// 指数退避
	delay := float64(p.InitialDelay)
	for i := 1; i < retryCount; i++ {
		delay = delay * p.BackoffFactor
		if delay > float64(p.MaxDelay) {
			delay = float64(p.MaxDelay)
			break
		}
	}

	// 添加抖动 (±25%)
	if p.Jitter {
		jitter := delay * 0.25
		delay = delay - jitter + (float64(int64(delay)*int64(retryCount)%100)/100.0)*2*jitter
	}

	return int64(delay)
}
