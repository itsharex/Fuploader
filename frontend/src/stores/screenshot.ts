import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import {
  getScreenshotConfig,
  updateScreenshotConfig,
  getScreenshots,
  deleteScreenshot,
  batchDeleteScreenshots,
  deleteAllScreenshots,
  getPlatformScreenshotStats,
  cleanOldScreenshots,
  openScreenshotDir
} from '../api'
import type {
  ScreenshotConfig,
  Screenshot,
  ScreenshotQuery,
  PlatformScreenshotConfig
} from '../types'
import { DEFAULT_SCREENSHOT_CONFIG, PLATFORM_NAME_MAP } from '../types'

export const useScreenshotStore = defineStore('screenshot', () => {
  // State
  const config = ref<ScreenshotConfig>({ ...DEFAULT_SCREENSHOT_CONFIG })
  const screenshots = ref<Screenshot[]>([])
  const platformStats = ref<PlatformScreenshotConfig[]>([])
  const selectedIds = ref<string[]>([])
  const loading = ref(false)
  const saving = ref(false)
  const total = ref(0)
  const totalSize = ref(0)
  const currentPage = ref(1)
  const pageSize = ref(20)

  // Getters
  const isAllSelected = computed(() => {
    return screenshots.value.length > 0 && selectedIds.value.length === screenshots.value.length
  })

  const selectedCount = computed(() => selectedIds.value.length)

  const hasSelection = computed(() => selectedIds.value.length > 0)

  const platformOptions = computed(() => [
    { label: '全部平台', value: '' },
    { label: '小红书', value: 'xiaohongshu' },
    { label: '视频号', value: 'tencent' },
    { label: '抖音', value: 'douyin' },
    { label: '快手', value: 'kuaishou' },
    { label: '百家号', value: 'baijiahao' },
    { label: 'TikTok', value: 'tiktok' }
  ])

  const totalScreenshots = computed(() => {
    return platformStats.value.reduce((sum, stat) => sum + stat.screenshotCount, 0)
  })

  // Actions
  async function fetchConfig() {
    try {
      const data = await getScreenshotConfig()
      config.value = data
    } catch (error) {
      console.error('获取截图配置失败:', error)
      config.value = { ...DEFAULT_SCREENSHOT_CONFIG }
    }
  }

  async function saveConfig() {
    saving.value = true
    try {
      await updateScreenshotConfig(config.value)
      return true
    } catch (error) {
      console.error('保存截图配置失败:', error)
      throw error
    } finally {
      saving.value = false
    }
  }

  async function fetchScreenshots(query: Partial<ScreenshotQuery> = {}) {
    loading.value = true
    try {
      const result = await getScreenshots({
        page: query.page ?? currentPage.value,
        pageSize: query.pageSize ?? pageSize.value,
        ...query
      })
      screenshots.value = result.list
      total.value = result.total
      totalSize.value = result.totalSize
      // 清空选择
      selectedIds.value = []
      return result
    } catch (error) {
      console.error('获取截图列表失败:', error)
      screenshots.value = []
      total.value = 0
      throw error
    } finally {
      loading.value = false
    }
  }

  async function fetchPlatformStats() {
    try {
      const stats = await getPlatformScreenshotStats()
      platformStats.value = stats
    } catch (error) {
      console.error('获取平台截图统计失败:', error)
      platformStats.value = []
    }
  }

  async function removeScreenshot(id: string) {
    try {
      await deleteScreenshot(id)
      // 从列表中移除
      screenshots.value = screenshots.value.filter(s => s.id !== id)
      // 从选中列表中移除
      selectedIds.value = selectedIds.value.filter(sid => sid !== id)
      total.value--
      return true
    } catch (error) {
      console.error('删除截图失败:', error)
      throw error
    }
  }

  async function removeSelected() {
    if (selectedIds.value.length === 0) return 0
    
    try {
      const deleted = await batchDeleteScreenshots(selectedIds.value)
      // 刷新列表
      await fetchScreenshots()
      return deleted
    } catch (error) {
      console.error('批量删除截图失败:', error)
      throw error
    }
  }

  async function removeAll() {
    try {
      const deleted = await deleteAllScreenshots()
      // 刷新列表
      await fetchScreenshots()
      return deleted
    } catch (error) {
      console.error('删除所有截图失败:', error)
      throw error
    }
  }

  async function cleanOld() {
    try {
      const cleaned = await cleanOldScreenshots()
      // 刷新列表
      await fetchScreenshots()
      return cleaned
    } catch (error) {
      console.error('清理旧截图失败:', error)
      throw error
    }
  }

  async function openDir(platform: string = '') {
    try {
      await openScreenshotDir(platform)
    } catch (error) {
      console.error('打开截图目录失败:', error)
      throw error
    }
  }

  function toggleSelection(id: string) {
    const index = selectedIds.value.indexOf(id)
    if (index > -1) {
      selectedIds.value.splice(index, 1)
    } else {
      selectedIds.value.push(id)
    }
  }

  function selectAll() {
    if (isAllSelected.value) {
      selectedIds.value = []
    } else {
      selectedIds.value = screenshots.value.map(s => s.id)
    }
  }

  function clearSelection() {
    selectedIds.value = []
  }

  function setPage(page: number) {
    currentPage.value = page
  }

  function setPageSize(size: number) {
    pageSize.value = size
    currentPage.value = 1
  }

  function updateConfig(partial: Partial<ScreenshotConfig>) {
    config.value = { ...config.value, ...partial }
  }

  function getPlatformName(platform: string): string {
    return PLATFORM_NAME_MAP[platform] || platform
  }

  function formatFileSize(bytes: number): string {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }

  return {
    // State
    config,
    screenshots,
    platformStats,
    selectedIds,
    loading,
    saving,
    total,
    totalSize,
    currentPage,
    pageSize,
    // Getters
    isAllSelected,
    selectedCount,
    hasSelection,
    platformOptions,
    totalScreenshots,
    // Actions
    fetchConfig,
    saveConfig,
    fetchScreenshots,
    fetchPlatformStats,
    removeScreenshot,
    removeSelected,
    removeAll,
    cleanOld,
    openDir,
    toggleSelection,
    selectAll,
    clearSelection,
    setPage,
    setPageSize,
    updateConfig,
    getPlatformName,
    formatFileSize
  }
})
