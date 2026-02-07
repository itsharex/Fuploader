<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAccountStore, useVideoStore, useTaskStore } from '../stores'
import { formatFileSize } from '../utils/format'
import { TASK_STATUS_LABELS } from '../utils/constants'

const router = useRouter()
const accountStore = useAccountStore()
const videoStore = useVideoStore()
const taskStore = useTaskStore()

const stats = computed(() => [
  {
    title: '总账号数',
    value: accountStore.accounts.length,
    icon: 'User',
    color: 'var(--color-primary-500)',
    trend: accountStore.validAccounts.length + ' 个有效'
  },
  {
    title: '视频总数',
    value: videoStore.videos.length,
    icon: 'VideoCamera',
    color: 'var(--color-primary-400)',
    trend: formatTotalVideoSize.value
  },
  {
    title: '待发布任务',
    value: taskStore.pendingTasks.length,
    icon: 'Clock',
    color: 'var(--color-warning-500)',
    trend: taskStore.runningTasks.length + ' 个进行中'
  },
  {
    title: '本月成功',
    value: taskStore.completedTasks.length,
    icon: 'CircleCheck',
    color: 'var(--color-success-500)',
    trend: '100% 成功率'
  }
])

const formatTotalVideoSize = computed(() => {
  const totalSize = videoStore.videos.reduce((sum, v) => sum + (v.fileSize || 0), 0)
  return formatFileSize(totalSize)
})

const recentTasks = computed(() => {
  return taskStore.tasks.slice(0, 5)
})

const platformDistribution = computed(() => {
  const distribution: Record<string, number> = {}
  accountStore.accounts.forEach(account => {
    distribution[account.platform] = (distribution[account.platform] || 0) + 1
  })
  return Object.entries(distribution).map(([platform, count]) => ({
    platform,
    count
  }))
})

function navigateTo(path: string) {
  router.push(path)
}

onMounted(() => {
  accountStore.fetchAccounts()
  videoStore.fetchVideos()
  taskStore.fetchTasks()
})
</script>

<template>
  <div class="dashboard">
    <!-- 统计卡片 -->
    <div class="stats-grid">
      <div
        v-for="stat in stats"
        :key="stat.title"
        class="stat-card"
        @click="stat.title === '总账号数' ? navigateTo('/accounts') : stat.title === '视频总数' ? navigateTo('/videos') : stat.title === '待发布任务' ? navigateTo('/tasks') : null"
      >
        <div class="stat-icon-wrapper" :style="{ backgroundColor: stat.color.startsWith('var') ? (stat.color.includes('primary') ? 'var(--primary-glow)' : stat.color.replace('500', 'glow')) : stat.color + '20' }">
          <el-icon :size="24" :color="stat.color">
            <component :is="stat.icon" />
          </el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ stat.value }}</div>
          <div class="stat-title">{{ stat.title }}</div>
          <div class="stat-trend">{{ stat.trend }}</div>
        </div>
      </div>
    </div>

    <div class="dashboard-grid">
      <!-- 最近任务 -->
      <div class="dashboard-card">
        <div class="card-header">
          <h3 class="card-title">
            <el-icon><List /></el-icon>
            最近任务
          </h3>
          <el-button text type="primary" @click="navigateTo('/tasks')">
            查看全部
            <el-icon class="el-icon--right"><ArrowRight /></el-icon>
          </el-button>
        </div>
        <div class="card-content">
          <el-empty v-if="recentTasks.length === 0" description="暂无任务" />
          <div v-else class="task-list">
            <div
              v-for="task in recentTasks"
              :key="task.id"
              class="task-item"
            >
              <div class="task-info">
                <div class="task-title">{{ task.video?.title || '未命名视频' }}</div>
                <div class="task-platform">
                  <el-tag size="small" :type="TASK_STATUS_LABELS[task.status]?.type">
                    {{ TASK_STATUS_LABELS[task.status]?.label }}
                  </el-tag>
                  <span class="platform-name">{{ task.platform }}</span>
                </div>
              </div>
              <div class="task-progress" v-if="task.status === 'uploading'">
                <el-progress
                  :percentage="task.progress"
                  :stroke-width="6"
                  :show-text="false"
                />
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 平台分布 -->
      <div class="dashboard-card">
        <div class="card-header">
          <h3 class="card-title">
            <el-icon><PieChart /></el-icon>
            平台分布
          </h3>
        </div>
        <div class="card-content">
          <el-empty v-if="platformDistribution.length === 0" description="暂无账号" />
          <div v-else class="platform-list">
            <div
              v-for="item in platformDistribution"
              :key="item.platform"
              class="platform-item"
            >
              <div class="platform-info">
                <span class="platform-name">{{ item.platform }}</span>
                <span class="platform-count">{{ item.count }} 个账号</span>
              </div>
              <el-progress
                :percentage="(item.count / accountStore.accounts.length) * 100"
                :stroke-width="8"
                :show-text="false"
                class="platform-progress"
              />
            </div>
          </div>
        </div>
      </div>
    </div>

  </div>
</template>

<style scoped>
.dashboard {
  padding-bottom: var(--spacing-xl);
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: var(--spacing-lg);
  margin-bottom: var(--spacing-xl);
}

.stat-card {
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  padding: var(--spacing-lg);
  display: flex;
  align-items: center;
  gap: var(--spacing-md);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.stat-card:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
  border-color: var(--primary-color);
}

.stat-icon-wrapper {
  width: 56px;
  height: 56px;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
}

.stat-content {
  flex: 1;
}

.stat-value {
  font-size: 28px;
  font-weight: 700;
  color: var(--text-primary);
  line-height: 1;
  margin-bottom: var(--spacing-xs);
}

.stat-title {
  font-size: 14px;
  color: var(--text-secondary);
  margin-bottom: var(--spacing-xs);
}

.stat-trend {
  font-size: 12px;
  color: var(--text-tertiary);
}

.dashboard-grid {
  display: grid;
  grid-template-columns: 2fr 1fr;
  gap: var(--spacing-lg);
  margin-bottom: var(--spacing-xl);
}

.dashboard-card {
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  overflow: hidden;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--spacing-md) var(--spacing-lg);
  border-bottom: 1px solid var(--border-color);
}

.card-title {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

.card-content {
  padding: var(--spacing-md);
  min-height: 200px;
}

.task-list {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-md);
}

.task-item {
  padding: var(--spacing-md);
  background: var(--bg-secondary);
  border-radius: var(--radius-md);
  border: 1px solid var(--border-color);
}

.task-info {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--spacing-sm);
}

.task-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
}

.task-platform {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
}

.platform-name {
  font-size: 12px;
  color: var(--text-tertiary);
}

.platform-list {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-lg);
}

.platform-item {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-sm);
}

.platform-info {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.platform-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
}

.platform-count {
  font-size: 12px;
  color: var(--text-secondary);
}

.platform-progress :deep(.el-progress-bar__outer) {
  background-color: var(--bg-tertiary);
}

@media (max-width: 1200px) {
  .stats-grid {
    grid-template-columns: repeat(2, 1fr);
  }
  
  .dashboard-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  .stats-grid {
    grid-template-columns: 1fr;
  }
}
</style>
