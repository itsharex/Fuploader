package utils

import (
	"Fuploader/internal/config"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/playwright-community/playwright-go"
)

// LogServiceInterface 日志服务接口（避免循环依赖）
type LogServiceInterface interface {
	Add(message string)
}

type Logger struct {
	file       *os.File
	logService LogServiceInterface
	mutex      sync.Mutex
}

var defaultLogger *Logger

func InitLogger() error {
	logPath := filepath.Join(config.Config.LogPath, fmt.Sprintf("app_%s.log", time.Now().Format("20060102")))
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	defaultLogger = &Logger{file: file}
	return nil
}

func GetLogger() *Logger {
	if defaultLogger == nil {
		_ = InitLogger()
	}
	return defaultLogger
}

// SetLogService 设置日志服务，用于前端日志输出
func SetLogService(service LogServiceInterface) {
	GetLogger().mutex.Lock()
	defer GetLogger().mutex.Unlock()
	GetLogger().logService = service
}

func (l *Logger) log(level string, msg string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	line := fmt.Sprintf("[%s] [%s] %s\n", timestamp, level, msg)

	l.mutex.Lock()
	defer l.mutex.Unlock()

	// 写入文件
	_, _ = l.file.WriteString(line)

	// 同时输出到前端
	if l.logService != nil {
		l.logService.Add(fmt.Sprintf("[%s] %s", level, msg))
	}
}

func (l *Logger) Info(msg string) {
	l.log("INFO", msg)
}

func (l *Logger) Error(msg string) {
	l.log("ERROR", msg)
}

func (l *Logger) Warn(msg string) {
	l.log("WARN", msg)
}

func (l *Logger) Debug(msg string) {
	l.log("DEBUG", msg)
}

func (l *Logger) Success(msg string) {
	l.log("SUCCESS", msg)
}

func Info(msg string) {
	GetLogger().Info(msg)
}

func Error(msg string) {
	GetLogger().Error(msg)
}

func Warn(msg string) {
	GetLogger().Warn(msg)
}

func Debug(msg string) {
	GetLogger().Debug(msg)
}

func Success(msg string) {
	GetLogger().Success(msg)
}

// Screenshot 截图并保存到日志目录
func Screenshot(page playwright.Page, name string) error {
	screenshotPath := filepath.Join(config.Config.LogPath, fmt.Sprintf("screenshot_%s_%s.png", time.Now().Format("20060102_150405"), name))
	_, err := page.Screenshot(playwright.PageScreenshotOptions{
		Path:     playwright.String(screenshotPath),
		FullPage: playwright.Bool(true),
	})
	if err != nil {
		Error(fmt.Sprintf("截图失败: %v", err))
		return err
	}
	Info(fmt.Sprintf("截图已保存: %s", screenshotPath))
	return nil
}
