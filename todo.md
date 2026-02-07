# Fuploader 后端开发待办事项

## 说明

本文档根据 `core-logic.md` 中的 Python 实现逻辑规划，分为两类任务：
- **【可直接移植】**: 有明确的 Python 实现参考，可直接翻译为 Go
- **【需参考 GitHub】**: 需要参考社区实现或第三方库

---

## 高优先级任务

### 1. 事件发射机制 (Wails EventsEmit)

**状态**: ⏳ 待实现  
**文件**: `internal/app/app.go`  
**类型**: 【可直接移植】参考 `sau_backend.py` SSE 实现

**实现要点**:
```python
# Python 参考 (sau_backend.py)
def sse_stream(status_queue):
    while True:
        if not status_queue.empty():
            msg = status_queue.get()
            yield f"data: {msg}\n\n"
        else:
            time.sleep(0.1)
```

**Go 实现**:
- [ ] 在 `emitEvent` 方法中使用 `runtime.EventsEmit` 向前端发送事件
- [ ] 在 `uploadService` 中订阅事件并转发到前端
- [ ] 事件列表:
  - `upload:progress` - 上传进度 (参考 Python 进度检测逻辑)
  - `upload:complete` - 上传完成
  - `upload:error` - 上传错误
  - `login:success` - 登录成功
  - `login:error` - 登录失败
  - `task:statusChanged` - 任务状态变更
  - `account:statusChanged` - 账号状态变更

---

### 2. 文件选择对话框

**状态**: ⏳ 待实现  
**文件**: `internal/app/app.go`  
**类型**: 【可直接移植】Wails 原生支持

**实现要点**:
- [ ] 实现 `OpenDirectory` 方法 - 使用 `runtime.OpenDirectoryDialog`
- [ ] 实现 `SelectVideoFile` 方法 - 使用 `runtime.OpenFileDialog`
- [ ] 支持多文件选择

```go
// Wails 实现示例
func (a *App) SelectVideoFile() (string, error) {
    return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
        Title: "选择视频文件",
        Filters: []runtime.FileFilter{
            {DisplayName: "视频文件 (*.mp4, *.mov)", Pattern: "*.mp4;*.mov"},
        },
    })
}
```

---

### 3. 抖音平台 Playwright 上传实现

**状态**: ⏳ 待实现  
**文件**: `internal/platform/douyin/uploader.go`  
**类型**: 【可直接移植】参考 `uploader/douyin_uploader/main.py`

**Python 实现参考**: `uploader/douyin_uploader/main.py`

**核心流程** (直接翻译 Python 逻辑):

```python
# Python 核心流程参考
async def upload(self, playwright: Playwright) -> None:
    # 1. 启动浏览器
    browser = await playwright.chromium.launch(headless=self.headless, executable_path=self.local_executable_path)
    
    # 2. 加载 Cookie
    context = await browser.new_context(storage_state=f"{self.account_file}")
    
    # 3. 访问上传页面
    await page.goto("https://creator.douyin.com/creator-micro/content/upload")
    
    # 4. 上传视频文件
    await page.locator("div[class^='container'] input").set_input_files(self.file_path)
    
    # 5. 填写标题和话题
    title_container = page.get_by_text('作品标题').locator("..").locator("xpath=following-sibling::div[1]").locator("input")
    await title_container.fill(self.title[:30])
    
    # 6. 添加标签
    css_selector = ".zone-container"
    for tag in self.tags:
        await page.type(css_selector, "#" + tag)
        await page.press(css_selector, "Space")
    
    # 7. 检测上传完成
    while True:
        number = await page.locator('[class^="long-card"] div:has-text("重新上传")').count()
        if number > 0:
            break
        await asyncio.sleep(2)
    
    # 8. 设置定时发布
    if self.publish_date != 0:
        await self.set_schedule_time_douyin(page, self.publish_date)
    
    # 9. 点击发布
    await page.get_by_role('button', name="发布", exact=True).click()
    
    # 10. 保存 Cookie
    await context.storage_state(path=self.account_file)
```

**Go 实现任务**:
- [ ] 添加 playwright-go 依赖
- [ ] 实现 `ValidateCookie` - 验证 Cookie 是否有效
  ```python
  # Python 参考
  async def cookie_auth(account_file):
      browser = await playwright.chromium.launch(headless=True)
      context = await browser.new_context(storage_state=account_file)
      page = await context.new_page()
      await page.goto("https://creator.douyin.com/creator-micro/content/upload")
      try:
          await page.wait_for_url("https://creator.douyin.com/creator-micro/content/upload", timeout=5000)
          if await page.get_by_text('手机号登录').count() or await page.get_by_text('扫码登录').count():
              return False
          return True
      except:
          return False
  ```
- [ ] 实现 `Upload` - 完整的视频上传流程
- [ ] 实现 `setScheduleTime` - 定时发布设置
- [ ] 实现 `setThumbnail` - 封面设置
- [ ] 实现 `setProductLink` - 商品链接设置
- [ ] 实现 `handleUploadError` - 上传错误重试
- [ ] 实现 `handleAutoVideoCover` - 自动处理封面提示

