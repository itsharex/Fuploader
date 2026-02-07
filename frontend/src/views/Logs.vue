<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { getLogs } from '../api'
import type { SimpleLog, LogQuery } from '../types'

const logs = ref<SimpleLog[]>([])
const loading = ref(false)
const keyword = ref('')
const autoRefresh = ref(false)
let refreshTimer: number | null = null

// 获取日志
async function fetchLogs() {
  loading.value = true
  const query: LogQuery = {
    keyword: keyword.value,
    limit: 200
  }
  logs.value = await getLogs(query)
  loading.value = false
}

// 搜索日志
function handleSearch() {
  fetchLogs()
}

// 清空搜索
function clearSearch() {
  keyword.value = ''
  fetchLogs()
}

// 切换自动刷新
function toggleAutoRefresh() {
  autoRefresh.value = !autoRefresh.value
  if (autoRefresh.value) {
    refreshTimer = window.setInterval(fetchLogs, 3000)
  } else {
    if (refreshTimer) {
      clearInterval(refreshTimer)
      refreshTimer = null
    }
  }
}

// 格式化日志消息（高亮关键词）
function formatMessage(message: string) {
  if (!keyword.value) return message
  const regex = new RegExp(`(${keyword.value})`, 'gi')
  return message.replace(regex, '<mark>$1</mark>')
}

// 按日期分组日志
const groupedLogs = computed(() => {
  const groups: Record<string, SimpleLog[]> = {}
  logs.value.forEach(log => {
    if (!groups[log.date]) {
      groups[log.date] = []
    }
    groups[log.date].push(log)
  })
  return groups
})

// 获取日志级别样式
function getLogLevelClass(message: string): string {
  const lowerMsg = message.toLowerCase()
  if (lowerMsg.includes('error') || lowerMsg.includes('失败') || lowerMsg.includes('错误')) {
    return 'log-error'
  }
  if (lowerMsg.includes('success') || lowerMsg.includes('成功')) {
    return 'log-success'
  }
  if (lowerMsg.includes('warn') || lowerMsg.includes('警告')) {
    return 'log-warning'
  }
  if (lowerMsg.includes('info') || lowerMsg.includes('信息')) {
    return 'log-info'
  }
  return 'log-default'
}

onMounted(() => {
  fetchLogs()
})
</script>

<template>
  <div class="logs-page">
    <div class="page-header">
      <div class="header-left">
        <h1 class="page-title">系统日志</h1>
        <p class="page-subtitle">查看应用运行日志</p>
      </div>
      <div class="header-actions">
        <el-input
          v-model="keyword"
          placeholder="搜索日志关键词..."
          class="search-input"
          clearable
          @keyup.enter="handleSearch"
          @clear="clearSearch"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
        <el-button type="primary" @click="fetchLogs" :loading="loading">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
        <el-button
          :type="autoRefresh ? 'success' : 'default'"
          @click="toggleAutoRefresh"
        >
          <el-icon><Timer /></el-icon>
          {{ autoRefresh ? '停止刷新' : '自动刷新' }}
        </el-button>
      </div>
    </div>

    <div class="logs-container">
      <el-empty v-if="logs.length === 0 && !loading" description="暂无日志" />
      
      <div v-else class="logs-list">
        <div
          v-for="(groupLogs, date) in groupedLogs"
          :key="date"
          class="log-group"
        >
          <div class="log-date">
            <el-icon><Calendar /></el-icon>
            <span>{{ date }}</span>
          </div>
          <div class="log-items">
            <div
              v-for="(log, index) in groupLogs"
              :key="index"
              class="log-item"
              :class="getLogLevelClass(log.message)"
            >
              <span class="log-time">{{ log.time }}</span>
              <span class="log-message" v-html="formatMessage(log.message)"></span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.logs-page {
  padding: var(--spacing-6);
  height: 100%;
  display: flex;
  flex-direction: column;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: var(--spacing-6);
}

.header-left {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-2);
}

.page-title {
  font-size: var(--text-2xl);
  font-weight: var(--font-bold);
  color: var(--text-primary);
  margin: 0;
}

.page-subtitle {
  font-size: var(--text-base);
  color: var(--text-secondary);
  margin: 0;
}

.header-actions {
  display: flex;
  gap: var(--spacing-3);
  align-items: center;
}

.search-input {
  width: 280px;
}

.logs-container {
  flex: 1;
  overflow-y: auto;
  background: var(--bg-card);
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-color);
  padding: var(--spacing-4);
}

.logs-list {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-6);
}

.log-group {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-3);
}

.log-date {
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
  font-size: var(--text-sm);
  font-weight: var(--font-semibold);
  color: var(--text-secondary);
  padding-bottom: var(--spacing-2);
  border-bottom: 1px solid var(--border-color);
}

.log-items {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-2);
}

.log-item {
  display: flex;
  gap: var(--spacing-4);
  padding: var(--spacing-3) var(--spacing-4);
  border-radius: var(--radius-md);
  font-size: var(--text-sm);
  font-family: var(--font-family-mono);
  transition: background-color var(--transition-fast);
}

.log-item:hover {
  background-color: var(--hover-bg);
}

.log-time {
  color: var(--text-tertiary);
  flex-shrink: 0;
  min-width: 80px;
}

.log-message {
  color: var(--text-primary);
  word-break: break-all;
  line-height: 1.6;
}

.log-message :deep(mark) {
  background-color: var(--color-primary-500);
  color: var(--text-inverse);
  padding: 0 4px;
  border-radius: var(--radius-sm);
}

/* 日志级别样式 - 深色主题 */
.log-error .log-message {
  color: var(--color-error-400);
}

.log-success .log-message {
  color: var(--color-success-400);
}

.log-warning .log-message {
  color: var(--color-warning-400);
}

.log-info .log-message {
  color: var(--color-info-400);
}

/* 浅色主题适配 */
[data-theme="light"] .log-error .log-message {
  color: var(--color-error-600);
}

[data-theme="light"] .log-success .log-message {
  color: var(--color-success-600);
}

[data-theme="light"] .log-warning .log-message {
  color: var(--color-warning-600);
}

[data-theme="light"] .log-info .log-message {
  color: var(--color-info-600);
}

[data-theme="light"] .log-message :deep(mark) {
  background-color: var(--color-primary-200);
  color: var(--color-primary-800);
}
</style>
