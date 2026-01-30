<template>
  <a-modal
    v-model:visible="visible"
    :footer="null"
    :width="modalWidth"
    :centered="true"
    @cancel="handleCancel"
  >
    <template #closeIcon>
      <div class="modal-header">
        <a-button type="text" size="small" @click="toggleBatchMode">
          {{ appStore.batchMode ? '退出批量' : '批量模式' }}
        </a-button>
      </div>
    </template>

    <div class="preview-content" :style="previewContentStyle">
      <!-- Image -->
      <div class="image-wrapper" @click="handleImageClick">
        <img
          v-if="currentImage?.localPath && fullImageUrl"
          :src="fullImageUrl"
          :alt="currentImage?.id"
          class="preview-image"
          @error="handleImageError"
        >
        <div v-else class="no-image-large">
          <div class="no-image-content">
            <div class="no-image-icon">📷</div>
            <div class="no-image-text">无图片</div>
          </div>
        </div>
      </div>

      <!-- Navigation buttons -->
      <div v-if="hasMultipleImages" class="nav-buttons">
        <a-button
          class="nav-button prev"
          shape="circle"
          size="large"
          :disabled="currentIndex === 0"
          @click.stop="navigatePrev"
        >
          <template #icon>‹</template>
        </a-button>
        <a-button
          class="nav-button next"
          shape="circle"
          size="large"
          :disabled="currentIndex === images.length - 1"
          @click.stop="navigateNext"
        >
          <template #icon>›</template>
        </a-button>
      </div>

      <!-- Selection checkbox -->
      <div v-if="appStore.batchMode" class="preview-selection">
        <a-checkbox
          :checked="appStore.isImageSelected(currentImage.id)"
          @change="handleSelectionChange"
        >
          选择
        </a-checkbox>
      </div>

      <!-- Image info -->
      <div class="image-info">
        <div class="info-row">
          <span class="info-label">ID:</span>
          <span class="info-value">{{ currentImage?.id }}</span>
        </div>
        <div class="info-row">
          <span class="info-label">状态:</span>
          <a-tag :color="getIntegrityStatusColor(currentImage?.integrityStatus)">
            {{ getIntegrityStatusText(currentImage?.integrityStatus) }}
          </a-tag>
        </div>
        <div class="info-row">
          <span class="info-label">发布时间:</span>
          <span class="info-value">{{ formatDate(currentImage?.postTime) }}</span>
        </div>
        <div class="info-row">
          <span class="info-label">标签:</span>
          <div class="tags-container">
            <a-tag
              v-for="tag in currentImage?.tags"
              :key="tag"
              size="small"
              class="info-tag"
              @click="handleTagClick(tag)"
            >
              {{ tag }}
            </a-tag>
          </div>
        </div>
      </div>

      <!-- Actions -->
      <div class="preview-actions">
        <a-space>
          <a-popconfirm
            title="确定要删除这张图片吗？"
            ok-text="删除"
            ok-type="danger"
            cancel-text="取消"
            @confirm="handleDelete"
          >
            <a-button type="primary" danger>
              删除
            </a-button>
          </a-popconfirm>
          <a-popconfirm
            title="确定要重新下载这张图片吗？"
            ok-text="重新下载"
            cancel-text="取消"
            @confirm="handleRedownload"
          >
            <a-button>重新下载</a-button>
          </a-popconfirm>
          <a-button @click="handleSetCover">
            设为封面
          </a-button>
        </a-space>
      </div>
    </div>
  </a-modal>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted, onBeforeUnmount } from 'vue'
import { useAppStore } from '@/stores/app'
import { useSourceStore } from '@/stores/source'
import { useFilterStore } from '@/stores/filter'
import { imageApi } from '@/api/image'
import { tagApi } from '@/api/tag'
import type { ImageMeta } from '@/types/api'
import { message } from 'ant-design-vue'
import { getIntegrityStatusText, getIntegrityStatusColor, formatDate } from '@/utils/format'

interface Props {
  images: ImageMeta[]
  currentIndex: number
}

