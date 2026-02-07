import { EventsOn, EventsOff } from '../../wailsjs/runtime'
import type {
  UploadProgressEvent,
  UploadCompleteEvent,
  UploadErrorEvent,
  LoginSuccessEvent,
  LoginErrorEvent,
  TaskStatusChangedEvent,
  AccountStatusChangedEvent
} from '../types/index'

// 事件名称常量
export const EVENTS = {
  UPLOAD_PROGRESS: 'upload:progress',
  UPLOAD_COMPLETE: 'upload:complete',
  UPLOAD_ERROR: 'upload:error',
  LOGIN_SUCCESS: 'login:success',
  LOGIN_ERROR: 'login:error',
  TASK_STATUS_CHANGED: 'task:statusChanged',
  ACCOUNT_STATUS_CHANGED: 'account:statusChanged'
} as const

// 事件处理器类型
export type EventHandlers = {
  onUploadProgress?: (event: UploadProgressEvent) => void
  onUploadComplete?: (event: UploadCompleteEvent) => void
  onUploadError?: (event: UploadErrorEvent) => void
  onLoginSuccess?: (event: LoginSuccessEvent) => void
  onLoginError?: (event: LoginErrorEvent) => void
  onTaskStatusChanged?: (event: TaskStatusChangedEvent) => void
  onAccountStatusChanged?: (event: AccountStatusChangedEvent) => void
}

// 设置事件监听
export function setupEventListeners(handlers: EventHandlers): () => void {
  const unsubscribers: (() => void)[] = []

  if (handlers.onUploadProgress) {
    unsubscribers.push(EventsOn(EVENTS.UPLOAD_PROGRESS, handlers.onUploadProgress))
  }

  if (handlers.onUploadComplete) {
    unsubscribers.push(EventsOn(EVENTS.UPLOAD_COMPLETE, handlers.onUploadComplete))
  }

  if (handlers.onUploadError) {
    unsubscribers.push(EventsOn(EVENTS.UPLOAD_ERROR, handlers.onUploadError))
  }

  if (handlers.onLoginSuccess) {
    unsubscribers.push(EventsOn(EVENTS.LOGIN_SUCCESS, handlers.onLoginSuccess))
  }

  if (handlers.onLoginError) {
    unsubscribers.push(EventsOn(EVENTS.LOGIN_ERROR, handlers.onLoginError))
  }

  if (handlers.onTaskStatusChanged) {
    unsubscribers.push(EventsOn(EVENTS.TASK_STATUS_CHANGED, handlers.onTaskStatusChanged))
  }

  if (handlers.onAccountStatusChanged) {
    unsubscribers.push(EventsOn(EVENTS.ACCOUNT_STATUS_CHANGED, handlers.onAccountStatusChanged))
  }

  // 返回取消订阅函数
  return () => {
    unsubscribers.forEach(unsubscribe => unsubscribe())
  }
}

// 单独的事件监听函数
export function onUploadProgress(handler: (event: UploadProgressEvent) => void): () => void {
  return EventsOn(EVENTS.UPLOAD_PROGRESS, handler)
}

export function onUploadComplete(handler: (event: UploadCompleteEvent) => void): () => void {
  return EventsOn(EVENTS.UPLOAD_COMPLETE, handler)
}

export function onUploadError(handler: (event: UploadErrorEvent) => void): () => void {
  return EventsOn(EVENTS.UPLOAD_ERROR, handler)
}

export function onLoginSuccess(handler: (event: LoginSuccessEvent) => void): () => void {
  return EventsOn(EVENTS.LOGIN_SUCCESS, handler)
}

export function onLoginError(handler: (event: LoginErrorEvent) => void): () => void {
  return EventsOn(EVENTS.LOGIN_ERROR, handler)
}

export function onTaskStatusChanged(handler: (event: TaskStatusChangedEvent) => void): () => void {
  return EventsOn(EVENTS.TASK_STATUS_CHANGED, handler)
}

export function onAccountStatusChanged(handler: (event: AccountStatusChangedEvent) => void): () => void {
  return EventsOn(EVENTS.ACCOUNT_STATUS_CHANGED, handler)
}
