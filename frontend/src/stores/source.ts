import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { sourceApi } from '@/api/source'
import { useAppStore } from './app'

export const useSourceStore = defineStore('source', () => {
  const appStore = useAppStore()

  // State
  const sources = ref<string[]>([])
  const currentSource = ref<string>('')
  const loading = ref(false)

  // Computed
  const hasSources = computed(() => sources.value.length > 0)
  const hasCurrentSource = computed(() => !!currentSource.value)

  // Actions
  async function loadSources() {
    loading.value = true
    try {
      const response = await sourceApi.getSources()
      sources.value = response.sources
      if (sources.value.length > 0 && !currentSource.value) {
        currentSource.value = sources.value[0]
      }
    } catch (error) {
      appStore.showError('加载图片源失败')
      console.error('Failed to load sources:', error)
    } finally {
      loading.value = false
    }
  }

  function setCurrentSource(source: string) {
    if (sources.value.includes(source)) {
      currentSource.value = source
    }
  }

  return {
    // State
    sources,
    currentSource,
    loading,

    // Computed
    hasSources,
    hasCurrentSource,

    // Actions
    loadSources,
    setCurrentSource
  }
})
