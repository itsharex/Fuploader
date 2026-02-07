// 统一导出所有类型定义

export * from './account'
export * from './video'
export * from './task'
export * from './schedule'
export * from './log'
export * from './screenshot'

// 通用响应类型
export interface APIResponse<T = any> {
  success: boolean
  code: string
  message: string
  data?: T
}

export interface PageResult<T> {
  list: T[]
  total: number
  page: number
  pageSize: number
}

// 应用版本信息
export interface AppVersion {
  version: string
  buildTime: string
  goVersion: string
  wailsVersion: string
}