---

### 4. 账号登录功能

**状态**: ⏳ 待实现  
**文件**: `internal/service/account.go`, `internal/platform/douyin/uploader.go`  
**类型**: 【可直接移植】参考 `uploader/douyin_uploader/main.py`

**Python 实现参考**:
```python
async def douyin_cookie_gen(account_file):
    async with async_playwright() as playwright:
        browser = await playwright.chromium.launch(headless=False)  # 必须可视化
        context = await browser.new_context()
        page = await context.new_page()
        await page.goto("https://creator.douyin.com/")
        await page.pause()  # 等待用户手动登录
        await context.storage_state(path=account_file)  # 保存 Cookie
```

**Go 实现任务**:
- [ ] 实现 `LoginAccount` 方法 - 打开浏览器让用户手动登录
- [ ] 登录成功后保存 Cookie
- [ ] 触发 `login:success` 或 `login:error` 事件
- [ ] 支持暂停等待用户操作 (Wails 可以通过 Events 实现类似功能)

---

### 5. 视频元数据提取

**状态**: ⏳ 待实现  
**文件**: `internal/service/file.go`  
**类型**: 【需参考 GitHub】使用第三方库如 `github.com/3d0c/gmf` 或 `github.com/asticode/go-astiencoder`

**实现要点**:
- [ ] 提取视频时长
- [ ] 提取视频分辨率 (width/height)
- [ ] 生成视频缩略图

**参考库**:
- `github.com/3d0c/gmf` - Go Media Framework
- `github.com/asticode/go-astiencoder` - 视频处理
- `github.com/u2takey/ffmpeg-go` - FFmpeg Go 绑定

---

## 中优先级任务

### 6. 其他平台上传实现

**目录**: `internal/platform/`  
**类型**: 【可直接移植】参考对应 `uploader/*_uploader/main.py`

每个平台的实现逻辑类似，只需适配不同的页面元素选择器:

| 平台 | Python 参考文件 | 状态 |
|------|----------------|------|
| 视频号 (`tencent/`) | `uploader/tencent_uploader/main.py` | ⏳ 待实现 |
| 快手 (`kuaishou/`) | `uploader/ks_uploader/main.py` | ⏳ 待实现 |
| Bilibili (`bilibili/`) | `uploader/bilibili_uploader/main.py` | ⏳ 待实现 |
| 小红书 (`xiaohongshu/`) | `uploader/xiaohongshu_uploader/main.py` | ⏳ 待实现 |
| 百家号 (`baijiahao/`) | `uploader/baijiahao_uploader/main.py` | ⏳ 待实现 |
| TikTok (`tiktok/`) | `uploader/tk_uploader/main_chrome.py` | ⏳ 待实现 |

**快手平台特殊处理** (参考 Python):
```python
# 快手使用文件选择器上传
async with page.expect_file_chooser() as fc_info:
    await upload_button.click()
file_chooser = await fc_info.value
await file_chooser.set_files(self.file_path)

# 快手只能添加3个话题
for index, tag in enumerate(self.tags[:3], start=1):
    await page.keyboard.type(f"#{tag} ")
```

**TikTok 特殊处理** (参考 Python):
```python
# TikTok 支持 iframe 和普通容器两种页面结构
if await page.locator('iframe[data-tt="Upload_index_iframe"]').count():
    self.locator_base = page.frame_locator(Tk_Locator.tk_iframe)
else:
    self.locator_base = page.locator(Tk_Locator.default)
```

---

### 7. 定时发布逻辑

**状态**: ⏳ 待实现  
**文件**: `internal/service/schedule.go`  
**类型**: 【可直接移植】参考 `utils/files_times.py`

**Python 实现参考**:
```python
def generate_schedule_time_next_day(total_videos, videos_per_day=1, daily_times=None, start_days=0):
    if daily_times is None:
        daily_times = [6, 11, 14, 16, 22]  # 默认发布时间
    
    schedule = []
    current_time = datetime.now()
    
    for video in range(total_videos):
        day = video // videos_per_day + start_days + 1
        daily_video_index = video % videos_per_day
        hour = daily_times[daily_video_index]
        
        time_offset = timedelta(days=day, hours=hour - current_time.hour, minutes=-current_time.minute)
        timestamp = current_time + time_offset
        schedule.append(timestamp)
    
    return schedule
```

**Go 实现任务**:
- [ ] 移植 `GenerateScheduleTimes` 算法
- [ ] 支持自定义每日发布时间和数量
- [ ] 支持从第 N 天开始

---

### 8. 错误处理优化

**状态**: ⏳ 待实现  
**文件**: `internal/app/app.go`, 各 service 文件  
**类型**: 【可直接移植】参考 Python 错误处理

**Python 错误处理参考**:
```python
# 统一错误处理
try:
    await page.wait_for_url("https://creator.douyin.com/creator-micro/content/manage**", timeout=3000)
except:
    # 尝试处理封面问题
    await self.handle_auto_video_cover(page)
    
# 上传错误重试
if await page.locator('div.progress-div > div:has-text("上传失败")').count():
    await self.handle_upload_error(page)
```

