<template>
  <div class="app">
    <a-layout style="min-height: 100vh">
      <a-layout-header class="header">
        <div class="logo">
          <h1 style="color: white; margin: 0">ImageSpider</h1>
        </div>
        <div class="source-selector">
          <a-select
            v-model:value="selectedSource"
            style="width: 200px"
            placeholder="选择图片源"
            :loading="sourceStore.loading"
          >
            <a-select-option
              v-for="source in sourceStore.sources"
              :key="source"
              :value="source"
            >
              {{ source }}
            </a-select-option>
          </a-select>
        </div>
        <div class="header-actions">
          <!-- Mobile filter toggle button -->
          <a-button
            v-if="isMobile"
            class="filter-toggle-btn"
            @click="showFilterDrawer = true"
          >
            <template #icon>🔍</template>
            筛选
          </a-button>
          <a-button
            :type="appStore.batchMode ? 'primary' : 'default'"
            @click="appStore.toggleBatchMode"
          >
            {{ appStore.batchMode ? '退出批量' : '批量模式' }}
          </a-button>
        </div>
      </a-layout-header>
      <a-layout style="padding: 24px">
        <a-row :gutter="24">
          <!-- Desktop sidebar -->
          <a-col v-if="!isMobile" :span="6">
            <a-card title="筛选器" :loading="filterStore.loading">
              <IntegrityFilter />
              <a-divider style="margin: 16px 0" />
              <TagFilter />
            </a-card>
          </a-col>
          <a-col :span="isMobile ? 24 : 18">
            <ImageGrid ref="imageGridRef" />
          </a-col>
        </a-row>
      </a-layout>
    </a-layout>

    <!-- Mobile filter drawer -->
    <a-drawer
      v-model:open="showFilterDrawer"
      title="筛选器"
      placement="left"
      :width="drawerWidth"
      :closable="true"
    >
      <div class="drawer-content">
        <IntegrityFilter />
        <a-divider style="margin: 16px 0" />
        <TagFilter />
      </div>
      <div class="drawer-footer">
        <a-button type="primary" block @click="showFilterDrawer = false">
          完成
        </a-button>
      </div>
    </a-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed, onMounted, onBeforeUnmount } from 'vue'
import { useAppStore } from './stores/app'
import { useSourceStore } from './stores/source'
import { useFilterStore } from './stores/filter'
import IntegrityFilter from './components/IntegrityFilter.vue'
import TagFilter from './components/TagFilter.vue'
import ImageGrid from './components/ImageGrid.vue'

const appStore = useAppStore()
const sourceStore = useSourceStore()
const filterStore = useFilterStore()

const selectedSource = ref<string>('')
const imageGridRef = ref<InstanceType<typeof ImageGrid>>()
const showFilterDrawer = ref(false)
const windowWidth = ref(window.innerWidth)

// Computed
const isMobile = computed(() => windowWidth.value < 768)
const drawerWidth = computed(() => Math.min(windowWidth.value * 0.85, 400))

// Handle window resize
function handleResize() {
  windowWidth.value = window.innerWidth
}

onMounted(() => {
  window.addEventListener('resize', handleResize)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', handleResize)
})

// Load sources on mount
sourceStore.loadSources()

// Watch for source changes
watch(selectedSource, (newSource) => {
  if (newSource) {
    sourceStore.setCurrentSource(newSource)
    filterStore.loadTags(newSource)
  }
})

// Watch for source store changes to sync select
watch(() => sourceStore.currentSource, (newSource) => {
  if (newSource && newSource !== selectedSource.value) {
    selectedSource.value = newSource
  }
})
</script>

<style>
.header {
  display: flex;
  align-items: center;
  padding: 0 24px;
}

.logo {
  margin-right: 24px;
}

.logo h1 {
  font-size: 20px;
}

.source-selector {
  flex: 1;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.filter-toggle-btn {
  display: none;
}

.drawer-content {
  padding-bottom: 60px;
}

.drawer-footer {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  padding: 16px;
  background: #fff;
  border-top: 1px solid #f0f0f0;
}

/* Mobile responsive styles */
@media (max-width: 767px) {
  .header {
    padding: 0 12px;
  }

  .logo {
    margin-right: 12px;
  }

  .logo h1 {
    font-size: 16px;
  }

  .source-selector {
    flex: 1;
    min-width: 0;
  }

  .source-selector .ant-select {
    width: 100% !important;
  }

  .filter-toggle-btn {
    display: inline-flex;
  }

  .header-actions {
    gap: 8px;
  }

  .header-actions .ant-btn {
    padding: 0 8px;
    font-size: 12px;
  }
}
</style>