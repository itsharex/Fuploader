<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()

const menuItems = computed(() => {
  return router.getRoutes()
    .filter(r => r.meta?.title)
    .map(r => ({
      path: r.path,
      name: r.name as string,
      title: r.meta.title as string,
      icon: r.meta.icon as string
    }))
})

const activeRoute = computed(() => route.path)

function navigateTo(path: string) {
  router.push(path)
}
</script>

<template>
  <aside class="sidebar">
    <div class="sidebar-header">
      <div class="logo">
        <el-icon class="logo-icon"><Upload /></el-icon>
        <span class="logo-text">Fuploader</span>
      </div>
    </div>
    
    <nav class="sidebar-nav">
      <div
        v-for="item in menuItems"
        :key="item.path"
        class="nav-item"
        :class="{ active: activeRoute === item.path }"
        @click="navigateTo(item.path)"
      >
        <el-icon class="nav-icon">
          <component :is="item.icon" />
        </el-icon>
        <span class="nav-text">{{ item.title }}</span>
      </div>
    </nav>
    
    <div class="sidebar-footer">
      <div class="version">v1.0.0</div>
    </div>
  </aside>
</template>

<style scoped>
.sidebar {
  width: 220px;
  height: 100%;
  background: var(--bg-secondary);
  border-right: 1px solid var(--border-color);
  display: flex;
  flex-direction: column;
  position: fixed;
  left: 0;
  top: 0;
  z-index: 100;
}

.sidebar-header {
  padding: var(--spacing-lg);
  border-bottom: 1px solid var(--border-color);
}

.logo {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
}

.logo-icon {
  font-size: 28px;
  color: var(--primary-color);
}

.logo-text {
  font-family: var(--font-family-display);
  font-size: 20px;
  font-weight: 700;
  background: var(--primary-gradient);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.sidebar-nav {
  flex: 1;
  padding: var(--spacing-md);
  display: flex;
  flex-direction: column;
  gap: var(--spacing-xs);
  overflow-y: auto;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: var(--spacing-md);
  padding: var(--spacing-md);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all var(--transition-fast);
  color: var(--text-secondary);
}

.nav-item:hover {
  background-color: var(--selected-bg);
  color: var(--selected-text);
}

.nav-item.active {
  background: var(--primary-gradient);
  color: var(--text-inverse);
  box-shadow: var(--shadow-glow);
}

.nav-icon {
  font-size: 18px;
}

.nav-text {
  font-size: 14px;
  font-weight: 500;
}

.sidebar-footer {
  padding: var(--spacing-md);
  border-top: 1px solid var(--border-color);
  text-align: center;
}

.version {
  font-size: 12px;
  color: var(--text-tertiary);
}
</style>
