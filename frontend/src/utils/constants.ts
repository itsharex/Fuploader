// 常量定义

// 平台列表
export const PLATFORMS = [
  { value: 'douyin', label: '抖音', color: '#000000' },
  { value: 'tencent', label: '视频号', color: '#07C160' },
  { value: 'kuaishou', label: '快手', color: '#FF6600' },
  { value: 'tiktok', label: 'TikTok', color: '#000000' },
  { value: 'bilibili', label: 'Bilibili', color: '#FB7299' },
  { value: 'xiaohongshu', label: '小红书', color: '#FF2442' },
  { value: 'baijiahao', label: '百家号', color: '#2932E1' }
] as const

// 账号状态
export const ACCOUNT_STATUS = {
  INVALID: 0,
  VALID: 1,
  EXPIRED: 2
} as const

export const ACCOUNT_STATUS_LABELS: Record<number, { label: string; type: 'info' | 'success' | 'warning' | 'danger' }> = {
  0: { label: '无效', type: 'danger' },
  1: { label: '有效', type: 'success' },
  2: { label: '已过期', type: 'warning' }
}

// 任务状态
export const TASK_STATUS = {
  PENDING: 'pending',
  UPLOADING: 'uploading',
  SUCCESS: 'success',
  FAILED: 'failed',
  CANCELLED: 'cancelled'
} as const

export const TASK_STATUS_LABELS: Record<string, { label: string; type: 'info' | 'warning' | 'success' | 'danger' | 'default' }> = {
  pending: { label: '等待中', type: 'info' },
  uploading: { label: '上传中', type: 'warning' },
  success: { label: '成功', type: 'success' },
  failed: { label: '失败', type: 'danger' },
  cancelled: { label: '已取消', type: 'default' }
}

// 应用配置
export const APP_CONFIG = {
  NAME: 'Fuploader',
  VERSION: '1.0.0',
  STORAGE_KEYS: {
    SETTINGS: 'fuploader_settings',
    RECENT_VIDEOS: 'fuploader_recent_videos'
  }
} as const

// 分页配置
export const PAGINATION = {
  DEFAULT_PAGE: 1,
  DEFAULT_PAGE_SIZE: 10,
  PAGE_SIZE_OPTIONS: [10, 20, 50, 100]
} as const
