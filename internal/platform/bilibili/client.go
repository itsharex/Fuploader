package bilibili

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"

	"Fuploader/internal/utils"

	"github.com/imroc/req/v3"
	"github.com/panjf2000/ants/v2"
	"github.com/schollz/progressbar/v3"
	"github.com/tidwall/gjson"
)

// Client B站API客户端
type Client struct {
	cookiePath string
	videoPath  string
	title      string
	desc       string
	upType     int64
	coverPath  string
	tid        int64
	tag        string
	source     string
	threadNum  int

	cookie   string
	csrf     string
	client   *req.Client
	upVideo  *UpVideo
	partChan chan Part
	chunks   int64
}

// UpVideo 上传视频信息
type UpVideo struct {
	videoSize     int64
	videoName     string
	coverUrl      string
	auth          string
	uploadBaseUrl string
	biliFileName  string
	uploadId      string
	chunkSize     int64
	bizId         int64
}

var wg sync.WaitGroup

// NewClient 创建B站客户端
func NewClient(cookiePath string, threadNum int) (*Client, error) {
	loginInfo, err := os.ReadFile(cookiePath)
	if err != nil || len(loginInfo) == 0 {
		return nil, fmt.Errorf("cookie文件不存在，请先登录")
	}

	// 解析Playwright StorageState格式
	var storageState PlaywrightStorageState
	if err := json.Unmarshal(loginInfo, &storageState); err != nil {
		return nil, fmt.Errorf("解析cookie文件失败: %w", err)
	}

	if len(storageState.Cookies) == 0 {
		return nil, fmt.Errorf("cookie文件中没有cookies")
	}

	// 转换为cookie字符串
	cookie := convertPlaywrightCookies(storageState.Cookies)

	// 从cookies中提取csrf
	var csrf string
	for _, c := range storageState.Cookies {
		if c.Name == "bili_jct" {
			csrf = c.Value
			break
		}
	}

	client := req.C().SetCommonHeaders(map[string]string{
		"user-agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"cookie":     cookie,
		"Connection": "keep-alive",
	})

	// 验证登录状态
	resp, err := client.R().Get("https://api.bilibili.com/x/web-interface/nav")
	if err != nil {
		return nil, fmt.Errorf("验证登录状态失败: %w", err)
	}

	uname := gjson.GetBytes(resp.Bytes(), "data.uname").String()
	if uname == "" {
		return nil, fmt.Errorf("cookie已失效，请重新登录")
	}

	utils.Info(fmt.Sprintf("[-] B站 - %s 登录成功", uname))

	return &Client{
		cookiePath: cookiePath,
		cookie:     cookie,
		csrf:       csrf,
		client:     client,
		upVideo:    &UpVideo{},
		threadNum:  threadNum,
	}, nil
}

// SetVideoInfo 设置视频信息
func (c *Client) SetVideoInfo(tid, upType int64, videoPath, coverPath, title, desc, tag, source string) *Client {
	c.videoPath = videoPath
	c.title = title
	c.desc = desc
	c.upType = upType
	c.tid = tid
	c.tag = tag
	c.source = source
	c.coverPath = coverPath
	c.upVideo.videoName = path.Base(videoPath)
	c.upVideo.videoSize = c.getVideoSize()
	return c
}

// getVideoSize 获取视频大小
func (c *Client) getVideoSize() int64 {
	file, err := os.Open(c.videoPath)
	if err != nil {
		return 0
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return 0
	}
	return fileInfo.Size()
}

// UploadCover 上传封面
func (c *Client) UploadCover() (string, error) {
	if c.coverPath == "" {
		return "", nil
	}

	data, err := os.ReadFile(c.coverPath)
	if err != nil {
		return "", fmt.Errorf("读取封面文件失败: %w", err)
	}

	var base64Encoding string
	mimeType := http.DetectContentType(data)
	switch mimeType {
	case "image/jpeg", "image/jpg":
		base64Encoding = "data:image/jpeg;base64,"
	case "image/png":
		base64Encoding = "data:image/png;base64,"
	case "image/gif":
		base64Encoding = "data:image/gif;base64,"
	default:
		return "", fmt.Errorf("不支持的图片格式: %s", mimeType)
	}

	base64Encoding += base64.StdEncoding.EncodeToString(data)

	var coverInfo CoverInfo
	_, err = c.client.R().
		SetFormDataFromValues(url.Values{
			"cover": {base64Encoding},
			"csrf":  {c.csrf},
		}).
		SetHeaders(map[string]string{
			"Referer": "https://member.bilibili.com/video/upload.html",
			"Origin":  "https://member.bilibili.com",
		}).
		SetResult(&coverInfo).
		Post("https://member.bilibili.com/x/vu/web/cover/up")

	if err != nil {
		return "", fmt.Errorf("上传封面失败: %w", err)
	}

	c.upVideo.coverUrl = coverInfo.Data.Url
	return coverInfo.Data.Url, nil
}

