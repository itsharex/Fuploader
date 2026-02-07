<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useAccountStore } from '../../stores'
import { useTaskStore } from '../../stores'

const route = useRoute()
const accountStore = useAccountStore()
const taskStore = useTaskStore()

const pageTitle = computed(() => {
  return (route.meta.title as string) || 'Fuploader'
})

const validAccountCount = computed(() => accountStore.validAccounts.length)
const runningTaskCount = computed(() => taskStore.runningTasks.length)
</script>

<template>
  <header class="header">
    <div class="header-left">
      <h1 class="page-title">{{ pageTitle }}</h1>
    </div>
    
    <div class="header-right">
      <div class="stat-item">
        <el-icon class="stat-icon" color="var(--success-color)"><CircleCheck /></el-icon>
        <span class="stat-label">有效账号</span>
        <span class="stat-value">{{ validAccountCount }}</span>
      </div>
      
      <div class="stat-item">
        <el-icon class="stat-icon" color="var(--warning-color)"><Loading /></el-icon>
        <span class="stat-label">进行中</span>
        <span class="stat-value">{{ runningTaskCount }}</span>
      </div>
      
      <el-divider direction="vertical" />
      
      <el-button type="primary" size="small" @click="$router.push('/publish')">
        <el-icon><Plus /></el-icon>
        新建发布
      </el-button>
    </div>
  </header>
</template>

<style scoped>
.header {
  height: 64px;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border-color);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 var(--spacing-lg);
  position: sticky;
  top: 0;
  z-index: 50;
}

.header-left {
  display: flex;
  align-items: center;
}

.page-title {
  font-family: var(--font-family-display);
  font-size: 20px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

.header-right {
  display: flex;
  align-items: center;
  gap: var(--spacing-lg);
}

.stat-item {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  padding: var(--spacing-sm) var(--spacing-md);
  background: var(--bg-tertiary);
  border-radius: var(--radius-md);
}

.stat-icon {
  font-size: 16px;
}

.stat-label {
  font-size: 13px;
  color: var(--text-secondary);
}

.stat-value {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
}

:deep(.el-divider--vertical) {
  border-color: var(--border-color);
}
</style>
