import {
  GetVideos,
  AddVideo,
  UpdateVideo,
  DeleteVideo,
  SelectVideoFile
} from '../../wailsjs/go/app/App'
import type { Video } from '../types'

// 获取视频列表
export async function getVideos(): Promise<Video[]> {
  try {
    const videos = await GetVideos()
    return (videos || []) as Video[]
  } catch (error) {
    console.error('获取视频列表失败:', error)
    throw error
  }
}

// 添加视频
export async function addVideo(filePath: string): Promise<Video> {
  try {
    const video = await AddVideo(filePath)
    return video as Video
  } catch (error) {
    console.error('添加视频失败:', error)
    throw error
  }
}

// 更新视频
export async function updateVideo(video: Video): Promise<void> {
  try {
    // 转换为 Wails 模型类型（处理可选字段）
    const wailsVideo = {
      ...video,
      title: video.title || '',
      description: video.description || '',
      tags: video.tags || [],
      thumbnail: video.thumbnail || ''
    }
    await UpdateVideo(wailsVideo as any)
  } catch (error) {
    console.error('更新视频失败:', error)
    throw error
  }
}

// 删除视频
export async function deleteVideo(id: number): Promise<void> {
  try {
    await DeleteVideo(id)
  } catch (error) {
    console.error('删除视频失败:', error)
    throw error
  }
}

// 选择视频文件
export async function selectVideoFile(): Promise<string> {
  try {
    const filePath = await SelectVideoFile()
    return filePath || ''
  } catch (error) {
    console.error('选择视频文件失败:', error)
    throw error
  }
}
