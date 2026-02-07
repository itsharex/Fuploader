package types

type UploadProgressEvent struct {
	TaskID   int    `json:"taskId"`
	Platform string `json:"platform"`
	Progress int    `json:"progress"`
	Message  string `json:"message"`
}

type UploadCompleteEvent struct {
	TaskID      int    `json:"taskId"`
	Platform    string `json:"platform"`
	PublishURL  string `json:"publishUrl"`
	CompletedAt string `json:"completedAt"`
}

type UploadErrorEvent struct {
	TaskID   int    `json:"taskId"`
	Platform string `json:"platform"`
	Error    string `json:"error"`
	CanRetry bool   `json:"canRetry"`
}

type LoginSuccessEvent struct {
	AccountID int    `json:"accountId"`
	Platform  string `json:"platform"`
	Username  string `json:"username"`
}

type LoginErrorEvent struct {
	AccountID int    `json:"accountId"`
	Platform  string `json:"platform"`
	Error     string `json:"error"`
}

type TaskStatusChangedEvent struct {
	TaskID    int    `json:"taskId"`
	OldStatus string `json:"oldStatus"`
	NewStatus string `json:"newStatus"`
}

type AccountStatusChangedEvent struct {
	AccountID int `json:"accountId"`
	OldStatus int `json:"oldStatus"`
	NewStatus int `json:"newStatus"`
}
