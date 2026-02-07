<template>
  <div class="task-card" :class="`status-${task.status}`">
    <div class="task-header">
      <div class="task-platform">
        <img :src="platformIcon" :alt="task.platform" />
        <span>{{ platformName }}</span>
      </div>
      <el-tag :type="statusType" size="small" effect="light">
        {{ statusText }}
      </el-tag>
    </div>

    <div class="task-video">
      <div class="video-thumb">
        <img :src="task.video.thumbnail || defaultThumbnail" />
      </div>
      <div class="video-info">
        <h5 class="video-title">{{ task.video.title || task.video.filename }}</h5>
        <span class="video-filename">{{ task.video.filename }}</span>
      </div>
    </div>

    <div class="task-progress" v-if="task.status === 'uploading'">
      <el-progress
        :percentage="task.progress"
        :status="progressStatus"
        :stroke-width="8"
      />
      <span class="progress-text">{{ task.progress }}%</span>
    </div>

    <div class="task-schedule" v-if="task.scheduleTime">
      <el-icon><Clock /></el-icon>
      <span>定时发布: {{ formatTime(task.scheduleTime) }}</span>
    </div>

    <div class="task-error" v-if="task.status === 'failed' && task.errorMsg">
      <el-icon><Warning /></el-icon>
      <span>{{ task.errorMsg }}</span>
    </div>

    <div class="task-result" v-if="task.status === 'success' && task.publishUrl">
      <el-button type="success" link size="small" @click="openUrl(task.publishUrl)">
        <el-icon><Link /></el-icon>
        查看已发布视频
      </el-button>
    </div>

    <div class="task-actions">
      <template v-if="task.status === 'pending'">
        <el-button type="danger" size="small" @click="$emit('cancel', task.id)">
          取消
        </el-button>
      </template>

      <template v-if="task.status === 'failed'">
        <el-button type="primary" size="small" @click="$emit('retry', task.id)">
          <el-icon><RefreshRight /></el-icon>
          重试
        </el-button>
        <el-button type="danger" size="small" @click="$emit('delete', task.id)">
          删除
        </el-button>
      </template>

      <template v-if="task.status === 'success' || task.status === 'cancelled'">
        <el-button type="danger" size="small" @click="$emit('delete', task.id)">
          删除
        </el-button>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { UploadTask } from '../../types'
import { formatTime } from '../../utils/format'

const props = defineProps<{
  task: UploadTask
}>()

defineEmits<{
  cancel: [id: number]
  retry: [id: number]
  delete: [id: number]
}>()

const defaultThumbnail = '/images/default-video-thumb.png'

const platformIcon = computed(() => {
  const icons: Record<string, string> = {
    douyin: '/icons/douyin.svg',
    tencent: '/icons/tencent.svg',
    kuaishou: '/icons/kuaishou.svg',
    bilibili: '/icons/bilibili.svg',
    xiaohongshu: '/icons/xiaohongshu.svg',
    baijiahao: '/icons/baijiahao.svg',
    tiktok: '/icons/tiktok.svg'
  }
  return icons[props.task.platform] || '/icons/default.svg'
})

const platformName = computed(() => {
  const names: Record<string, string> = {
    douyin: '抖音',
    tencent: '视频号',
    kuaishou: '快手',
    bilibili: 'B站',
    xiaohongshu: '小红书',
    baijiahao: '百家号',
    tiktok: 'TikTok'
  }
  return names[props.task.platform] || props.task.platform
})

const statusType = computed(() => {
  const types: Record<string, any> = {
    pending: 'info',
    uploading: 'warning',
    success: 'success',
    failed: 'danger',
    cancelled: ''
  }
  return types[props.task.status] || 'info'
})

const statusText = computed(() => {
  const texts: Record<string, string> = {
    pending: '等待中',
    uploading: '上传中',
    success: '成功',
    failed: '失败',
    cancelled: '已取消'
  }
  return texts[props.task.status] || props.task.status
})

const progressStatus = computed(() => {
  if (props.task.status === 'failed') return 'exception'
  if (props.task.progress === 100) return 'success'
  return ''
})

function openUrl(url: string) {
  window.open(url, '_blank')
}
</script>

<style scoped>
.task-card {
  background: var(--bg-card);
  border-radius: var(--radius-md);
  padding: var(--spacing-md);
  box-shadow: var(--shadow-card);
  transition: all var(--transition-slow);
  border-left: 4px solid transparent;
  border: 1px solid var(--border-color);
}

.task-card:hover {
  box-shadow: var(--shadow-card-hover);
}

.task-card.status-pending {
  border-left-color: var(--info-color);
}

.task-card.status-uploading {
  border-left-color: var(--warning-color);
}

.task-card.status-success {
  border-left-color: var(--success-color);
}

.task-card.status-failed {
  border-left-color: var(--error-color);
}

.task-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.task-platform {
  display: flex;
  align-items: center;
  gap: 8px;
}

.task-platform img {
  width: 24px;
  height: 24px;
  object-fit: contain;
}

.task-platform span {
  font-weight: 500;
  color: var(--text-primary);
}

.task-video {
  display: flex;
  gap: 12px;
  margin-bottom: 12px;
  padding: 12px;
  background: var(--bg-secondary);
  border-radius: var(--radius-sm);
}

.video-thumb {
  width: 80px;
  height: 45px;
  border-radius: var(--radius-sm);
  overflow: hidden;
  flex-shrink: 0;
}

.video-thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.video-info {
  flex: 1;
  min-width: 0;
}

.video-title {
  margin: 0 0 4px 0;
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.video-filename {
  font-size: 12px;
  color: var(--text-tertiary);
}

.task-progress {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.task-progress .el-progress {
  flex: 1;
}

.progress-text {
  font-size: 13px;
  color: var(--text-secondary);
  min-width: 40px;
  text-align: right;
}

.task-schedule {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  padding: 8px 12px;
  background: var(--info-bg);
  border-radius: var(--radius-sm);
  color: var(--info-color);
  font-size: 13px;
}

.task-error {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  padding: 8px 12px;
  background: var(--error-bg);
  border-radius: var(--radius-sm);
  color: var(--error-color);
  font-size: 13px;
}

.task-result {
  margin-bottom: 12px;
}

.task-actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}
</style>
