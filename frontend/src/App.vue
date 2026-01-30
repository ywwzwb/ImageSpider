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
        <div class="batch-toggle">
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
          <a-col :span="6">
            <a-card title="筛选器" :loading="filterStore.loading">
              <IntegrityFilter />
              <a-divider style="margin: 16px 0" />
              <TagFilter />
            </a-card>
          </a-col>
          <a-col :span="18">
            <ImageGrid ref="imageGridRef" />
          </a-col>
        </a-row>
      </a-layout>
    </a-layout>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
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

// No need to set up message function anymore


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

.source-selector {
  flex: 1;
}

.batch-toggle {
  margin-left: 24px;
}
</style>