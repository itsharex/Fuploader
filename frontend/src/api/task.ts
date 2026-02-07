import {
  CreateUploadTask,
  GetUploadTasks,
  CancelUploadTask,
  RetryUploadTask,
  DeleteUploadTask
} from '../../wailsjs/go/app/App'
import type { UploadTask, TaskStatus } from '../types'

// 创建上传任务
export async function createUploadTask(
  videoId: number,
  accountIds: number[],
  scheduleTime?: string | null,
  metadata?: Record<string, any>
): Promise<UploadTask[]> {
  try {
    const scheduleTimeParam = scheduleTime || null
    const metadataParam = metadata ? JSON.stringify(metadata) : null
    const tasks = await CreateUploadTask(videoId, accountIds, scheduleTimeParam, metadataParam)
    return (tasks || []) as UploadTask[]
  } catch (error) {
    console.error('创建上传任务失败:', error)
    throw error
  }
}

// 获取任务列表
export async function getUploadTasks(status?: TaskStatus | ''): Promise<UploadTask[]> {
  try {
    const tasks = await GetUploadTasks(status || '')
    return (tasks || []) as UploadTask[]
  } catch (error) {
    console.error('获取任务列表失败:', error)
    throw error
  }
}

// 取消任务
export async function cancelUploadTask(id: number): Promise<void> {
  try {
    await CancelUploadTask(id)
  } catch (error) {
    console.error('取消任务失败:', error)
    throw error
  }
}

// 重试任务
export async function retryUploadTask(id: number): Promise<void> {
  try {
    await RetryUploadTask(id)
  } catch (error) {
    console.error('重试任务失败:', error)
    throw error
  }
}

// 删除任务
export async function deleteUploadTask(id: number): Promise<void> {
  try {
    await DeleteUploadTask(id)
  } catch (error) {
    console.error('删除任务失败:', error)
    throw error
  }
}
