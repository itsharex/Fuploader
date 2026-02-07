<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useTaskStore } from '../stores'
import { formatDateTime, getRelativeTime } from '../utils/format'
import { TASK_STATUS_LABELS } from '../utils/constants'
import type { TaskStatus } from '../types'

const taskStore = useTaskStore()

const statusFilter = ref<TaskStatus | ''>('')

const filteredTasks = computed(() => {
  if (!statusFilter.value) return taskStore.tasks
  return taskStore.tasks.filter(t => t.status === statusFilter.value)
})

async function handleCancelTask(taskId: number) {
  try {
    await ElMessageBox.confirm('确定要取消这个任务吗？', '确认取消', {
      confirmButtonText: '取消任务',
      cancelButtonText: '保留',
      type: 'warning'
    })
    
    await taskStore.cancelTask(taskId)
    ElMessage.success('任务已取消')
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('取消失败')
    }
  }
}

async function handleRetryTask(taskId: number) {
  try {
    await taskStore.retryTask(taskId)
    ElMessage.success('任务已重新提交')
  } catch (error) {
    ElMessage.error('重试失败')
  }
}

async function handleDeleteTask(taskId: number) {
  try {
    await ElMessageBox.confirm('确定要删除这个任务吗？', '确认删除', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning'
    })
    
    await taskStore.deleteTask(taskId)
    ElMessage.success('任务已删除')
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

// 一键删除已完成的任务
async function handleBatchDelete() {
  const completedTasks = filteredTasks.value.filter(
    t => t.status === 'success' || t.status === 'failed' || t.status === 'cancelled'
  )
  
  if (completedTasks.length === 0) {
    ElMessage.info('没有可删除的已完成任务')
    return
  }
  
  try {
    await ElMessageBox.confirm(
      `确定要删除 ${completedTasks.length} 个已完成的任务吗？`,
      '批量删除',
      {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    
    // 批量删除
    for (const task of completedTasks) {
      await taskStore.deleteTask(task.id)
    }
    
    ElMessage.success(`已删除 ${completedTasks.length} 个任务`)
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('批量删除失败')
    }
  }
}

function getProgressStatus(status: string) {
  switch (status) {
    case 'success': return 'success'
    case 'failed': return 'exception'
    case 'uploading': return ''
    default: return ''
  }
}

onMounted(() => {
  taskStore.fetchTasks()
})
</script>

<template>
  <div class="tasks-page">
    <div class="page-header">
      <div class="header-left">
        <h2 class="page-title">任务队列</h2>
        <p class="page-subtitle">管理您的上传任务</p>
      </div>
      <div class="header-right">
        <el-radio-group v-model="statusFilter" size="small">
          <el-radio-button label="">全部</el-radio-button>
          <el-radio-button label="pending">等待中</el-radio-button>
          <el-radio-button label="uploading">上传中</el-radio-button>
          <el-radio-button label="success">成功</el-radio-button>
          <el-radio-button label="failed">失败</el-radio-button>
        </el-radio-group>
        <el-divider direction="vertical" />
        <el-button
          type="danger"
          size="small"
          plain
          @click="handleBatchDelete"
        >
          <el-icon><Delete /></el-icon>
          一键删除
        </el-button>
      </div>
    </div>

    <div class="tasks-list" v-if="filteredTasks.length > 0">
      <div
        v-for="task in filteredTasks"
        :key="task.id"
        class="task-card"
        :class="task.status"
      >
        <div class="task-main">
          <div class="task-video-thumb">
            <img v-if="task.video?.thumbnail" :src="task.video.thumbnail" :alt="task.video.filename">
            <div v-else class="thumb-placeholder">
              <el-icon><VideoCamera /></el-icon>
            </div>
          </div>
          
          <div class="task-info">
            <div class="task-header">
              <h4 class="task-title">{{ task.video?.title || task.video?.filename || '未命名视频' }}</h4>
              <el-tag
                :type="TASK_STATUS_LABELS[task.status]?.type"
                size="small"
                effect="dark"
              >
                {{ TASK_STATUS_LABELS[task.status]?.label }}
              </el-tag>
            </div>
            
            <div class="task-meta">
              <span class="meta-item">
                <el-icon><Platform /></el-icon>
                {{ task.platform }}
              </span>
              <span class="meta-item">
                <el-icon><User /></el-icon>
                {{ task.account?.name || '未知账号' }}
              </span>
              <span class="meta-item" v-if="task.scheduleTime">
                <el-icon><Clock /></el-icon>
                {{ formatDateTime(task.scheduleTime) }}
              </span>
              <span class="meta-item">
                <el-icon><Timer /></el-icon>
                {{ getRelativeTime(task.createdAt) }}
              </span>
            </div>

            <div class="task-progress" v-if="task.status === 'uploading'">
              <el-progress
                :percentage="task.progress"
                :status="getProgressStatus(task.status)"
                :stroke-width="8"
              />
              <span class="progress-text">{{ task.progress }}%</span>
            </div>

            <div class="task-error" v-if="task.errorMsg">
              <el-icon><Warning /></el-icon>
              <span>{{ task.errorMsg }}</span>
            </div>

            <div class="task-success" v-if="task.publishUrl">
              <el-icon><Link /></el-icon>
              <a :href="task.publishUrl" target="_blank" rel="noopener">查看发布结果</a>
            </div>
          </div>
        </div>

        <div class="task-actions">
          <el-button
            v-if="task.status === 'pending' || task.status === 'uploading'"
            type="warning"
            size="small"
            @click="handleCancelTask(task.id)"
          >
            <el-icon><CircleClose /></el-icon>
            取消
          </el-button>
          
          <el-button
            v-if="task.status === 'failed'"
            type="primary"
            size="small"
            @click="handleRetryTask(task.id)"
          >
            <el-icon><RefreshRight /></el-icon>
            重试
          </el-button>
          
          <el-button
            v-if="task.status === 'success' || task.status === 'failed' || task.status === 'cancelled'"
            type="danger"
            size="small"
            plain
            @click="handleDeleteTask(task.id)"
          >
            <el-icon><Delete /></el-icon>
          </el-button>
        </div>
      </div>
    </div>

    <el-empty
      v-else
      description="暂无任务"
      :image-size="120"
    >
      <el-button type="primary" @click="$router.push('/publish')">创建新任务</el-button>
    </el-empty>
  </div>
</template>

<style scoped>
.tasks-page {
  padding-bottom: var(--spacing-xl);
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--spacing-xl);
}

.header-left {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-xs);
}

