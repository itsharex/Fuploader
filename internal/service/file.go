package service

import (
	"Fuploader/internal/config"
	"Fuploader/internal/database"
	"Fuploader/internal/utils"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"
)

type FileService struct {
	db *gorm.DB
}

func NewFileService(db *gorm.DB) *FileService {
	return &FileService{db: db}
}

func (s *FileService) GetVideos(ctx context.Context) ([]database.Video, error) {
	var videos []database.Video
	result := s.db.Find(&videos)
	if result.Error != nil {
		return nil, fmt.Errorf("query videos failed: %w", result.Error)
	}
	return videos, nil
}

func (s *FileService) AddVideo(ctx context.Context, filePath string) (*database.Video, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("stat file failed: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	if ext != ".mp4" && ext != ".mov" && ext != ".avi" {
		return nil, fmt.Errorf("unsupported video format: %s", ext)
	}

	filename := fmt.Sprintf("%d_%s", time.Now().Unix(), filepath.Base(filePath))
	dstPath := filepath.Join(config.Config.VideoPath, filename)

	srcFile, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open source file failed: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dstPath)
	if err != nil {
		return nil, fmt.Errorf("create dest file failed: %w", err)
	}
	defer dstFile.Close()

	if _, err = io.Copy(dstFile, srcFile); err != nil {
		return nil, fmt.Errorf("copy file failed: %w", err)
	}

	video := &database.Video{
		Filename:  filepath.Base(filePath),
		FilePath:  dstPath,
		FileSize:  info.Size(),
		CreatedAt: time.Now().Format(time.RFC3339),
	}

	result := s.db.Create(video)
	if result.Error != nil {
		return nil, fmt.Errorf("save video to db failed: %w", result.Error)
	}

	utils.Info(fmt.Sprintf("Video added: %s", video.Filename))
	return video, nil
}

func (s *FileService) UpdateVideo(ctx context.Context, video *database.Video) error {
	result := s.db.Save(video)
	if result.Error != nil {
		return fmt.Errorf("update video failed: %w", result.Error)
	}
	return nil
}

func (s *FileService) DeleteVideo(ctx context.Context, id int) error {
	var video database.Video
	result := s.db.First(&video, id)
	if result.Error != nil {
		return fmt.Errorf("video not found")
	}

	if err := os.Remove(video.FilePath); err != nil && !os.IsNotExist(err) {
		utils.Error(fmt.Sprintf("Remove video file failed: %v", err))
	}

	result = s.db.Delete(&video)
	if result.Error != nil {
		return fmt.Errorf("delete video from db failed: %w", result.Error)
	}

	return nil
}

// GetVideoByID 根据ID获取视频
func (s *FileService) GetVideoByID(ctx context.Context, id int) (*database.Video, error) {
	var video database.Video
	result := s.db.First(&video, id)
	if result.Error != nil {
		return nil, fmt.Errorf("video not found: %w", result.Error)
	}
	return &video, nil
}
