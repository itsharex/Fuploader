import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { ScheduleConfig } from '../types'
import * as scheduleApi from '../api/schedule'

export const useScheduleStore = defineStore('schedule', () => {
  // State
  const config = ref<ScheduleConfig | null>(null)
  const loading = ref(false)
  const generatedTimes = ref<string[]>([])

  // Getters
  const scheduleConfig = computed(() => config.value)
  const isLoading = computed(() => loading.value)
  
  const defaultConfig = computed((): ScheduleConfig => ({
    id: 0,
    videosPerDay: 1,
    dailyTimes: ['09:00'],
    startDays: 0,
    timeZone: 'Asia/Shanghai'
  }))

  // Actions
  async function fetchConfig() {
    loading.value = true
    try {
      config.value = await scheduleApi.getScheduleConfig()
    } finally {
      loading.value = false
    }
  }

  async function updateConfig(newConfig: ScheduleConfig) {
    loading.value = true
    try {
      await scheduleApi.updateScheduleConfig(newConfig)
      config.value = newConfig
    } finally {
      loading.value = false
    }
  }

  async function generateTimes(videoCount: number) {
    loading.value = true
    try {
      generatedTimes.value = await scheduleApi.generateScheduleTimes(videoCount)
      return generatedTimes.value
    } finally {
      loading.value = false
    }
  }

  function addTimeSlot(time: string) {
    if (config.value && !config.value.dailyTimes.includes(time)) {
      config.value.dailyTimes.push(time)
      config.value.dailyTimes.sort()
    }
  }

  function removeTimeSlot(index: number) {
    if (config.value && config.value.dailyTimes.length > 1) {
      config.value.dailyTimes.splice(index, 1)
    }
  }

  return {
    config,
    loading,
    generatedTimes,
    scheduleConfig,
    isLoading,
    defaultConfig,
    fetchConfig,
    updateConfig,
    generateTimes,
    addTimeSlot,
    removeTimeSlot
  }
})
