<template>
  <a-modal
    :visible="props.visible"
    :footer="null"
    :width="modalWidth"
    :centered="true"
    :mask-closable="true"
    :keyboard="true"
    @cancel="handleCancel"
  >
    <div class="preview-content" @click.self="handleCancel">
      <!-- Image container with fixed positioning context -->
      <div class="image-container" :style="{ height: maxImageHeight + 'px' }">
        <!-- Image wrapper for scrolling when zoomed -->
        <div class="image-wrapper" :class="{ zoomed: isZoomed }">
          <img
            v-if="currentImage?.localPath && fullImageUrl && !imageError"
            :src="fullImageUrl"
            :alt="currentImage?.id"
            class="preview-image"
            :class="{ zoomed: isZoomed }"
            :style="{ maxHeight: isZoomed ? 'none' : maxImageHeight + 'px' }"
            @error="handleImageError"
            @click="toggleZoom"
          >
          <div v-else class="no-image-large">
            <div class="no-image-content">
              <div class="no-image-icon">📷</div>
              <div class="no-image-text">无图片</div>
            </div>
          </div>
        </div>

        <!-- Zoom hint - positioned relative to container -->
        <div v-if="currentImage?.localPath && fullImageUrl && !imageError" class="zoom-hint" @click="toggleZoom">
          {{ isZoomed ? '点击缩小' : '点击放大' }}
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
          :disabled="currentIndex >= (images || []).length - 1"
          @click.stop="navigateNext"
        >
          <template #icon>›</template>
        </a-button>
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
          <a-dropdown>
            <a-button>
              标记状态
              <span style="margin-left: 4px">▼</span>
            </a-button>
            <template #overlay>
              <a-menu @click="handleStatusChange">
                <a-menu-item key="0">
                  <a-tag :color="getIntegrityStatusColor(ImageIntegrityStatus.UNKNOWN)">未知</a-tag>
                </a-menu-item>
                <a-menu-item key="1">
                  <a-tag :color="getIntegrityStatusColor(ImageIntegrityStatus.GOOD)">正常</a-tag>
                </a-menu-item>
                <a-menu-item key="-1">
                  <a-tag :color="getIntegrityStatusColor(ImageIntegrityStatus.BAD)">破损</a-tag>
                </a-menu-item>
              </a-menu>
            </template>
          </a-dropdown>
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
import { ref, computed, watch, nextTick, onMounted, onBeforeUnmount, h } from 'vue'
import { useSourceStore } from '@/stores/source'
import { useFilterStore } from '@/stores/filter'
import { imageApi } from '@/api/image'
import { tagApi } from '@/api/tag'
import type { ImageMeta } from '@/types/api'
import { ImageIntegrityStatus } from '@/types/api'
import { message, Modal } from 'ant-design-vue'
import { getIntegrityStatusText, getIntegrityStatusColor, formatDate } from '@/utils/format'

interface Props {
  images: ImageMeta[]
  currentIndex: number
  visible: boolean
}

