<template>
  <div class="tag-filter">
    <div class="filter-title" style="margin-bottom: 8px; font-weight: 500">
      标签筛选
    </div>
    <div class="tag-list">
      <div
        v-for="tag in filterStore.tags"
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
import { useFilterStore } from '@/stores/filter'
import { useSourceStore } from '@/stores/source'

const filterStore = useFilterStore()
const sourceStore = useSourceStore()

function handleTagClick(tag: string) {
  filterStore.toggleTag(tag)
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

.tag-list {
  max-height: 400px;
  overflow-y: auto;
}

.tag-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  margin-bottom: 4px;
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
