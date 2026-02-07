import { GetLogs } from '../../wailsjs/go/app/App'
import type { SimpleLog, LogQuery } from '../types'

// 获取日志列表
export async function getLogs(query: LogQuery = {}): Promise<SimpleLog[]> {
  try {
    const result = await GetLogs({
      keyword: query.keyword || '',
      limit: query.limit || 100
    })
    return result || []
  } catch (error) {
    console.error('获取日志失败:', error)
    return []
  }
}