// Upload 执行上传
func (c *Client) Upload() error {
	// 1. 预上传
	var preUpInfo PreUpInfo
	_, err := c.client.R().
		SetQueryParams(map[string]string{
			"probe_version": "20211012",
			"upcdn":         "bda2",
			"zone":          "cs",
			"name":          c.upVideo.videoName,
			"r":             "upos",
			"profile":       "ugcfx/bup",
			"ssl":           "0",
			"version":       "2.10.4.0",
			"build":         "2100400",
			"size":          strconv.FormatInt(c.upVideo.videoSize, 10),
			"webVersion":    "2.0.0",
		}).
		SetHeaders(map[string]string{
			"Referer": "https://member.bilibili.com/video/upload.html",
		}).
		SetResult(&preUpInfo).
		Get("https://member.bilibili.com/preupload")

	if err != nil {
		return fmt.Errorf("预上传失败: %w", err)
	}

	c.upVideo.uploadBaseUrl = fmt.Sprintf("https:%s/%s", preUpInfo.Endpoint, strings.Split(preUpInfo.UposUri, "//")[1])
	c.upVideo.biliFileName = strings.Split(strings.Split(strings.Split(preUpInfo.UposUri, "//")[1], "/")[1], ".")[0]
	c.upVideo.chunkSize = preUpInfo.ChunkSize
	c.upVideo.auth = preUpInfo.Auth
	c.upVideo.bizId = preUpInfo.BizId

	utils.Info(fmt.Sprintf("[-] B站 - 预上传成功，分片大小: %d", c.upVideo.chunkSize))

	// 2. 上传视频
	if err := c.uploadVideo(); err != nil {
		return fmt.Errorf("上传视频失败: %w", err)
	}

	// 3. 提交视频
	if err := c.submitVideo(preUpInfo.BizId); err != nil {
		return fmt.Errorf("提交视频失败: %w", err)
	}

	return nil
}

