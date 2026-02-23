<template>
  <div class="image-grid">
    <!-- Batch operation bar -->
    <div v-if="appStore.batchMode" class="batch-bar">
      <a-space>
        <span>已选择 {{ appStore.selectedImages.size }} 项</span>
        <a-button size="small" @click="appStore.selectAll(currentImageIds)">
          全选当前页
        </a-button>
        <a-button size="small" @click="appStore.clearSelection()">
          取消选择
        </a-button>
        <a-popconfirm
          title="确定要删除选中的图片吗？"
          ok-text="删除"
          ok-type="danger"
          cancel-text="取消"
          @confirm="handleBatchDelete"
        >
          <a-button size="small" type="primary" danger>
            批量删除
          </a-button>
        </a-popconfirm>
        <a-popconfirm
          title="确定要重新下载选中的图片吗？"
          ok-text="重新下载"
          cancel-text="取消"
          @confirm="handleBatchRedownload"
        >
          <a-button size="small">批量重新下载</a-button>
        </a-popconfirm>
        <a-dropdown>
          <a-button size="small">
            批量标记状态
            <span style="margin-left: 4px">▼</span>
          </a-button>
          <template #overlay>
            <a-menu @click="handleBatchStatusChange">
              <a-menu-item key="0">
                <a-tag size="small" :color="getIntegrityStatusColor(0)">未知</a-tag>
              </a-menu-item>
              <a-menu-item key="1">
                <a-tag size="small" :color="getIntegrityStatusColor(1)">正常</a-tag>
              </a-menu-item>
              <a-menu-item key="-1">
                <a-tag size="small" :color="getIntegrityStatusColor(-1)">破损</a-tag>
              </a-menu-item>
            </a-menu>
          </template>
        </a-dropdown>
      </a-space>
    </div>

    <!-- Images grid -->
    <a-row :gutter="[16, 16]" style="margin-bottom: 24px">
      <a-col
        v-for="image in images"
        :key="image.id"
        :xs="24"
        :sm="12"
        :md="12"
        :lg="8"
        :xl="6"
        :xxl="4"
      >
        <ImageCard
          :image="image"
          @preview="handlePreview"
          @refresh="loadImages"
        />
      </a-col>
    </a-row>

    <!-- Loading state -->
    <div v-if="loading" class="loading-state">
      <a-spin size="large" />
    </div>

    <!-- Empty state -->
    <div v-if="!loading && (images || []).length === 0" class="empty-state">
      <a-empty description="暂无图片" />
    </div>

    <!-- Pagination -->
    <div v-if="total > 0" class="pagination-container">
      <a-pagination
        v-model:current="currentPage"
        :total="total"
        :page-size="pageSize"
        show-size-changer
        :page-size-options="['24', '48', '96']"
        show-quick-jumper
        :show-total="(total: number) => `共 ${total} 条`"
        @change="handlePageChange"
        @showSizeChange="handlePageSizeChange"
      />
    </div>

    <!-- Image preview modal -->
    <ImagePreview
      v-model:visible="previewVisible"
      v-model:current-index="previewIndex"
      :images="images"
      @refresh="loadImages"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { message } from 'ant-design-vue'
import ImageCard from './ImageCard.vue'
import ImagePreview from './ImagePreview.vue'
import { useAppStore } from '@/stores/app'
import { useSourceStore } from '@/stores/source'
import { useFilterStore } from '@/stores/filter'
import { imageApi } from '@/api/image'
import type { ImageMeta } from '@/types/api'
import { getIntegrityStatusColor } from '@/utils/format'

const appStore = useAppStore()
const sourceStore = useSourceStore()
const filterStore = useFilterStore()

// State
const images = ref<ImageMeta[]>([])
const total = ref(0)
const currentPage = ref(1)
const pageSize = ref(48)
const loading = ref(false)
const previewVisible = ref(false)
const previewIndex = ref(0)

// Computed
const currentImageIds = computed(() => images.value.map(img => img.id))