interface Emits {
  (e: 'update:visible', value: boolean): void
  (e: 'update:currentIndex', value: number): void
  (e: 'refresh'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const appStore = useAppStore()
const sourceStore = useSourceStore()
const filterStore = useFilterStore()

// State
const visible = ref(false)
const modalWidth = ref(800)
const maxImageHeight = ref(600)

// Computed
const currentImage = computed(() => props.images[props.currentIndex] || null)
const hasMultipleImages = computed(() => props.images.length > 1)
const fullImageUrl = computed(() => {
  if (!currentImage.value?.localPath) return null
  return `/image/${currentImage.value.localPath.replace(/^\/image\//, '')}`
})

const previewContentStyle = computed(() => ({
  maxHeight: `${maxImageHeight.value}px`,
  overflow: 'auto'
}))

// Keyboard navigation
function handleKeydown(event: KeyboardEvent) {
  if (!visible.value) return

  switch (event.key) {
    case 'ArrowLeft':
      navigatePrev()
      break
    case 'ArrowRight':
      navigateNext()
      break
    case 'Escape':
      handleCancel()
      break
    case 'Delete':
      if (event.ctrlKey || event.metaKey) {
        event.preventDefault()
        handleDelete()
      }
      break
  }
}

// Navigation
function navigatePrev() {
  if (props.currentIndex > 0) {
    emit('update:currentIndex', props.currentIndex - 1)
  }
}

function navigateNext() {
  if (props.currentIndex < props.images.length - 1) {
    emit('update:currentIndex', props.currentIndex + 1)
  }
}

// Event handlers
function handleCancel() {
  visible.value = false
}

function handleImageClick() {
  if (appStore.batchMode) {
    handleSelectionChange()
  }
}

function handleSelectionChange() {
  if (currentImage.value) {
    appStore.toggleImageSelection(currentImage.value.id)
  }
}

function handleTagClick(tag: string) {
  if (!filterStore.hasTag(tag)) {
    filterStore.addTag(tag)
  }
}

function toggleBatchMode() {
  appStore.toggleBatchMode()
}

// CRUD operations
async function handleDelete() {
  if (!currentImage.value) return

  try {
    const response = await imageApi.batchDelete(sourceStore.currentSource, [currentImage.value.id])
    if (response.deleted && response.deleted > 0) {
      message.success('删除成功')
      emit('refresh')
      nextTick(() => {
        if (props.images.length === 0) {
          visible.value = false
        } else if (props.currentIndex >= props.images.length) {
          emit('update:currentIndex', props.images.length - 1)
        }
      })
    } else {
      message.error('删除失败')
    }
  } catch (error) {
    console.error('Failed to delete image:', error)
    message.error('删除失败')
  }
}

async function handleRedownload() {
  if (!currentImage.value) return

  try {
    const response = await imageApi.batchRedownload(sourceStore.currentSource, [currentImage.value.id])
    if (response.redownloaded && response.redownloaded > 0) {
      message.success('已标记为重新下载')
      emit('refresh')
    } else {
      message.error('重新下载失败')
    }
  } catch (error) {
    console.error('Failed to redownload image:', error)
    message.error('重新下载失败')
  }
}

async function handleSetCover() {
  if (!currentImage.value || !currentImage.value.tags.length) {
    message.warning('该图片没有标签')
    return
  }

  try {
    const tag = currentImage.value.tags[0]
    await tagApi.setTagCover(sourceStore.currentSource, tag, currentImage.value.id)
    message.success('已设置为标签封面')
  } catch (error) {
    console.error('Failed to set tag cover:', error)
    message.error('设置封面失败')
  }
}

function handleImageError() {
  console.error('Failed to load image:', currentImage.value?.id)
}

// Calculate modal size based on viewport
function calculateModalSize() {
  const viewportWidth = window.innerWidth
  const viewportHeight = window.innerHeight

  // Leave margins for modal
  modalWidth.value = Math.min(viewportWidth - 100, 1200)
  maxImageHeight.value = viewportHeight - 400 // Leave space for info and actions
}

// Watch for visibility changes
watch(() => props.currentIndex, () => {
  // Scroll to top when image changes
  const content = document.querySelector('.preview-content')
  if (content) {
    content.scrollTop = 0
  }
})

// Expose visible to parent
watch(visible, (value) => {
  emit('update:visible', value)
})

// Mount
onMounted(() => {
  calculateModalSize()
  window.addEventListener('resize', calculateModalSize)
  document.addEventListener('keydown', handleKeydown)

  // Set initial visibility
  visible.value = true
})

// Cleanup
onBeforeUnmount(() => {
  window.removeEventListener('resize', calculateModalSize)
  document.removeEventListener('keydown', handleKeydown)
})

// Initialize on visibility change
watch(() => props.currentIndex, () => {
  if (visible.value) {
    nextTick(() => {
      calculateModalSize()
    })
  }
})
</script>

<style scoped>
.preview-content {
  position: relative;
  padding: 16px;
}

.image-wrapper {
  text-align: center;
  margin-bottom: 16px;
}

.preview-image {
  max-width: 100%;
  max-height: v-bind(maxImageHeight + 'px');
  object-fit: contain;
  border-radius: 4px;
}

.no-image-large {
  width: 100%;
  height: 400px;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: #f0f0f0;
  border-radius: 4px;
}

.no-image-icon {
  font-size: 64px;
  margin-bottom: 16px;
}

.no-image-text {
  color: #8c8c8c;
  font-size: 16px;
}

.nav-buttons {
  position: absolute;
  top: 50%;
  left: 8px;
  right: 8px;
  transform: translateY(-50%);
  display: flex;
  justify-content: space-between;
  pointer-events: none;
}

.nav-button {
  pointer-events: all;
}

.preview-selection {
  position: absolute;
  top: 16px;
  right: 16px;
  background: rgba(255, 255, 255, 0.9);
  padding: 8px;
  border-radius: 4px;
}

.image-info {
  background: #fafafa;
  padding: 16px;
  border-radius: 4px;
  margin-bottom: 16px;
}

.info-row {
  display: flex;
  margin-bottom: 12px;
}

.info-row:last-child {
  margin-bottom: 0;
}

.info-label {
  min-width: 80px;
  font-weight: 500;
  color: #595959;
}

.info-value {
  flex: 1;
  word-break: break-all;
}

.tags-container {
  flex: 1;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.info-tag {
  cursor: pointer;
}

.preview-actions {
  display: flex;
  justify-content: center;
  gap: 8px;
}

.modal-header {
  display: flex;
  justify-content: flex-end;
  width: 100%;
}
</style>
