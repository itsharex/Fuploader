# Fuploader 后端 API 实现需求文档

本文档汇总了前端开发中需要后端 Wails 提供的所有绑定方法。

---

## 账号管理 API

### 1. GetAccounts
获取所有账号列表

```go
func (a *App) GetAccounts() ([]Account, error)
```

**返回:**
- `[]Account` - 账号列表

**Account 结构:**
```go
type Account struct {
    ID        int       `json:"id"`
    Platform  string    `json:"platform"`  // douyin, tencent, kuaishou, tiktok, bilibili, xiaohongshu, baijiahao
    Name      string    `json:"name"`
    Username  string    `json:"username,omitempty"`
    Avatar    string    `json:"avatar,omitempty"`
    CookiePath string   `json:"cookiePath,omitempty"`
    Status    int       `json:"status"`    // 0-无效, 1-有效, 2-已过期
    CreatedAt string    `json:"createdAt"`
    UpdatedAt string    `json:"updatedAt"`
}
```

---

### 2. AddAccount
添加新账号

```go
func (a *App) AddAccount(platform string, name string) (Account, error)
```

**参数:**
- `platform` - 平台类型 (douyin, tencent, kuaishou, tiktok, bilibili, xiaohongshu, baijiahao)
- `name` - 账号名称/备注

**返回:**
- `Account` - 新创建的账号

---

### 3. DeleteAccount
删除账号

```go
func (a *App) DeleteAccount(id int) error
```

**参数:**
- `id` - 账号ID

---

### 4. ValidateAccount
验证账号有效性

```go
func (a *App) ValidateAccount(id int) (bool, error)
```

**参数:**
- `id` - 账号ID

**返回:**
- `bool` - true=有效, false=无效

---

### 5. LoginAccount
触发账号登录流程

```go
func (a *App) LoginAccount(id int) error
```

**参数:**
- `id` - 账号ID

**说明:** 此方法应打开浏览器让用户完成登录，登录成功后触发 `login:success` 事件

---

### 6. UpdateAccount
更新账号信息

```go
func (a *App) UpdateAccount(account Account) error
```

**参数:**
- `account` - 账号对象

---

## 视频管理 API

### 1. GetVideos
获取所有视频列表

```go
func (a *App) GetVideos() ([]Video, error)
```

**返回:**
- `[]Video` - 视频列表

**Video 结构:**
```go
type Video struct {
    ID          int       `json:"id"`
    Filename    string    `json:"filename"`
    FilePath    string    `json:"filePath"`
    FileSize    int64     `json:"fileSize"`
    Duration    float64   `json:"duration,omitempty"`
    Width       int       `json:"width,omitempty"`
    Height      int       `json:"height,omitempty"`
    Title       string    `json:"title,omitempty"`
    Description string    `json:"description,omitempty"`
    Tags        []string  `json:"tags,omitempty"`
    Thumbnail   string    `json:"thumbnail,omitempty"`
    CreatedAt   string    `json:"createdAt"`
}
```

---

### 2. AddVideo
添加视频

```go
func (a *App) AddVideo(filePath string) (Video, error)
```

**参数:**
- `filePath` - 视频文件路径

**返回:**
- `Video` - 新添加的视频信息

---

### 3. UpdateVideo
更新视频信息

```go
func (a *App) UpdateVideo(video Video) error
```

**参数:**
- `video` - 视频对象

---

### 4. DeleteVideo
删除视频

```go
func (a *App) DeleteVideo(id int) error
```

**参数:**
- `id` - 视频ID

---

### 5. SelectVideoFile
打开文件选择对话框

```go
func (a *App) SelectVideoFile() (string, error)
```

**返回:**
- `string` - 选中的文件路径

---

## 上传任务 API

### 1. CreateUploadTask
创建上传任务

```go
func (a *App) CreateUploadTask(videoID int, accountIDs []int, scheduleTime *string) ([]UploadTask, error)
```

**参数:**
- `videoID` - 视频ID
- `accountIDs` - 账号ID列表
- `scheduleTime` - 定时发布时间 (ISO8601格式，null表示立即发布)

**返回:**
- `[]UploadTask` - 创建的任务列表

**UploadTask 结构:**
```go
type UploadTask struct {
    ID          int       `json:"id"`
    VideoID     int       `json:"videoId"`
    Video       Video     `json:"video"`
    AccountID   int       `json:"accountId"`
    Account     Account   `json:"account"`
    Platform    string    `json:"platform"`
    Status      string    `json:"status"`      // pending, uploading, success, failed, cancelled
    Progress    int       `json:"progress"`    // 0-100
    ScheduleTime *string  `json:"scheduleTime,omitempty"`
    PublishURL  string    `json:"publishUrl,omitempty"`
    ErrorMsg    string    `json:"errorMsg,omitempty"`
    RetryCount  int       `json:"retryCount"`
    CreatedAt   string    `json:"createdAt"`
    UpdatedAt   string    `json:"updatedAt"`
}
```

---

### 2. GetUploadTasks
获取任务列表

```go
func (a *App) GetUploadTasks(status string) ([]UploadTask, error)
```

**参数:**
- `status` - 状态过滤 (空字符串表示全部，可选值: pending, uploading, success, failed, cancelled)