// Watch for filter changes
watch(
  [
    () => sourceStore.currentSource,
    () => filterStore.selectedTagList,
    () => filterStore.selectedStatusList
  ],
  () => {
    currentPage.value = 1
    loadImages()
  }
)

// Load images
async function loadImages() {
  if (!sourceStore.currentSource) return

  loading.value = true
  try {
    const offset = (currentPage.value - 1) * pageSize.value
    const response = await imageApi.getImages(
      sourceStore.currentSource,
      {
        tags: filterStore.selectedTagList,
        integrityStatus: filterStore.selectedStatusList,
        offset,
        limit: pageSize.value
      }
    )

    images.value = response.imageList
    total.value = response.totalCount

    // Clear selection for images that no longer exist
    const existingIds = new Set(images.value.map(img => img.id))
    appStore.selectedImages.forEach((id: string) => {
      if (!existingIds.has(id)) {
        appStore.selectedImages.delete(id)
      }
    })
  } catch (error) {
    appStore.showError('加载图片失败')
    console.error('Failed to load images:', error)
  } finally {
    loading.value = false
  }
}

// Pagination handlers
function handlePageChange(page: number, newPageSize: number) {
  currentPage.value = page
  pageSize.value = newPageSize
  loadImages()
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

function handlePageSizeChange(_current: number, size: number) {
  pageSize.value = size
  currentPage.value = 1
  loadImages()
}

// Batch operations
async function handleBatchDelete() {
  const ids = appStore.getSelectedIds()
  if (ids.length === 0) {
    message.warning('请先选择图片')
    return
  }

  try {
    const response = await imageApi.batchDelete(sourceStore.currentSource, ids)
    if (response.deleted && response.deleted > 0) {
      message.success(`成功删除 ${response.deleted} 张图片`)
      appStore.clearSelection()
      loadImages()
    } else {
      message.error('删除失败')
    }
  } catch (error) {
    console.error('Failed to batch delete:', error)
    message.error('删除失败')
  }
}

async function handleBatchRedownload() {
  const ids = appStore.getSelectedIds()
  if (ids.length === 0) {
    message.warning('请先选择图片')
    return
  }

  try {
    const response = await imageApi.batchRedownload(sourceStore.currentSource, ids)
    if (response.redownloaded && response.redownloaded > 0) {
      message.success(`已标记 ${response.redownloaded} 张图片重新下载`)
      appStore.clearSelection()
      loadImages()
    } else {
      message.error('重新下载失败')
    }
  } catch (error) {
    console.error('Failed to batch redownload:', error)
    message.error('重新下载失败')
  }
}

async function handleBatchStatusChange(menuInfo: { key: string }) {
  const ids = appStore.getSelectedIds()
  if (ids.length === 0) {
    message.warning('请先选择图片')
    return
  }

  const status = parseInt(menuInfo.key, 10)
  const statusText = status === 0 ? '未知' : status === 1 ? '正常' : '破损'

  try {
    const response = await imageApi.batchUpdateStatus(sourceStore.currentSource, ids, status)
    if (response.updated && response.updated > 0) {
      message.success(`已将 ${response.updated} 张图片标记为${statusText}`)
      appStore.clearSelection()
      loadImages()
    } else {
      message.error('状态更新失败')
    }
  } catch (error) {
    console.error('Failed to batch update status:', error)
    message.error('状态更新失败')
  }
}

// Preview
function handlePreview(image: ImageMeta) {
  const index = images.value.findIndex(img => img.id === image.id)
  if (index !== -1) {
    previewIndex.value = index
    previewVisible.value = true
  }
}

// Initial load
loadImages()

defineExpose({ loadImages })
</script>

<style scoped>
.image-grid {
  width: 100%;
}

.batch-bar {
  background: #e6f7ff;
  border: 1px solid #91d5ff;
  border-radius: 4px;
  padding: 12px 16px;
  margin-bottom: 16px;
}

.loading-state {
  text-align: center;
  padding: 48px;
}

.empty-state {
  text-align: center;
  padding: 48px;
}

.pagination-container {
  display: flex;
  justify-content: center;
  margin-top: 24px;
}
</style>
