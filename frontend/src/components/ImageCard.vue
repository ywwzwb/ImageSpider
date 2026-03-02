<template>
  <div class="image-card" :class="{ selected: isSelected }">
    <div class="image-container" @click="handleImageClick">
      <img
        v-if="image.localPath && thumbnailUrl"
        :src="thumbnailUrl"
        :alt="image.id"
        class="thumbnail"
        @error="handleImageError"
      >
      <div v-else class="no-image">
        <div class="no-image-content">
          <div class="no-image-icon">📷</div>
          <div class="no-image-text">无缩略图</div>
        </div>
      </div>

      <!-- Image ID -->
      <div class="image-id">
        <a-tag size="small">{{ truncateText(image.id, 8) }}</a-tag>
      </div>

      <!-- Selection checkbox -->
      <div v-if="appStore.batchMode" class="selection-checkbox">
        <a-checkbox
          :checked="isSelected"
          @click.stop
          @change="handleSelectionChange"
        />
      </div>

      <!-- Integrity status -->
      <div class="integrity-status">
        <a-tag
          :color="getIntegrityStatusColor(image.integrityStatus)"
          size="small"
        >
          {{ getIntegrityStatusText(image.integrityStatus) }}
        </a-tag>
      </div>

      <!-- Tags -->
      <div v-if="(image.tags || []).length > 0" class="tags-preview">
        <template v-for="tag in visibleTags" :key="tag">
          <a-tag
            size="small"
            class="tag-chip"
            @click.stop="handleTagClick(tag)"
          >
            {{ truncateText(tag, 10) }}
          </a-tag>
        </template>
        <a-tag
          v-if="hiddenTagCount > 0"
          size="small"
          class="tag-more"
          @click.stop="$emit('preview', image)"
        >
          +{{ hiddenTagCount }}
        </a-tag>
      </div>
    </div>

    <!-- Actions -->
    <div v-if="showActions" class="card-actions">
      <a-tooltip title="删除">
        <a-button
          type="text"
          size="small"
          danger
          @click.stop="handleDelete"
        >
          <template #icon>🗑️</template>
        </a-button>
      </a-tooltip>
      <a-tooltip title="重新下载">
        <a-button
          type="text"
          size="small"
          @click.stop="handleRedownload"
        >
          <template #icon>🔄</template>
        </a-button>
      </a-tooltip>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { ImageMeta } from '@/types/api'
import { useAppStore } from '@/stores/app'
import { useFilterStore } from '@/stores/filter'
import { useSourceStore } from '@/stores/source'
import { getIntegrityStatusText, getIntegrityStatusColor, getThumbnailPath, truncateText } from '@/utils/format'
import { imageApi } from '@/api/image'
import { message, Modal } from 'ant-design-vue'

interface Props {
  image: ImageMeta
  showActions?: boolean
}

interface Emits {
  (e: 'preview', image: ImageMeta): void
  (e: 'refresh'): void
}

const props = withDefaults(defineProps<Props>(), {
  showActions: true
})

const emit = defineEmits<Emits>()

const appStore = useAppStore()
const filterStore = useFilterStore()
const sourceStore = useSourceStore()

// Computed
const isSelected = computed(() => appStore.isImageSelected(props.image.id))
const thumbnailUrl = computed(() => getThumbnailPath(props.image.localPath))
const visibleTags = computed(() => (props.image.tags || []).slice(0, 3))
const hiddenTagCount = computed(() => Math.max(0, (props.image.tags || []).length - 3))

// Methods
function handleImageClick() {
  if (appStore.batchMode) {
    appStore.toggleImageSelection(props.image.id)
  } else {
    emit('preview', props.image)
  }
}

function handleSelectionChange() {
  appStore.toggleImageSelection(props.image.id)
}

function handleTagClick(tag: string) {
  if (!filterStore.hasTag(tag)) {
    filterStore.addTag(tag)
  }
}

function handleImageError() {
  console.error('Failed to load image:', props.image.id)
}

function handleDelete() {
  Modal.confirm({
    title: '确认删除',
    content: `确定要删除图片 ${props.image.id} 吗？`,
    okText: '删除',
    okType: 'danger',
    cancelText: '取消',
    onOk: async () => {
      const response = await imageApi.batchDelete(sourceStore.currentSource, [props.image.id])
      if (response.deleted && response.deleted > 0) {
        message.success('删除成功')
        emit('refresh')
      } else {
        message.error('删除失败')
      }
    },
    onCancel: () => {
      // 用户点击取消，不执行任何操作
    }
  })
}

function handleRedownload() {
  Modal.confirm({
    title: '确认重新下载',
    content: `确定要重新下载图片 ${props.image.id} 吗？`,
    okText: '重新下载',
    cancelText: '取消',
    onOk: async () => {
      const response = await imageApi.batchRedownload(sourceStore.currentSource, [props.image.id])
      if (response.redownloaded && response.redownloaded > 0) {
        message.success('已标记为重新下载')
        emit('refresh')
      } else {
        message.error('重新下载失败')
      }
    },
    onCancel: () => {
      // 用户点击取消，不执行任何操作
    }
  })
}
</script>

<style scoped>
.image-card {
  border: 1px solid #d9d9d9;
  border-radius: 8px;
  overflow: hidden;
  background: white;
  transition: all 0.2s;
}

.image-card:hover {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
}

.image-card.selected {
  border-color: #1890ff;
  box-shadow: 0 0 0 2px rgba(24, 144, 255, 0.1);
}

.image-container {
  position: relative;
  aspect-ratio: 1;
  overflow: hidden;
  cursor: pointer;
  background-color: #f5f5f5;
}

.thumbnail {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.no-image {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: #f0f0f0;
}

.no-image-content {
  text-align: center;
}

.no-image-icon {
  font-size: 48px;
  margin-bottom: 8px;
}

.no-image-text {
  color: #8c8c8c;
  font-size: 14px;
}

.image-id {
  position: absolute;
  top: 8px;
  left: 8px;
}

.selection-checkbox {
  position: absolute;
  top: 8px;
  right: 8px;
}

.integrity-status {
  position: absolute;
  bottom: 8px;
  left: 8px;
}

.tags-preview {
  position: absolute;
  bottom: 8px;
  left: 8px;
  right: 8px;
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.tag-chip {
  cursor: pointer;
}

.tag-chip:hover {
  opacity: 0.8;
}

.tag-more {
  cursor: pointer;
  background-color: rgba(0, 0, 0, 0.6);
  color: white;
}

.card-actions {
  display: flex;
  justify-content: flex-end;
  padding: 8px;
  gap: 4px;
}
</style>
