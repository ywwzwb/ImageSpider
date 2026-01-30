import { defineStore } from 'pinia'
import { ref, computed, watch } from 'vue'
import type { TagInfo, ImageIntegrityStatus } from '@/types/api'
import { tagApi } from '@/api/tag'
import { useAppStore } from './app'
import { useSourceStore } from './source'

export const useFilterStore = defineStore('filter', () => {
  const appStore = useAppStore()
  const sourceStore = useSourceStore()

  // State
  const tags = ref<TagInfo[]>([])
  const selectedTags = ref<Set<string>>(new Set())
  const selectedStatus = ref<Set<ImageIntegrityStatus>>(new Set())
  const loading = ref(false)
  const tagsOffset = ref(0)
  const hasMoreTags = ref(true)

  // Constants
  const TAGS_LIMIT = 50

  // Computed
  const selectedTagList = computed(() => Array.from(selectedTags.value))
  const selectedStatusList = computed(() => Array.from(selectedStatus.value))

  // Actions
  async function loadTags(sourceId: string, append = false) {
    if (!append) {
      tagsOffset.value = 0
      tags.value = []
      hasMoreTags.value = true
    }

    if (!hasMoreTags.value && append) return

    loading.value = true
    try {
      const response = await tagApi.getTags(sourceId, tagsOffset.value, TAGS_LIMIT)

      if (append) {
        tags.value.push(...response.tagList)
      } else {
        tags.value = response.tagList
      }

      tagsOffset.value += response.tagList.length
      hasMoreTags.value = tagsOffset.value < response.totalCount
    } catch (error) {
      appStore.showError('加载标签失败')
      console.error('Failed to load tags:', error)
    } finally {
      loading.value = false
    }
  }

  function toggleTag(tag: string) {
    if (selectedTags.value.has(tag)) {
      selectedTags.value.delete(tag)
    } else {
      selectedTags.value.add(tag)
    }
  }

  function addTag(tag: string) {
    selectedTags.value.add(tag)
  }

  function removeTag(tag: string) {
    selectedTags.value.delete(tag)
  }

  function hasTag(tag: string): boolean {
    return selectedTags.value.has(tag)
  }

  function toggleStatus(status: ImageIntegrityStatus) {
    if (selectedStatus.value.has(status)) {
      selectedStatus.value.delete(status)
    } else {
      selectedStatus.value.add(status)
    }
  }

  function setStatus(status: ImageIntegrityStatus, selected: boolean) {
    if (selected) {
      selectedStatus.value.add(status)
    } else {
      selectedStatus.value.delete(status)
    }
  }

  function hasStatus(status: ImageIntegrityStatus): boolean {
    return selectedStatus.value.has(status)
  }

  function reset() {
    selectedTags.value.clear()
    selectedStatus.value.clear()
  }

  // Load tags when source changes
  watch(() => sourceStore.currentSource, (newSource) => {
    if (newSource) {
      loadTags(newSource)
    }
  })

  return {
    // State
    tags,
    loading,
    hasMoreTags,

    // Computed
    selectedTagList,
    selectedStatusList,

    // Actions
    loadTags,
    toggleTag,
    addTag,
    removeTag,
    hasTag,
    toggleStatus,
    setStatus,
    hasStatus,
    reset
  }
})
