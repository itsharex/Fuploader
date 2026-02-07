<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import { useAccountStore, useTaskStore, useVideoStore } from './stores'
import { setupEventListeners } from './utils/event'
import AppSidebar from './components/common/AppSidebar.vue'
import AppHeader from './components/common/AppHeader.vue'

const accountStore = useAccountStore()
const taskStore = useTaskStore()
const videoStore = useVideoStore()

let unsubscribeEvents: (() => void) | null = null

onMounted(() => {
  // 初始化数据
  accountStore.fetchAccounts()
  videoStore.fetchVideos()
  taskStore.fetchTasks()
  
  // 设置事件监听
  unsubscribeEvents = setupEventListeners({
    onUploadProgress: (event) => {
      console.log('[Event] Upload Progress:', event)
      taskStore.updateProgress(event)
    },
    onUploadComplete: (event) => {
      taskStore.updateTaskStatus(event.taskId, 'success')
      taskStore.updateTaskPublishUrl(event.taskId, event.publishUrl)
    },
    onUploadError: (event) => {
      taskStore.updateTaskError(event.taskId, event.error)
    },
    onLoginSuccess: (event) => {
      accountStore.updateAccountStatus(event.accountId, 1)
    },
    onLoginError: (event) => {
      accountStore.updateAccountStatus(event.accountId, 0)
    },
    onTaskStatusChanged: (event) => {
      taskStore.updateTaskStatus(event.taskId, event.newStatus)
    },
    onAccountStatusChanged: (event) => {
      accountStore.updateAccountStatus(event.accountId, event.newStatus)
    }
  })
})

onUnmounted(() => {
  if (unsubscribeEvents) {
    unsubscribeEvents()
  }
})
</script>

<template>
  <div class="app">
    <AppSidebar />
    
    <div class="main-container">
      <AppHeader />
      
      <main class="main-content">
        <router-view v-slot="{ Component }">
          <transition name="fade" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </main>
    </div>
  </div>
</template>

<style scoped>
.app {
  display: flex;
  height: 100vh;
  width: 100vw;
  background: var(--bg-primary);
}

.main-container {
  flex: 1;
  margin-left: 220px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.main-content {
  flex: 1;
  padding: var(--spacing-lg);
  overflow-y: auto;
  background: var(--bg-primary);
}

/* 页面切换动画 */
.fade-enter-active,
.fade-leave-active {
  transition: opacity var(--transition-slow);
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
