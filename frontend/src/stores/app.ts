import { defineStore } from 'pinia'
import { ref } from 'vue'
import { message } from 'ant-design-vue'

export const useAppStore = defineStore('app', () => {
  // Loading state
  const loading = ref(false)

  // Batch mode state
  const batchMode = ref(false)
  const selectedImages = ref<Set<string>>(new Set())

  // Message functions
  function showMessage(options: any) {
    message.open(options)
  }

  function showSuccess(text: string) {
    message.success(text)
  }

  function showError(text: string) {
    message.error(text)
  }

  function showInfo(text: string) {
    message.info(text)
  }

  // Batch operations
  function toggleBatchMode() {
    batchMode.value = !batchMode.value
    if (!batchMode.value) {
      selectedImages.value.clear()
    }
  }

  function isImageSelected(id: string): boolean {
    return selectedImages.value.has(id)
  }

  function toggleImageSelection(id: string) {
    if (selectedImages.value.has(id)) {
      selectedImages.value.delete(id)
    } else {
      selectedImages.value.add(id)
    }
  }

  function selectAll(ids: string[]) {
    selectedImages.value = new Set(ids)
  }

  function clearSelection() {
    selectedImages.value.clear()
  }

  function getSelectedIds(): string[] {
    return Array.from(selectedImages.value)
  }

  return {
    // State
    loading,
    batchMode,
    selectedImages,

    // Methods
    showMessage,
    showSuccess,
    showError,
    showInfo,
    toggleBatchMode,
    isImageSelected,
    toggleImageSelection,
    selectAll,
    clearSelection,
    getSelectedIds
  }
})
