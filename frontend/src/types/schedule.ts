// 定时配置模型
export interface ScheduleConfig {
  id: number
  videosPerDay: number
  dailyTimes: string[]
  startDays: number
  timeZone: string
}

// 定时配置表单
export interface ScheduleConfigForm {
  videosPerDay: number
  dailyTimes: string[]
  startDays: number
  timeZone: string
}
