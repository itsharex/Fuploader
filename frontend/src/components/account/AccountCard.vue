<template>
  <div class="account-card" :class="{ 'is-expired': account.status === 0 }">
    <div class="account-header">
      <div class="platform-icon">
        <img :src="platformIcon" :alt="account.platform" />
      </div>
      <div class="account-info">
        <h4 class="account-name">{{ account.name }}</h4>
        <span class="platform-name">{{ platformName }}</span>
      </div>
      <div class="account-status">
        <el-tag :type="statusType" size="small">
          {{ statusText }}
        </el-tag>
      </div>
    </div>

    <div class="account-body" v-if="account.username">
      <div class="username">
        <el-icon><User /></el-icon>
        <span>{{ account.username }}</span>
      </div>
    </div>

    <div class="account-footer">
      <span class="update-time">更新于 {{ formatTime(account.updatedAt) }}</span>
      <div class="actions">
        <el-button
          v-if="account.status !== 1"
          type="primary"
          size="small"
          @click="$emit('login', account.id)"
        >
          登录
        </el-button>
        <el-button
          type="info"
          size="small"
          @click="$emit('validate', account.id)"
        >
          验证
        </el-button>
        <el-button
          type="danger"
          size="small"
          @click="$emit('delete', account.id)"
        >
          删除
        </el-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Account } from '../../types'
import { formatTime } from '../../utils/format'

const props = defineProps<{
  account: Account
}>()

defineEmits<{
  login: [id: number]
  validate: [id: number]
  delete: [id: number]
}>()

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
  return icons[props.account.platform] || '/icons/default.svg'
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
  return names[props.account.platform] || props.account.platform
})

const statusType = computed(() => {
  switch (props.account.status) {
    case 1: return 'success'
    case 2: return 'warning'
    default: return 'danger'
  }
})

const statusText = computed(() => {
  switch (props.account.status) {
    case 1: return '有效'
    case 2: return '已过期'
    default: return '无效'
  }
})
</script>

<style scoped>
.account-card {
  background: var(--bg-card);
  border-radius: var(--radius-md);
  padding: var(--spacing-md);
  box-shadow: var(--shadow-card);
  transition: all var(--transition-slow);
  border: 1px solid var(--border-color);
}

.account-card:hover {
  box-shadow: var(--shadow-card-hover);
}

.account-card.is-expired {
  border-color: var(--error-color);
}

.account-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.platform-icon {
  width: 48px;
  height: 48px;
  border-radius: var(--radius-md);
  overflow: hidden;
  background: var(--bg-secondary);
  display: flex;
  align-items: center;
  justify-content: center;
}

.platform-icon img {
  width: 32px;
  height: 32px;
  object-fit: contain;
}

.account-info {
  flex: 1;
}

.account-name {
  margin: 0 0 4px 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
}

.platform-name {
  font-size: 13px;
  color: var(--text-tertiary);
}

.account-body {
  margin-bottom: 12px;
  padding: 8px 0;
  border-top: 1px solid var(--border-color);
  border-bottom: 1px solid var(--border-color);
}

.username {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--text-secondary);
  font-size: 14px;
}

.account-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.update-time {
  font-size: 12px;
  color: var(--text-tertiary);
}

.actions {
  display: flex;
  gap: 8px;
}
</style>