// uploadVideo 上传视频文件
func (c *Client) uploadVideo() error {
	// 初始化上传
	var upInfo UpInfo
	c.client.SetCommonHeader("X-Upos-Auth", c.upVideo.auth)
	_, err := c.client.R().
		SetQueryParams(map[string]string{
			"uploads":       "",
			"output":        "json",
			"profile":       "ugcfx/bup",
			"filesize":      strconv.FormatInt(c.upVideo.videoSize, 10),
			"partsize":      strconv.FormatInt(c.upVideo.chunkSize, 10),
			"biz_id":        strconv.FormatInt(c.upVideo.bizId, 10),
			"meta_upos_uri": c.getMetaUposUri(),
		}).
		SetResult(&upInfo).
		Post(c.upVideo.uploadBaseUrl)

	if err != nil {
		return err
	}

	c.upVideo.uploadId = upInfo.UploadId
	c.chunks = int64(math.Ceil(float64(c.upVideo.videoSize) / float64(c.upVideo.chunkSize)))

	utils.Info(fmt.Sprintf("[-] B站 - 开始上传，总分片数: %d", c.chunks))

	// 创建进度条
	bar := progressbar.NewOptions(
		int(c.upVideo.videoSize/1024/1024),
		progressbar.OptionSetDescription("上传视频中..."),
		progressbar.OptionSetWidth(50),
		progressbar.OptionShowBytes(true),
		progressbar.OptionShowCount(),
		progressbar.OptionSetPredictTime(true),
	)

	// 创建分片通道
	c.partChan = make(chan Part, c.chunks)
	go func() {
		for p := range c.partChan {
			_ = p
		}
	}()

	// 创建协程池
	pool, err := ants.NewPool(c.threadNum)
	if err != nil {
		return err
	}
	defer pool.Release()

	// 打开文件
	file, err := os.Open(c.videoPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 分片上传
	chunk := 0
	start := 0
	end := 0

	for {
		buf := make([]byte, c.upVideo.chunkSize)
		size, err := file.Read(buf)
		if err != nil && err != io.EOF {
			break
		}
		buf = buf[:size]

		if size > 0 {
			wg.Add(1)
			end += size
			chunkData := make([]byte, size)
			copy(chunkData, buf)

			_ = pool.Submit(func() {
				defer wg.Done()
				c.uploadChunk(chunk, start, end, size, chunkData, bar)
			})

			start += size
			chunk++
		}

		if err == io.EOF {
			break
		}
	}

	wg.Wait()
	close(c.partChan)
	bar.Finish()

	utils.Info("[-] B站 - 视频上传完成，确认分片...")

	// 确认分片
	return c.confirmChunks()
}

// uploadChunk 上传单个分片
func (c *Client) uploadChunk(chunk, start, end, size int, buf []byte, bar *progressbar.ProgressBar) {
	resp, err := c.client.R().
		SetHeaders(map[string]string{
			"Content-Type":   "application/octet-stream",
			"Content-Length": strconv.Itoa(size),
		}).
		SetQueryParams(map[string]string{
			"partNumber": strconv.Itoa(chunk + 1),
			"uploadId":   c.upVideo.uploadId,
			"chunk":      strconv.Itoa(chunk),
			"chunks":     strconv.Itoa(int(c.chunks)),
			"size":       strconv.Itoa(size),
			"start":      strconv.Itoa(start),
			"end":        strconv.Itoa(end),
			"total":      strconv.FormatInt(c.upVideo.videoSize, 10),
		}).
		SetBodyBytes(buf).
		SetRetryCount(3).
		Put(c.upVideo.uploadBaseUrl)

	if err != nil {
		utils.Warn(fmt.Sprintf("[-] B站 - 分片 %d 上传失败: %v", chunk, err))
		return
	}

	if resp.StatusCode != 200 {
		utils.Warn(fmt.Sprintf("[-] B站 - 分片 %d 上传失败，状态码: %d", chunk, resp.StatusCode))
		return
	}

	bar.Add(len(buf) / 1024 / 1024)
	c.partChan <- Part{
		PartNumber: int64(chunk + 1),
		ETag:       "etag",
	}
}

// confirmChunks 确认分片
func (c *Client) confirmChunks() error {
	// 收集所有分片信息
	parts := make([]Part, c.chunks)
	for i := int64(0); i < c.chunks; i++ {
		parts[i] = Part{
			PartNumber: i + 1,
			ETag:       "etag",
		}
	}

	reqJson := ReqJson{Parts: parts}
	jsonData, _ := json.Marshal(&reqJson)

	_, err := c.client.R().
		SetHeaders(map[string]string{
			"Content-Type": "application/json",
			"Origin":       "https://member.bilibili.com",
			"Referer":      "https://member.bilibili.com/",
		}).
		SetQueryParams(map[string]string{
			"output":   "json",
			"profile":  "ugcfx/bup",
			"name":     c.upVideo.videoName,
			"uploadId": c.upVideo.uploadId,
			"biz_id":   strconv.FormatInt(c.upVideo.bizId, 10),
		}).
		SetBodyBytes(jsonData).
		SetRetryCount(5).
		Post(c.upVideo.uploadBaseUrl)

	return err
}

// submitVideo 提交视频
func (c *Client) submitVideo(bizId int64) error {
	addReq := AddReqJson{
		Copyright:    c.upType,
		Cover:        c.upVideo.coverUrl,
		Title:        c.title,
		Tid:          c.tid,
		Tag:          c.tag,
		DescFormatId: 0,
		Desc:         c.desc,
		Source:       c.source,
		Dynamic:      "",
		Interactive:  0,
		Videos: []Video{
			{
				Filename: c.upVideo.biliFileName,
				Title:    "",
				Desc:     "",
			},
		},
		ActReserveCreate: 0,
		NoDisturbance:    0,
		NoReprint:        1,
		Subtitle: Subtitle{
			Open: 0,
			Lan:  "",
		},
		Dolby:         0,
		LosslessMusic: 0,
		Csrf:          c.csrf,
	}

	// 调试日志：打印实际发送的标题
	utils.Info(fmt.Sprintf("[-] B站调试 - 外层Title长度: %d, 内容: %s", len(c.title), c.title))
	if len(addReq.Videos) > 0 {
		utils.Info(fmt.Sprintf("[-] B站调试 - Videos[0].Title长度: %d, 内容: %s", len(addReq.Videos[0].Title), addReq.Videos[0].Title))
	}

	resp, err := c.client.R().
		SetQueryParams(map[string]string{
			"csrf": c.csrf,
		}).
		SetHeaders(map[string]string{
			"Referer": "https://member.bilibili.com/video/upload.html",
			"Origin":  "https://member.bilibili.com",
		}).
		SetBodyJsonMarshal(addReq).
		Post("https://member.bilibili.com/x/vu/web/add/v3")

	if err != nil {
		return err
	}

	utils.Info(fmt.Sprintf("[-] B站 - 提交视频响应: %s", resp.String()))

	// 检查响应
	code := gjson.GetBytes(resp.Bytes(), "code").Int()
	if code != 0 {
		message := gjson.GetBytes(resp.Bytes(), "message").String()
		return fmt.Errorf("提交视频失败: %s", message)
	}

	return nil
}

// getMetaUposUri 获取meta upos uri
func (c *Client) getMetaUposUri() string {
	var metaUposUri PreUpInfo
	c.client.R().
		SetQueryParams(map[string]string{
			"name":       "file_meta.txt",
			"size":       "2000",
			"r":          "upos",
			"profile":    "fxmeta/bup",
			"ssl":        "0",
			"version":    "2.10.4",
			"build":      "2100400",
			"webVersion": "2.0.0",
		}).
		SetResult(&metaUposUri).
		Get("https://member.bilibili.com/preupload")
	return metaUposUri.UposUri
}

// ValidateCookie 验证Cookie是否有效
func ValidateCookie(cookiePath string) (bool, string, error) {
	utils.Info(fmt.Sprintf("[-] B站验证 - 开始验证，cookie路径: %s", cookiePath))

	loginInfo, err := os.ReadFile(cookiePath)
	if err != nil || len(loginInfo) == 0 {
		utils.Error(fmt.Sprintf("[-] B站验证 - 读取cookie文件失败: %v", err))
		return false, "", fmt.Errorf("cookie文件不存在")
	}

	utils.Info(fmt.Sprintf("[-] B站验证 - 读取到cookie文件，大小: %d 字节", len(loginInfo)))

	// 解析Playwright StorageState格式
	var storageState PlaywrightStorageState
	if err := json.Unmarshal(loginInfo, &storageState); err != nil {
		utils.Error(fmt.Sprintf("[-] B站验证 - 解析cookie文件失败: %v", err))
		return false, "", fmt.Errorf("解析cookie文件失败: %w", err)
	}

	if len(storageState.Cookies) == 0 {
		utils.Error("[-] B站验证 - cookie文件中没有cookies")
		return false, "", fmt.Errorf("cookie文件中没有cookies")
	}

	utils.Info(fmt.Sprintf("[-] B站验证 - 共 %d 个cookie", len(storageState.Cookies)))
	cookie := convertPlaywrightCookies(storageState.Cookies)
	utils.Info(fmt.Sprintf("[-] B站验证 - 转换后的cookie字符串长度: %d", len(cookie)))

	return validateCookieString(cookie)
}

// PlaywrightCookie Playwright的cookie格式
type PlaywrightCookie struct {
	Name     string  `json:"name"`
	Value    string  `json:"value"`
	Domain   string  `json:"domain"`
	Path     string  `json:"path"`
	Expires  float64 `json:"expires"`
	HttpOnly bool    `json:"httpOnly"`
	Secure   bool    `json:"secure"`
	SameSite string  `json:"sameSite"`
}

// PlaywrightStorageState Playwright StorageState格式
type PlaywrightStorageState struct {
	Cookies []PlaywrightCookie `json:"cookies"`
	Origins []struct {
		Origin       string `json:"origin"`
		LocalStorage []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"localStorage"`
	} `json:"origins"`
}

// convertPlaywrightCookies 转换Playwright cookie格式为字符串
func convertPlaywrightCookies(cookies []PlaywrightCookie) string {
	var parts []string
	for _, c := range cookies {
		parts = append(parts, fmt.Sprintf("%s=%s", c.Name, c.Value))
	}
	return strings.Join(parts, "; ")
}

// validateCookieString 验证cookie字符串
func validateCookieString(cookie string) (bool, string, error) {
	utils.Info("[-] B站验证 - 开始请求API验证cookie")

	client := req.C().SetCommonHeaders(map[string]string{
		"user-agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"cookie":     cookie,
		"Referer":    "https://www.bilibili.com",
	})

	resp, err := client.R().Get("https://api.bilibili.com/x/web-interface/nav")
	if err != nil {
		utils.Error(fmt.Sprintf("[-] B站验证 - API请求失败: %v", err))
		return false, "", err
	}

	respBody := resp.Bytes()
	utils.Info(fmt.Sprintf("[-] B站验证 - API响应: %s", string(respBody)))

	isLogin := gjson.GetBytes(respBody, "data.isLogin").Bool()
	uname := gjson.GetBytes(respBody, "data.uname").String()
	code := gjson.GetBytes(respBody, "code").Int()
	message := gjson.GetBytes(respBody, "message").String()

	utils.Info(fmt.Sprintf("[-] B站验证 - 解析结果: code=%d, message=%s, isLogin=%v, uname=%s", code, message, isLogin, uname))

	return isLogin, uname, nil
}
