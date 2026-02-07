package types

import "fmt"

// ErrorCode 错误码类型
type ErrorCode int

// 错误码定义
const (
	// 通用错误 (1-99)
	ErrCodeSuccess          ErrorCode = 0
	ErrCodeUnknown          ErrorCode = 1
	ErrCodeInvalidParam     ErrorCode = 2
	ErrCodeInternalError    ErrorCode = 3
	ErrCodeNotFound         ErrorCode = 4
	ErrCodeAlreadyExists    ErrorCode = 5
	ErrCodePermissionDenied ErrorCode = 6
	ErrCodeTimeout          ErrorCode = 7

	// 数据库错误 (100-199)
	ErrCodeDBConnect    ErrorCode = 100
	ErrCodeDBQuery      ErrorCode = 101
	ErrCodeDBInsert     ErrorCode = 102
	ErrCodeDBUpdate     ErrorCode = 103
	ErrCodeDBDelete     ErrorCode = 104
	ErrCodeDBNotFound   ErrorCode = 105
	ErrCodeDBConstraint ErrorCode = 106

	// 账号相关错误 (200-299)
	ErrCodeAccountNotFound    ErrorCode = 200
	ErrCodeAccountInvalid     ErrorCode = 201
	ErrCodeAccountLoginFailed ErrorCode = 202
	ErrCodeAccountExists      ErrorCode = 203
	ErrCodeCookieInvalid      ErrorCode = 204
	ErrCodeCookieExpired      ErrorCode = 205

	// 视频相关错误 (300-399)
	ErrCodeVideoNotFound    ErrorCode = 300
	ErrCodeVideoInvalid     ErrorCode = 301
	ErrCodeVideoTooLarge    ErrorCode = 302
	ErrCodeVideoFormatError ErrorCode = 303
	ErrCodeVideoReadError   ErrorCode = 304

	// 上传任务错误 (400-499)
	ErrCodeTaskNotFound     ErrorCode = 400
	ErrCodeTaskCreateFailed ErrorCode = 401
	ErrCodeTaskCancelFailed ErrorCode = 402
	ErrCodeTaskAlreadyRunning ErrorCode = 403
	ErrCodeTaskRetryFailed  ErrorCode = 404
	ErrCodeTaskDeleteFailed ErrorCode = 405

	// 平台上传错误 (500-599)
	ErrCodePlatformNotSupported ErrorCode = 500
	ErrCodePlatformLoginFailed  ErrorCode = 501
	ErrCodePlatformUploadFailed ErrorCode = 502
	ErrCodePlatformCookieInvalid ErrorCode = 503
	ErrCodePlatformRateLimited  ErrorCode = 504
	ErrCodePlatformNetworkError ErrorCode = 505

	// 调度错误 (600-699)
	ErrCodeScheduleInvalid   ErrorCode = 600
	ErrCodeScheduleConflict  ErrorCode = 601
	ErrCodeScheduleTimeError ErrorCode = 602
)