interface Emits {
  (e: 'update:visible', value: boolean): void
  (e: 'update:currentIndex', value: number): void
  (e: 'refresh'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const sourceStore = useSourceStore()
const filterStore = useFilterStore()

// State
const modalWidth = ref(800)
const maxImageHeight = ref(600)
const isZoomed = ref(false)
const imageError = ref(false)

// Computed
const currentImage = computed(() => props.images?.[props.currentIndex] || null)
const hasMultipleImages = computed(() => (props.images || []).length > 1)
const fullImageUrl = computed(() => {
  if (!currentImage.value?.localPath) return null
  return `/image/${currentImage.value.localPath.replace(/^\/image\//, '')}`
})

// Keyboard navigation
function handleKeydown(event: KeyboardEvent) {
  if (!props.visible) return

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
  if (props.currentIndex < (props.images || []).length - 1) {
    emit('update:currentIndex', props.currentIndex + 1)
  }
}

// Event handlers
function handleCancel() {
  emit('update:visible', false)
}

function handleTagClick(tag: string) {
  if (!filterStore.hasTag(tag)) {
    filterStore.addTag(tag)
  }
}

async function handleStatusChange(menuInfo: { key: string }) {
  if (!currentImage.value) return

  const status = parseInt(menuInfo.key, 10) as ImageIntegrityStatus
  try {
    const response = await imageApi.batchUpdateStatus(sourceStore.currentSource, [currentImage.value.id], status)
    if (response.updated && response.updated > 0) {
      message.success('状态更新成功')
      emit('refresh')
    } else {
      message.error('状态更新失败')
    }
  } catch (error) {
    console.error('Failed to update image status:', error)
    message.error('状态更新失败')
  }
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
        const imageCount = (props.images || []).length
        if (imageCount === 0) {
          emit('update:visible', false)
        } else if (props.currentIndex >= imageCount) {
          emit('update:currentIndex', imageCount - 1)
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

function handleSetCover() {
  if (!currentImage.value || !(currentImage.value.tags || []).length) {
    message.warning('该图片没有标签')
    return
  }

  const tags = currentImage.value.tags
  let selectedTag = tags[0]

  // 创建选择标签的 Modal
  Modal.confirm({
    title: '选择要设置为封面的标签',
    content: () => h('div', { style: 'padding: 16px 0' }, [
      h('select', {
        style: {
          width: '100%',
          padding: '8px 12px',
          borderRadius: '4px',
          border: '1px solid #d9d9d9',
          fontSize: '14px'
        },
        onChange: (e: Event) => {
          selectedTag = (e.target as HTMLSelectElement).value
        }
      }, tags.map(tag => h('option', { value: tag, key: tag }, tag)))
    ]),
    okText: '设为封面',
    cancelText: '取消',
    async onOk() {
      try {
        await tagApi.setTagCover(sourceStore.currentSource, selectedTag, currentImage.value!.id)
        message.success(`已设置为「${selectedTag}」的封面`)
      } catch (error) {
        console.error('Failed to set tag cover:', error)
        message.error('设置封面失败')
      }
    }
  })
}

function handleImageError() {
  console.error('Failed to load image:', currentImage.value?.id)
  imageError.value = true
}

function toggleZoom() {
  isZoomed.value = !isZoomed.value
}

// Calculate modal size based on viewport
function calculateModalSize() {
  const viewportWidth = window.innerWidth
  const viewportHeight = window.innerHeight

  // Modal takes up to 90% of viewport width
  modalWidth.value = Math.min(viewportWidth * 0.9, 1200)
  // Image area takes up to 50% of viewport height (leave room for info/actions)
  maxImageHeight.value = viewportHeight * 0.5
}

// Watch for visibility changes
watch(() => props.currentIndex, () => {
  // Reset zoom and error when image changes
  isZoomed.value = false
  imageError.value = false
  // Scroll to top when image changes
  const content = document.querySelector('.preview-content')
  if (content) {
    content.scrollTop = 0
  }
})

// Reset zoom when modal closes
watch(() => props.visible, (visible) => {
  if (!visible) {
    isZoomed.value = false
  }
})

// Mount
onMounted(() => {
  calculateModalSize()
  window.addEventListener('resize', calculateModalSize)
  document.addEventListener('keydown', handleKeydown)
})

// Cleanup
onBeforeUnmount(() => {
  window.removeEventListener('resize', calculateModalSize)
  document.removeEventListener('keydown', handleKeydown)
})

// Initialize on visibility change
watch(() => props.currentIndex, () => {
  if (props.visible) {
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

.image-container {
  position: relative;
  margin-bottom: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.image-wrapper {
  text-align: center;
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.image-wrapper.zoomed {
  overflow: auto;
  display: block;
}

.preview-image {
  max-width: 100%;
  width: auto;
  height: auto;
  object-fit: contain;
  border-radius: 4px;
  cursor: zoom-in;
  transition: all 0.3s ease;
  display: block;
  margin: 0 auto;
}

.preview-image.zoomed {
  max-width: none;
  cursor: zoom-out;
}

.zoom-hint {
  position: absolute;
  bottom: 8px;
  right: 8px;
  background: rgba(0, 0, 0, 0.6);
  color: white;
  padding: 8px 12px;
  border-radius: 4px;
  font-size: 14px;
  cursor: pointer;
  opacity: 0.8;
  transition: opacity 0.2s;
  z-index: 10;
}

.zoom-hint:hover {
  opacity: 1;
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
  flex-wrap: wrap;
}

/* Ensure modal fits viewport */
:deep(.ant-modal) {
  max-height: 90vh;
  top: 5vh;
  margin: 0 auto;
}

:deep(.ant-modal-content) {
  max-height: 90vh;
  display: flex;
  flex-direction: column;
}

:deep(.ant-modal-body) {
  max-height: calc(90vh - 110px);
  overflow-y: auto;
  padding: 16px;
}
</style>
