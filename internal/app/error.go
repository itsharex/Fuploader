package app

import "Fuploader/internal/config"

type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

func (e *AppError) Error() string {
	return e.Message
}

func NewAppError(code string, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

func NewAppErrorWithDetail(code string, message string, detail string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Detail:  detail,
	}
}

func ErrInvalidParam(message string) *AppError {
	return NewAppError(config.ErrInvalidParam, message)
}

func ErrAccountNotFound() *AppError {
	return NewAppError(config.ErrAccountNotFound, "账号不存在")
}

func ErrAccountInvalid() *AppError {
	return NewAppError(config.ErrAccountInvalid, "账号Cookie失效")
}

func ErrVideoNotFound() *AppError {
	return NewAppError(config.ErrVideoNotFound, "视频不存在")
}

func ErrVideoInvalid(message string) *AppError {
	return NewAppError(config.ErrVideoInvalid, message)
}

func ErrTaskNotFound() *AppError {
	return NewAppError(config.ErrTaskNotFound, "任务不存在")
}

func ErrTaskCannotCancel() *AppError {
	return NewAppError(config.ErrTaskCannotCancel, "任务无法取消")
}

func ErrUploadFailed(detail string) *AppError {
	return NewAppErrorWithDetail(config.ErrUploadFailed, "上传失败", detail)
}

func ErrNetworkError(detail string) *AppError {
	return NewAppErrorWithDetail(config.ErrNetworkError, "网络错误", detail)
}

func ErrPlatformError(detail string) *AppError {
	return NewAppErrorWithDetail(config.ErrPlatformError, "平台错误", detail)
}

func ErrScheduleInvalid(message string) *AppError {
	return NewAppError(config.ErrScheduleInvalid, message)
}

func ErrInternal(detail string) *AppError {
	return NewAppErrorWithDetail(config.ErrInternal, "内部错误", detail)
}