.page-title {
  font-size: 24px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

.page-subtitle {
  font-size: 14px;
  color: var(--text-secondary);
  margin: 0;
}

.tasks-list {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-md);
}

.task-card {
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  padding: var(--spacing-lg);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--spacing-lg);
  transition: all var(--transition-fast);
}

.task-card:hover {
  border-color: var(--primary-color);
}

.task-card.uploading {
  border-color: var(--warning-color);
  animation: pulse-border 2s infinite;
}

@keyframes pulse-border {
  0%, 100% { border-color: var(--warning-color); }
  50% { border-color: var(--warning-color); opacity: 0.5; }
}

.task-main {
  display: flex;
  gap: var(--spacing-md);
  flex: 1;
}

.task-video-thumb {
  width: 120px;
  aspect-ratio: 16/9;
  border-radius: var(--radius-md);
  overflow: hidden;
  background: var(--bg-secondary);
  flex-shrink: 0;
}

.task-video-thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.thumb-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-tertiary);
}

.task-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: var(--spacing-sm);
}

.task-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--spacing-md);
}

.task-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

.task-meta {
  display: flex;
  gap: var(--spacing-lg);
  flex-wrap: wrap;
}

.meta-item {
  display: flex;
  align-items: center;
  gap: var(--spacing-xs);
  font-size: 12px;
  color: var(--text-secondary);
}

.task-progress.header-right {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
}

.header-right .el-divider {
  margin: 0 var(--spacing-xs);
}

.task-progress :deep(.el-progress) {
  flex: 1;
}

.progress-text {
  font-size: 12px;
  color: var(--text-secondary);
  min-width: 40px;
  text-align: right;
}

.task-error {
  display: flex;
  align-items: center;
  gap: var(--spacing-xs);
  font-size: 12px;
  color: var(--error-color);
  padding: var(--spacing-xs) var(--spacing-sm);
  background: var(--error-bg);
  border-radius: var(--radius-sm);
}

.task-success {
  display: flex;
  align-items: center;
  gap: var(--spacing-xs);
  font-size: 12px;
}

.task-success a {
  color: var(--success-color);
  text-decoration: none;
}

.task-success a:hover {
  text-decoration: underline;
}

.task-actions {
  display: flex;
  gap: var(--spacing-sm);
  flex-shrink: 0;
}
</style>
