<template>
  <div class="tag-filter">
    <div class="filter-title" style="margin-bottom: 8px; font-weight: 500">
      标签筛选
      <span v-if="selectedTags.length > 0" class="selected-count">({{ selectedTags.length }})</span>
    </div>

    <!-- Selected tags section -->
    <div v-if="selectedTags.length > 0" class="selected-tags-section">
      <div class="selected-tags-header">
        <span class="section-label">已选中</span>
        <a-button type="link" size="small" @click="clearSelectedTags">
          清空
        </a-button>
      </div>
      <div class="selected-tags-list">
        <div
          v-for="tag in selectedTags"
          :key="tag.tag"
          class="tag-item selected"
          @click="handleTagClick(tag.tag)"
        >
          <span class="tag-name">{{ tag.tag }}</span>
          <span class="tag-count">({{ tag.count }})</span>
          <span class="remove-icon">✕</span>
        </div>
      </div>
      <a-divider style="margin: 12px 0" />
    </div>

    <!-- All tags section -->
    <div class="tag-list">
      <div
        v-for="tag in unselectedTags"
        :key="tag.tag"
        class="tag-item"
        :class="{ selected: filterStore.hasTag(tag.tag) }"
        @click="handleTagClick(tag.tag)"
      >
        <span class="tag-name">{{ tag.tag }}</span>
        <span class="tag-count">({{ tag.count }})</span>
      </div>
      <div v-if="filterStore.loading" class="loading-more">
        <a-spin size="small" />
      </div>
      <div
        v-else-if="filterStore.hasMoreTags"
        class="load-more"
        @click="loadMore"
      >
        加载更多...
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useFilterStore } from '@/stores/filter'
import { useSourceStore } from '@/stores/source'
import type { TagInfo } from '@/types/api'

const filterStore = useFilterStore()
const sourceStore = useSourceStore()

// Build a map of loaded tags for quick lookup
const loadedTagMap = computed(() => {
  const map = new Map<string, number>()
  for (const tag of filterStore.tags) {
    map.set(tag.tag, tag.count)
  }
  return map
})

// Computed: all selected tags (including those not loaded yet)
const selectedTags = computed(() => {
  const selected: TagInfo[] = []
  for (const tagName of filterStore.selectedTagList) {
    const count = loadedTagMap.value.get(tagName) || 0
    selected.push({ tag: tagName, count })
  }
  return selected
})

const unselectedTags = computed(() => {
  return filterStore.tags.filter(tag => !filterStore.hasTag(tag.tag))
})

function handleTagClick(tag: string) {
  filterStore.toggleTag(tag)
}

function clearSelectedTags() {
  filterStore.reset()
}

async function loadMore() {
  if (sourceStore.currentSource) {
    await filterStore.loadTags(sourceStore.currentSource, true)
  }
}
</script>

<style scoped>
.tag-filter {
  padding: 12px 0;
}

.filter-title {
  display: flex;
  align-items: center;
  gap: 8px;
}

.selected-count {
  color: #1890ff;
  font-size: 14px;
}

.selected-tags-section {
  margin-bottom: 8px;
}

.selected-tags-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.section-label {
  font-size: 12px;
  color: #8c8c8c;
}

.selected-tags-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.tag-list {
  max-height: 300px;
  overflow-y: auto;
}

.tag-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.2s;
  border: 1px solid transparent;
}

.tag-item:hover {
  background-color: #f5f5f5;
}

.tag-item.selected {
  background-color: #e6f7ff;
  border-color: #91d5ff;
}

.tag-item.selected:hover {
  background-color: #bae7ff;
}

.tag-name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tag-count {
  color: #8c8c8c;
  font-size: 12px;
  margin-left: 8px;
}

.remove-icon {
  margin-left: 8px;
  color: #ff4d4f;
  font-size: 12px;
  opacity: 0;
  transition: opacity 0.2s;
}

.tag-item.selected:hover .remove-icon {
  opacity: 1;
}

.load-more,
.loading-more {
  text-align: center;
  padding: 12px;
  color: #1890ff;
  cursor: pointer;
  transition: color 0.2s;
}

.load-more:hover {
  color: #40a9ff;
}
</style>
