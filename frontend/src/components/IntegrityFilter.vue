<template>
  <div class="integrity-filter">
    <div class="filter-title" style="margin-bottom: 8px; font-weight: 500">
      完整性状态
    </div>
    <a-checkbox-group
      :value="selectedStatusList"
      @change="handleStatusChange"
      style="display: flex; flex-direction: column; gap: 8px"
    >
      <a-checkbox :value="ImageIntegrityStatus.UNKNOWN">
        <a-tag :color="getIntegrityStatusColor(ImageIntegrityStatus.UNKNOWN)">
          未知
        </a-tag>
      </a-checkbox>
      <a-checkbox :value="ImageIntegrityStatus.GOOD">
        <a-tag :color="getIntegrityStatusColor(ImageIntegrityStatus.GOOD)">
          正常
        </a-tag>
      </a-checkbox>
      <a-checkbox :value="ImageIntegrityStatus.BAD">
        <a-tag :color="getIntegrityStatusColor(ImageIntegrityStatus.BAD)">
          破损
        </a-tag>
      </a-checkbox>
    </a-checkbox-group>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ImageIntegrityStatus } from '@/types/api'
import { getIntegrityStatusColor } from '@/utils/format'
import { useFilterStore } from '@/stores/filter'

const filterStore = useFilterStore()

const selectedStatusList = computed(() => filterStore.selectedStatusList)

function handleStatusChange(checkedValues: number[]) {
  // Clear all first
  filterStore.setStatus(ImageIntegrityStatus.UNKNOWN, false)
  filterStore.setStatus(ImageIntegrityStatus.GOOD, false)
  filterStore.setStatus(ImageIntegrityStatus.BAD, false)

  // Set checked ones
  checkedValues.forEach((value) => {
    filterStore.setStatus(value as ImageIntegrityStatus, true)
  })
}
</script>

<style scoped>
.integrity-filter {
  padding: 12px 0;
  border-bottom: 1px solid #f0f0f0;
}
</style>
