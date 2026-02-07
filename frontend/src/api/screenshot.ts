import {
  GetScreenshotConfig,
  UpdateScreenshotConfig,
  GetScreenshots,
  DeleteScreenshot,
  BatchDeleteScreenshots,
  DeleteAllScreenshots,
  GetPlatformScreenshotStats,
  CleanOldScreenshots,
  OpenScreenshotDir
} from '../../wailsjs/go/app/App'
import { types } from '../../wailsjs/go/models'
import type {
  ScreenshotConfig,
  Screenshot,
  ScreenshotQuery,
  ScreenshotListResult,
  PlatformScreenshotConfig
} from '../types'

// 获取截图配置
export async function getScreenshotConfig(): Promise<ScreenshotConfig> {
  try {
    const config = await GetScreenshotConfig()
    return config as ScreenshotConfig
  } catch (error) {
    console.error('获取截图配置失败:', error)
    throw error
  }
}

// 更新截图配置
export async function updateScreenshotConfig(config: ScreenshotConfig): Promise<void> {
  try {
    await UpdateScreenshotConfig(config)
  } catch (error) {
    console.error('更新截图配置失败:', error)
    throw error
  }
}

// 获取截图列表
export async function getScreenshots(query: Partial<ScreenshotQuery> = {}): Promise<ScreenshotListResult> {
  try {
    // 使用 Go 生成的模型类，确保必需参数有默认值
    const params = new types.ScreenshotQuery({
      page: query.page ?? 1,
      pageSize: query.pageSize ?? 20,
      platform: query.platform,
      type: query.type,
      startDate: query.startDate,
      endDate: query.endDate
    })
    const result = await GetScreenshots(params)
    return result as ScreenshotListResult
  } catch (error) {
    console.error('获取截图列表失败:', error)
    throw error
  }
}

// 删除单个截图
export async function deleteScreenshot(id: string): Promise<void> {
  try {
    await DeleteScreenshot(id)
  } catch (error) {
    console.error('删除截图失败:', error)
    throw error
  }
}

// 批量删除截图
export async function batchDeleteScreenshots(ids: string[]): Promise<number> {
  try {
    const deleted = await BatchDeleteScreenshots(ids)
    return deleted as number
  } catch (error) {
    console.error('批量删除截图失败:', error)
    throw error
  }
}

// 删除所有截图
export async function deleteAllScreenshots(): Promise<number> {
  try {
    const deleted = await DeleteAllScreenshots()
    return deleted as number
  } catch (error) {
    console.error('删除所有截图失败:', error)
    throw error
  }
}

// 获取各平台截图统计
export async function getPlatformScreenshotStats(): Promise<PlatformScreenshotConfig[]> {
  try {
    const stats = await GetPlatformScreenshotStats()
    return stats as PlatformScreenshotConfig[]
  } catch (error) {
    console.error('获取平台截图统计失败:', error)
    throw error
  }
}

// 清理旧截图
export async function cleanOldScreenshots(): Promise<number> {
  try {
    const cleaned = await CleanOldScreenshots()
    return cleaned as number
  } catch (error) {
    console.error('清理旧截图失败:', error)
    throw error
  }
}

// 打开截图目录
export async function openScreenshotDir(platform: string = ''): Promise<void> {
  try {
    await OpenScreenshotDir(platform)
  } catch (error) {
    console.error('打开截图目录失败:', error)
    throw error
  }
}
