import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

export type ThemeType = 'dark' | 'light'

export const useThemeStore = defineStore('theme', () => {
  // 从 localStorage 读取主题设置，默认深色
  const savedTheme = localStorage.getItem('theme') as ThemeType
  const theme = ref<ThemeType>(savedTheme || 'dark')

  // 应用主题
  function applyTheme(newTheme: ThemeType) {
    theme.value = newTheme
    document.documentElement.setAttribute('data-theme', newTheme)
    localStorage.setItem('theme', newTheme)
  }

  // 切换主题
  function toggleTheme() {
    const newTheme = theme.value === 'dark' ? 'light' : 'dark'
    applyTheme(newTheme)
  }

  // 初始化主题
  function initTheme() {
    document.documentElement.setAttribute('data-theme', theme.value)
  }

  // 监听主题变化
  watch(theme, (newTheme) => {
    document.documentElement.setAttribute('data-theme', newTheme)
  }, { immediate: true })

  return {
    theme,
    applyTheme,
    toggleTheme,
    initTheme
  }
})
