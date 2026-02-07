import {
  GetScheduleConfig,
  UpdateScheduleConfig,
  GenerateScheduleTimes
} from '../../wailsjs/go/app/App'
import type { ScheduleConfig } from '../types'

// 获取定时配置
export async function getScheduleConfig(): Promise<ScheduleConfig | null> {
  try {
    const config = await GetScheduleConfig()
    return config as ScheduleConfig | null
  } catch (error) {
    console.error('获取定时配置失败:', error)
    throw error
  }
}

// 更新定时配置
export async function updateScheduleConfig(config: ScheduleConfig): Promise<void> {
  try {
    await UpdateScheduleConfig(config)
  } catch (error) {
    console.error('更新定时配置失败:', error)
    throw error
  }
}

// 生成定时时间
export async function generateScheduleTimes(videoCount: number): Promise<string[]> {
  try {
    const times = await GenerateScheduleTimes(videoCount)
    return times || []
  } catch (error) {
    console.error('生成定时时间失败:', error)
    throw error
  }
}