**Go 实现任务**:
- [ ] 定义 AppError 结构体
- [ ] 统一错误码返回
- [ ] 添加更详细的错误信息
- [ ] 实现上传错误自动重试机制

---

## 低优先级任务

### 9. 并发控制优化

**状态**: ⏳ 待实现  
**文件**: `internal/service/upload.go`  
**类型**: 【可直接移植】Go 原生支持

**实现要点**:
- [ ] 实现上传任务队列
- [ ] 限制并发上传数量 (使用 channel 或 goroutine pool)
- [ ] 支持取消正在执行的任务

```go
// Go 实现示例
type UploadWorkerPool struct {
    workers int
    jobChan chan UploadJob
    wg      sync.WaitGroup
}
```

---

### 10. 配置持久化

**状态**: ⏳ 待实现  
**文件**: `internal/config/config.go`  
**类型**: 【可直接移植】参考 `conf.example.py`

**Python 配置参考**:
```python
BASE_DIR = Path(__file__).resolve().parent
LOCAL_CHROME_PATH = "C:\Program Files\Google\Chrome\Application\chrome.exe"
LOCAL_CHROME_HEADLESS = False
```

**Go 实现任务**:
- [ ] 支持从配置文件读取配置 (YAML/JSON)
- [ ] 支持运行时修改配置
- [ ] 配置项: Chrome 路径、上传并发数、超时时间、存储路径等

---

### 11. 日志系统增强

**状态**: ⏳ 待实现  
**文件**: `internal/utils/logger.go`  
**类型**: 【可直接移植】参考 `utils/log.py`

**Python 日志参考**:
```python
# 各平台独立日志
douyin_logger = create_logger('douyin', 'logs/douyin.log')
tencent_logger = create_logger('tencent', 'logs/tencent.log')
```

**Go 实现任务**:
- [ ] 支持日志级别设置
- [ ] 支持日志文件轮转
- [ ] 各平台独立日志文件
- [ ] 添加更多日志上下文信息

---

### 12. 反检测机制

**状态**: ⏳ 待实现  
**文件**: `internal/platform/*/uploader.go`  
**类型**: 【可直接移植】参考 `utils/base_social_media.py`

**Python 实现参考**:
```python
async def set_init_script(context):
    stealth_js_path = Path(BASE_DIR / "utils/stealth.min.js")
    await context.add_init_script(path=stealth_js_path)
    return context
```

**Go 实现任务**:
- [ ] 移植 `stealth.min.js` 反检测脚本
- [ ] 在创建 browser context 时注入脚本
- [ ] 隐藏 `navigator.webdriver` 属性

---

### 13. 数据库索引优化

**状态**: ⏳ 待实现  
**文件**: `internal/database/models.go`  
**类型**: 【可直接移植】参考 `db/createTable.py`

**实现要点**:
- [ ] 检查并优化数据库索引
- [ ] 添加必要的复合索引
- [ ] 参考现有 SQLite 表结构

---

### 14. 测试

**状态**: ⏳ 待实现  
**类型**: 【需参考 GitHub】Go 标准测试框架

**实现要点**:
- [ ] 单元测试 (使用 `testing` 包)
- [ ] 集成测试
- [ ] Mock Playwright 行为

---

## 依赖安装

```bash
# 已安装依赖
go get github.com/wailsapp/wails/v2/pkg/runtime
go get gorm.io/gorm
go get gorm.io/driver/sqlite
go get github.com/robfig/cron/v3

# 需要添加的依赖
go get github.com/playwright-community/playwright-go

# 可选依赖 (视频处理)
go get github.com/3d0c/gmf  # 或替代方案
```

---

## 实现优先级建议

### 第一阶段 (核心功能)
1. ✅ 事件发射机制 - 前后端通信基础
2. ✅ 文件选择对话框 - 用户交互基础
3. ✅ 抖音 Playwright 实现 - 核心上传逻辑
4. ✅ 账号登录功能 - Cookie 获取

### 第二阶段 (功能完善)
5. 定时发布逻辑
6. 视频元数据提取
7. 错误处理优化
8. 配置持久化

### 第三阶段 (平台扩展)
9. 其他平台实现 (视频号、快手、B站等)

### 第四阶段 (性能优化)
10. 并发控制优化
11. 日志系统增强
12. 反检测机制
13. 测试覆盖

---

## 参考资源

### Python 参考文件
- `uploader/douyin_uploader/main.py` - 抖音上传实现
- `uploader/tencent_uploader/main.py` - 视频号上传实现
- `uploader/ks_uploader/main.py` - 快手上传实现
- `utils/files_times.py` - 定时发布算法
- `utils/log.py` - 日志配置
- `db/createTable.py` - 数据库表结构

### Go 参考库
- [playwright-go](https://github.com/playwright-community/playwright-go)
- [Wails 文档](https://wails.io/docs/introduction)
- [GORM 文档](https://gorm.io/docs/)

---

**最后更新**: 2024-01-01
