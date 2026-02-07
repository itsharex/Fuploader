import { createPinia } from 'pinia'

// 导出 pinia 实例
export const pinia = createPinia()

// 导出所有 store
export * from './account'
export * from './video'
export * from './task'
export * from './schedule'
export * from './theme'
export * from './screenshot'
