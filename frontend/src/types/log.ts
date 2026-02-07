// 日志类型定义

// 简洁日志条目
export interface SimpleLog {
  date: string    // 日期，格式：2006/1/2
  time: string    // 时间，格式：15:04:05
  message: string // 日志内容
}

// 日志查询参数
export interface LogQuery {
  keyword?: string  // 关键词搜索
  limit?: number    // 返回条数，默认100
}
