import {
  GetAppVersion,
  OpenDirectory
} from '../../wailsjs/go/app/App'
import type { AppVersion } from '../types'

// 获取应用版本
export async function getAppVersion(): Promise<AppVersion> {
  try {
    const version = await GetAppVersion()
    return version as AppVersion
  } catch (error) {
    console.error('获取应用版本失败:', error)
    throw error
  }
}

// 打开目录
export async function openDirectory(path: string): Promise<void> {
  try {
    await OpenDirectory(path)
  } catch (error) {
    console.error('打开目录失败:', error)
    throw error
  }
}
