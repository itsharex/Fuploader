import type { PlatformType } from './account'
import type { Video } from './video'
import type { Account } from './account'

// 任务状态
export type TaskStatus = 'pending' | 'uploading' | 'success' | 'failed' | 'cancelled'

// 任务状态配置
export const TASK_STATUS_CONFIG: Record<TaskStatus, { label: string; type: 'info' | 'warning' | 'success' | 'danger' | 'default' }> = {
  pending: { label: '等待中', type: 'info' },
  uploading: { label: '上传中', type: 'warning' },
  success: { label: '成功', type: 'success' },
  failed: { label: '失败', type: 'danger' },
  cancelled: { label: '已取消', type: 'default' }
}

// 上传任务模型
export interface UploadTask {
  id: number
  videoId: number
  video: Video
  accountId: number
  account: Account
  platform: PlatformType
  status: TaskStatus
  progress: number
  scheduleTime?: string
  publishUrl?: string
  errorMsg?: string
  retryCount: number
  createdAt: string
  updatedAt: string
}

// 创建任务参数
export interface CreateTaskParams {
  videoId: number
  accountIds: number[]
  scheduleTime?: string | null
}

// 上传进度事件
export interface UploadProgressEvent {
  taskId: number
  platform: PlatformType
  progress: number
  message: string
}

// 上传完成事件
export interface UploadCompleteEvent {
  taskId: number
  platform: PlatformType
  publishUrl: string
  completedAt: string
}

// 上传错误事件
export interface UploadErrorEvent {
  taskId: number
  platform: PlatformType
  error: string
  canRetry: boolean
}

// 任务状态变更事件
export interface TaskStatusChangedEvent {
  taskId: number
  oldStatus: TaskStatus
  newStatus: TaskStatus
}