// ErrorCodeMessage 错误码对应的消息
var ErrorCodeMessage = map[ErrorCode]string{
	ErrCodeSuccess:          "成功",
	ErrCodeUnknown:          "未知错误",
	ErrCodeInvalidParam:     "参数错误",
	ErrCodeInternalError:    "内部错误",
	ErrCodeNotFound:         "资源不存在",
	ErrCodeAlreadyExists:    "资源已存在",
	ErrCodePermissionDenied: "权限不足",
	ErrCodeTimeout:          "操作超时",

	ErrCodeDBConnect:    "数据库连接失败",
	ErrCodeDBQuery:      "数据库查询失败",
	ErrCodeDBInsert:     "数据库插入失败",
	ErrCodeDBUpdate:     "数据库更新失败",
	ErrCodeDBDelete:     "数据库删除失败",
	ErrCodeDBNotFound:   "数据库记录不存在",
	ErrCodeDBConstraint: "数据库约束错误",

	ErrCodeAccountNotFound:    "账号不存在",
	ErrCodeAccountInvalid:     "账号无效",
	ErrCodeAccountLoginFailed: "账号登录失败",
	ErrCodeAccountExists:      "账号已存在",
	ErrCodeCookieInvalid:      "Cookie 无效",
	ErrCodeCookieExpired:      "Cookie 已过期",

	ErrCodeVideoNotFound:    "视频不存在",
	ErrCodeVideoInvalid:     "视频无效",
	ErrCodeVideoTooLarge:    "视频文件过大",
	ErrCodeVideoFormatError: "视频格式错误",
	ErrCodeVideoReadError:   "视频读取失败",

	ErrCodeTaskNotFound:       "任务不存在",
	ErrCodeTaskCreateFailed:   "任务创建失败",
	ErrCodeTaskCancelFailed:   "任务取消失败",
	ErrCodeTaskAlreadyRunning: "任务正在运行中",
	ErrCodeTaskRetryFailed:    "任务重试失败",
	ErrCodeTaskDeleteFailed:   "任务删除失败",

	ErrCodePlatformNotSupported: "平台不支持",
	ErrCodePlatformLoginFailed:  "平台登录失败",
	ErrCodePlatformUploadFailed: "平台上传失败",
	ErrCodePlatformCookieInvalid: "平台 Cookie 无效",
	ErrCodePlatformRateLimited:  "平台请求过于频繁",
	ErrCodePlatformNetworkError: "平台网络错误",

	ErrCodeScheduleInvalid:   "定时配置无效",
	ErrCodeScheduleConflict:  "定时时间冲突",
	ErrCodeScheduleTimeError: "定时时间错误",
}

// AppError 应用错误
type AppError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Detail  string    `json:"detail,omitempty"`
}

// Error 实现 error 接口
func (e *AppError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("[%d] %s: %s", e.Code, e.Message, e.Detail)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// NewError 创建错误
func NewError(code ErrorCode, detail ...string) *AppError {
	msg, ok := ErrorCodeMessage[code]
	if !ok {
		msg = "未知错误"
	}

	detailStr := ""
	if len(detail) > 0 {
		detailStr = detail[0]
	}

	return &AppError{
		Code:    code,
		Message: msg,
		Detail:  detailStr,
	}
}

// WrapError 包装错误
func WrapError(code ErrorCode, err error) *AppError {
	appErr := NewError(code)
	if err != nil {
		appErr.Detail = err.Error()
	}
	return appErr
}

// IsErrorCode 检查错误码
func IsErrorCode(err error, code ErrorCode) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code == code
	}
	return false
}

// 便捷函数

// NewAccountNotFoundError 账号不存在错误
func NewAccountNotFoundError(id int) *AppError {
	return NewError(ErrCodeAccountNotFound, fmt.Sprintf("账号 ID: %d", id))
}

// NewVideoNotFoundError 视频不存在错误
func NewVideoNotFoundError(id int) *AppError {
	return NewError(ErrCodeVideoNotFound, fmt.Sprintf("视频 ID: %d", id))
}

// NewTaskNotFoundError 任务不存在错误
func NewTaskNotFoundError(id int) *AppError {
	return NewError(ErrCodeTaskNotFound, fmt.Sprintf("任务 ID: %d", id))
}

// NewPlatformNotSupportedError 平台不支持错误
func NewPlatformNotSupportedError(platform string) *AppError {
	return NewError(ErrCodePlatformNotSupported, fmt.Sprintf("平台: %s", platform))
}

// NewDBError 数据库错误
func NewDBError(operation string, err error) *AppError {
	var code ErrorCode
	switch operation {
	case "connect":
		code = ErrCodeDBConnect
	case "query":
		code = ErrCodeDBQuery
	case "insert":
		code = ErrCodeDBInsert
	case "update":
		code = ErrCodeDBUpdate
	case "delete":
		code = ErrCodeDBDelete
	default:
		code = ErrCodeInternalError
	}
	return WrapError(code, err)
}
