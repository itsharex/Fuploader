// 截图配置
export interface ScreenshotConfig {
  enabled: boolean
  globalDir: string
  platformDirs: Record<string, string>
  autoClean: boolean
  maxAgeDays: number
  maxSizeMB: number
}

// 截图信息
export interface Screenshot {
  id: string
  filename: string
  platform: string
  type: string
  size: number
  createdAt: string
  path: string
}

// 截图查询参数
export interface ScreenshotQuery {
  platform?: string
  type?: string
  startDate?: string
  endDate?: string
  page?: number
  pageSize?: number
}

// 截图列表结果
export interface ScreenshotListResult {
  list: Screenshot[]
  total: number
  page: number
  pageSize: number
  totalSize: number
  platformStats: Record<string, number>
}

// 平台截图配置
export interface PlatformScreenshotConfig {
  platform: string
  name: string
  dir: string
  screenshotCount: number
}

// 默认截图配置
export const DEFAULT_SCREENSHOT_CONFIG: ScreenshotConfig = {
  enabled: false,
  globalDir: './screenshots',
  platformDirs: {
    xiaohongshu: './screenshots/xiaohongshu',
    tencent: './screenshots/tencent',
    douyin: './screenshots/douyin',
    kuaishou: './screenshots/kuaishou',
    baijiahao: './screenshots/baijiahao',
    tiktok: './screenshots/tiktok'
  },
  autoClean: false,
  maxAgeDays: 30,
  maxSizeMB: 500
}

// 截图类型标签
export const SCREENSHOT_TYPE_LABELS: Record<string, string> = {
  navigate_complete: '导航完成',
  upload_timeout: '上传超时',
  upload_success: '上传成功',
  upload_error: '上传错误',
  publish_timeout: '发布超时',
  publish_success: '发布成功',
  publish_error: '发布错误',
  publishing: '发布中',
  draft_timeout: '草稿超时',
  draft_success: '草稿成功',
  saving_draft: '保存草稿',
  cover_not_found: '封面未找到',
  error: '错误截图',
  unknown: '未知类型'
}

// 平台名称映射
export const PLATFORM_NAME_MAP: Record<string, string> = {
  xiaohongshu: '小红书',
  tencent: '视频号',
  douyin: '抖音',
  kuaishou: '快手',
  baijiahao: '百家号',
  tiktok: 'TikTok',
  unknown: '未知平台'
}
