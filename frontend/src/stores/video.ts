import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Video } from '../types'
import * as videoApi from '../api/video'

export const useVideoStore = defineStore('video', () => {
  // State
  const videos = ref<Video[]>([])
  const loading = ref(false)
  const currentVideo = ref<Video | null>(null)

  // Getters
  const videoList = computed(() => videos.value)
  const isLoading = computed(() => loading.value)
  
  const videosWithThumbnail = computed(() =>
    videos.value.filter(v => v.thumbnail)
  )

  // Actions
  async function fetchVideos() {
    loading.value = true
    try {
      videos.value = await videoApi.getVideos()
    } finally {
      loading.value = false
    }
  }

  async function addVideo(filePath: string) {
    loading.value = true
    try {
      const video = await videoApi.addVideo(filePath)
      videos.value.push(video)
      return video
    } finally {
      loading.value = false
    }
  }

  async function selectAndAddVideo() {
    const filePath = await videoApi.selectVideoFile()
    if (filePath) {
      return await addVideo(filePath)
    }
    return null
  }

  async function updateVideo(video: Video) {
    loading.value = true
    try {
      await videoApi.updateVideo(video)
      const index = videos.value.findIndex(v => v.id === video.id)
      if (index !== -1) {
        videos.value[index] = video
      }
    } finally {
      loading.value = false
    }
  }

  async function deleteVideo(id: number) {
    loading.value = true
    try {
      await videoApi.deleteVideo(id)
      videos.value = videos.value.filter(v => v.id !== id)
    } finally {
      loading.value = false
    }
  }

  function setCurrentVideo(video: Video | null) {
    currentVideo.value = video
  }

  return {
    videos,
    loading,
    currentVideo,
    videoList,
    isLoading,
    videosWithThumbnail,
    fetchVideos,
    addVideo,
    selectAndAddVideo,
    updateVideo,
    deleteVideo,
    setCurrentVideo
  }
})