**返回:**
- `[]UploadTask` - 任务列表

---

### 3. CancelUploadTask
取消上传任务

```go
func (a *App) CancelUploadTask(id int) error
```

**参数:**
- `id` - 任务ID

---

### 4. RetryUploadTask
重试失败的任务

```go
func (a *App) RetryUploadTask(id int) error
```

**参数:**
- `id` - 任务ID

---

### 5. DeleteUploadTask
删除任务

```go
func (a *App) DeleteUploadTask(id int) error
```

**参数:**
- `id` - 任务ID

---

## 定时配置 API

### 1. GetScheduleConfig
获取定时发布配置

```go
func (a *App) GetScheduleConfig() (*ScheduleConfig, error)
```

**返回:**
- `*ScheduleConfig` - 配置对象

**ScheduleConfig 结构:**
```go
type ScheduleConfig struct {
    ID           int      `json:"id"`
    VideosPerDay int      `json:"videosPerDay"`
    DailyTimes   []string `json:"dailyTimes"`   // 如: ["09:00", "15:00", "20:00"]
    StartDays    int      `json:"startDays"`    // 从今天开始第几天开始
    TimeZone     string   `json:"timeZone"`     // 如: "Asia/Shanghai"
}
```

---

### 2. UpdateScheduleConfig
更新定时发布配置

```go
func (a *App) UpdateScheduleConfig(config ScheduleConfig) error
```

**参数:**
- `config` - 配置对象

---

### 3. GenerateScheduleTimes
生成定时发布时间列表

```go
func (a *App) GenerateScheduleTimes(videoCount int) ([]string, error)
```

**参数:**
- `videoCount` - 视频数量

**返回:**
- `[]string` - ISO8601 格式的时间列表

---

## 系统 API

### 1. GetAppVersion
获取应用版本信息

```go
func (a *App) GetAppVersion() (AppVersion, error)
```

**返回:**
- `AppVersion` - 版本信息

**AppVersion 结构:**
```go
type AppVersion struct {
    Version      string `json:"version"`
    BuildTime    string `json:"buildTime"`
    GoVersion    string `json:"goVersion"`
    WailsVersion string `json:"wailsVersion"`
}
```

---

### 2. OpenDirectory
打开目录

```go
func (a *App) OpenDirectory(path string) error
```

**参数:**
- `path` - 目录路径

---

## 事件定义

后端需要触发以下事件通知前端：

### 1. 上传进度事件
```go
runtime.EventsEmit(a.ctx, "upload:progress", UploadProgressEvent{
    TaskID:   taskID,
    Platform: platform,
    Progress: progress,  // 0-100
    Message:  message,
})
```

### 2. 上传完成事件
```go
runtime.EventsEmit(a.ctx, "upload:complete", UploadCompleteEvent{
    TaskID:      taskID,
    Platform:    platform,
    PublishURL:  publishURL,
    CompletedAt: time.Now().Format(time.RFC3339),
})
```

### 3. 上传错误事件
```go
runtime.EventsEmit(a.ctx, "upload:error", UploadErrorEvent{
    TaskID:   taskID,
    Platform: platform,
    Error:    errorMsg,
    CanRetry: canRetry,
})
```

### 4. 登录成功事件
```go
runtime.EventsEmit(a.ctx, "login:success", LoginSuccessEvent{
    AccountID: accountID,
    Platform:  platform,
    Username:  username,
})
```

### 5. 登录失败事件
```go
runtime.EventsEmit(a.ctx, "login:error", LoginErrorEvent{
    AccountID: accountID,
    Platform:  platform,
    Error:     errorMsg,
})
```

### 6. 任务状态变更事件
```go
runtime.EventsEmit(a.ctx, "task:statusChanged", TaskStatusChangedEvent{
    TaskID:    taskID,
    OldStatus: oldStatus,
    NewStatus: newStatus,
})
```

### 7. 账号状态变更事件
```go
runtime.EventsEmit(a.ctx, "account:statusChanged", AccountStatusChangedEvent{
    AccountID: accountID,
    OldStatus: oldStatus,
    NewStatus: newStatus,
})
```

---

## 事件数据结构

```go
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
```

---

## 实现检查清单

- [ ] GetAccounts
- [ ] AddAccount
- [ ] DeleteAccount
- [ ] ValidateAccount
- [ ] LoginAccount
- [ ] UpdateAccount
- [ ] GetVideos
- [ ] AddVideo
- [ ] UpdateVideo
- [ ] DeleteVideo
- [ ] SelectVideoFile
- [ ] CreateUploadTask
- [ ] GetUploadTasks
- [ ] CancelUploadTask
- [ ] RetryUploadTask
- [ ] DeleteUploadTask
- [ ] GetScheduleConfig
- [ ] UpdateScheduleConfig
- [ ] GenerateScheduleTimes
- [ ] GetAppVersion
- [ ] OpenDirectory
- [ ] 事件: upload:progress
- [ ] 事件: upload:complete
- [ ] 事件: upload:error
- [ ] 事件: login:success
- [ ] 事件: login:error
- [ ] 事件: task:statusChanged
- [ ] 事件: account:statusChanged
