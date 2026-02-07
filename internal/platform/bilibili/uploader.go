package bilibili

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"Fuploader/internal/platform/browser"
	"Fuploader/internal/platform/uploader"
	"Fuploader/internal/types"
	"Fuploader/internal/utils"

	"github.com/playwright-community/playwright-go"
)

// browserPool 全局浏览器池实例（用于登录）
var browserPool *browser.Pool

// initBrowserPool 初始化浏览器池
func initBrowserPool() {
	if browserPool == nil {
		browserPool = browser.NewPool(2, 5)
	}
}

// Uploader B站上传器
type Uploader struct {
	*uploader.Base
	threadNum int
}

// NewUploader 创建上传器
func NewUploader(cookiePath string) *Uploader {
	initBrowserPool()
	return &Uploader{
		Base:      uploader.NewBase("bilibili", cookiePath, browserPool),
		threadNum: 4, // 默认4线程
	}
}

// Platform 返回平台名称
func (u *Uploader) Platform() string {
	return "bilibili"
}

// ValidateCookie 验证Cookie是否有效
func (u *Uploader) ValidateCookie(ctx context.Context) (bool, error) {
	isValid, uname, err := ValidateCookie(u.GetCookiePath())
	if err != nil {
		utils.Warn(fmt.Sprintf("[-] B站验证 - %v", err))
		return false, nil
	}

	if isValid {
		utils.Info(fmt.Sprintf("[-] B站验证 - %s 登录有效", uname))
	} else {
		utils.Info("[-] B站验证 - Cookie已失效")
	}

	return isValid, nil
}

// Upload 上传视频
func (u *Uploader) Upload(ctx context.Context, task *types.VideoTask) error {
	utils.Info(fmt.Sprintf("[+] B站上传 - 开始上传任务: %s", task.VideoPath))

	// 检查视频文件
	if _, err := os.Stat(task.VideoPath); err != nil {
		return fmt.Errorf("视频文件不存在: %w", err)
	}

	// 创建客户端
	client, err := NewClient(u.GetCookiePath(), u.threadNum)
	if err != nil {
		return fmt.Errorf("创建客户端失败: %w", err)
	}

	// 设置视频信息
	// 默认分区：生活-日常 (160)，原创
	tid := int64(160)
	upType := int64(1)
	source := ""

	// 解析标签
	tag := strings.Join(task.Tags, ",")

	// 设置视频信息并上传封面
	client.SetVideoInfo(tid, upType, task.VideoPath, task.Thumbnail, task.Title, task.Description, tag, source)

	// 上传封面
	if task.Thumbnail != "" {
		_, err = client.UploadCover()
		if err != nil {
			utils.Warn(fmt.Sprintf("[-] B站上传 - 封面上传失败: %v", err))
		}
	}

	// 执行上传
	if err := client.Upload(); err != nil {
		return fmt.Errorf("上传失败: %w", err)
	}

	utils.Info("[+] B站上传 - 视频上传成功")
	return nil
}

// Login 登录 - 使用浏览器扫码登录
func (u *Uploader) Login() error {
	ctx := context.Background()

	// 从浏览器池获取上下文
	browserCtx, err := browserPool.GetContext(ctx, u.GetCookiePath(), u.GetContextOptions())
	if err != nil {
		return fmt.Errorf("get browser context failed: %w", err)
	}
	defer browserCtx.Release()

	page, err := browserCtx.GetPage()
	if err != nil {
		return fmt.Errorf("get page failed: %w", err)
	}

	// 访问登录页面
	utils.Info("[-] 正在打开 B站 登录页面...")
	if _, err := page.Goto("https://passport.bilibili.com/login", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	}); err != nil {
		return fmt.Errorf("goto login page failed: %w", err)
	}

	// 等待页面完全加载
	if err := browserCtx.WaitForPageLoad(); err != nil {
		utils.Warn(fmt.Sprintf("[-] 等待页面加载警告: %v", err))
	}
	time.Sleep(3 * time.Second)

	utils.Info("[-] 请在浏览器窗口中使用 B站 APP 扫描二维码完成登录")

	// 检测登录成功
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("login timeout")
		case <-ticker.C:
			// 检查页面是否已关闭
			if browserCtx.IsPageClosed() {
				utils.Error("[-] 页面已被关闭，登录中断")
				return fmt.Errorf("page closed by user")
			}

			// 检查是否已登录
			phoneLoginCount, _ := page.GetByText("手机号登录").Count()
			qrLoginCount, _ := page.GetByText("扫码登录").Count()
			loginBtnCount, _ := page.GetByText("登录").Count()

			if phoneLoginCount == 0 && qrLoginCount == 0 && loginBtnCount == 0 {
				// 再检查一下是否有用户头像
				avatar, _ := page.Locator(".header-avatar, .user-avatar").Count()
				if avatar > 0 {
					utils.Info("[-] 登录成功，检测到用户头像")
					// 保存Cookie
					return u.saveCookiesFromPage(page)
				}
			}

			// 检查是否跳转到主页
			url := page.URL()
			if url == "https://www.bilibili.com/" || url == "https://www.bilibili.com" {
				utils.Info("[-] 登录成功，已跳转到主页")
				// 保存Cookie
				return u.saveCookiesFromPage(page)
			}
		}
	}
}

// saveCookiesFromPage 从页面保存Cookie - 保存为Playwright StorageState格式
func (u *Uploader) saveCookiesFromPage(page playwright.Page) error {
	// 获取StorageState（包含cookies和origins）
	storageState, err := page.Context().StorageState()
	if err != nil {
		return fmt.Errorf("get storage state failed: %w", err)
	}

	// 保存到文件
	data, err := json.Marshal(storageState)
	if err != nil {
		return fmt.Errorf("marshal storage state failed: %w", err)
	}

	if err := os.WriteFile(u.GetCookiePath(), data, 0644); err != nil {
		return fmt.Errorf("write cookie file failed: %w", err)
	}

	utils.Info(fmt.Sprintf("[-] Cookie已保存到: %s (StorageState格式)", u.GetCookiePath()))
	return nil
}
