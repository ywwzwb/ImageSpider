<template>
  <div class="integrity-filter">
    <div class="filter-title" style="margin-bottom: 8px; font-weight: 500">
      完整性状态
    </div>
    <div class="checkbox-group">
      <label class="checkbox-item">
        <input
          type="checkbox"
          :value="ImageIntegrityStatus.UNKNOWN"
          v-model="selectedStatuses"
        />
        <span class="status-text">
          <a-tag :color="getIntegrityStatusColor(ImageIntegrityStatus.UNKNOWN)">未知</a-tag>
        </span>
      </label>
      <label class="checkbox-item">
        <input
          type="checkbox"
          :value="ImageIntegrityStatus.GOOD"
          v-model="selectedStatuses"
        />
        <span class="status-text">
          <a-tag :color="getIntegrityStatusColor(ImageIntegrityStatus.GOOD)">正常</a-tag>
        </span>
      </label>
      <label class="checkbox-item">
        <input
          type="checkbox"
          :value="ImageIntegrityStatus.BAD"
          v-model="selectedStatuses"
        />
        <span class="status-text">
          <a-tag :color="getIntegrityStatusColor(ImageIntegrityStatus.BAD)">破损</a-tag>
        </span>
      </label>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ImageIntegrityStatus } from '@/types/api'
import { getIntegrityStatusColor } from '@/utils/format'
import { useFilterStore } from '@/stores/filter'

const filterStore = useFilterStore()

// Use computed with getter/setter for v-model
const selectedStatuses = computed({
  get: () => filterStore.selectedStatusList,
  set: (values: number[]) => {
    // Clear all first
    filterStore.setStatus(ImageIntegrityStatus.UNKNOWN, false)
    filterStore.setStatus(ImageIntegrityStatus.GOOD, false)
    filterStore.setStatus(ImageIntegrityStatus.BAD, false)

    // Set new values
    values.forEach((value) => {
      filterStore.setStatus(value as ImageIntegrityStatus, true)
    })
  }
})
</script>

<style scoped>
.integrity-filter {
  padding: 12px 0;
  border-bottom: 1px solid #f0f0f0;
}

.checkbox-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.checkbox-item {
  display: flex;
  align-items: center;
  cursor: pointer;
}

.checkbox-item input[type="checkbox"] {
  margin-right: 8px;
}

.status-text {
  flex: 1;
}
</style>
