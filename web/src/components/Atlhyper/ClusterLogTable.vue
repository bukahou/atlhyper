<template>
  <div class="log-table-container">
    <!-- 🗞 标题 -->
    <div class="log-title">
      <h2>异常告警日志 一览表</h2>
      <hr>
    </div>

    <!-- 行数 + 时间范围 -->
    <div class="toolbar">
      <div class="row-size-selector">
        显示
        <el-select
          v-model="pageSize"
          class="row-size-dropdown"
          size="small"
          @change="handlePageSizeChange"
        >
          <el-option
            v-for="num in [5, 10, 20, 30]"
            :key="num"
            :label="num"
            :value="num"
          />
        </el-select>
        条
      </div>
      <el-select
        v-model="selectedDays"
        class="time-filter"
        placeholder="时间范围"
        size="small"
      >
        <el-option :label="'全部'" :value="0" />
        <el-option :label="'最近 1 天'" :value="1" />
        <el-option :label="'最近 2 天'" :value="2" />
        <el-option :label="'最近 3 天'" :value="3" />
      </el-select>
    </div>

    <!-- 📋 表格 -->
    <el-table
      :data="pagedLogs"
      border
      style="width: 100%"
      :header-cell-style="{
        background: '#f5f7fa',
        color: '#333',
        fontWeight: 600,
      }"
      empty-text="暂无日志数据"
    >
      <el-table-column prop="reason" label="原因" width="150">
        <template #header>
          <el-select
            v-model="selectedReason"
            placeholder="全部原因"
            clearable
            size="small"
            style="width: 100%"
          >
            <el-option
              v-for="item in reasonOptions"
              :key="item"
              :label="item"
              :value="item"
            />
          </el-select>
        </template>
      </el-table-column>

      <el-table-column
        prop="message"
        label="详细信息"
        min-width="300"
        show-overflow-tooltip
      >
        <template #header>
          <el-select
            v-model="selectedMessage"
            placeholder="全部信息"
            clearable
            size="small"
            style="width: 100%"
          >
            <el-option
              v-for="item in messageOptions"
              :key="item"
              :label="item"
              :value="item"
            />
          </el-select>
        </template>
      </el-table-column>

      <el-table-column prop="kind" label="资源类型" width="120">
        <template #header>
          <el-select
            v-model="selectedKind"
            placeholder="全部类型"
            clearable
            size="small"
            style="width: 100%"
          >
            <el-option
              v-for="item in kindOptions"
              :key="item"
              :label="item"
              :value="item"
            />
          </el-select>
        </template>
      </el-table-column>

      <el-table-column prop="name" label="资源名称" width="160">
        <template #header>
          <el-select
            v-model="selectedName"
            placeholder="资源名称"
            clearable
            size="small"
            style="width: 100%"
          >
            <el-option
              v-for="item in nameOptions"
              :key="item"
              :label="item"
              :value="item"
            />
          </el-select>
        </template>
      </el-table-column>

      <el-table-column prop="namespace" label="命名空间" width="140">
        <template #header>
          <el-select
            v-model="selectedNamespace"
            placeholder="命名空间"
            clearable
            size="small"
            style="width: 100%"
          >
            <el-option
              v-for="item in namespaceOptions"
              :key="item"
              :label="item"
              :value="item"
            />
          </el-select>
        </template>
      </el-table-column>

      <el-table-column prop="node" label="节点" width="140">
        <template #header>
          <el-select
            v-model="selectedNode"
            placeholder="节点"
            clearable
            size="small"
            style="width: 100%"
          >
            <el-option
              v-for="item in nodeOptions"
              :key="item"
              :label="item"
              :value="item"
            />
          </el-select>
        </template>
      </el-table-column>

      <el-table-column prop="timestamp" label="时间" width="180" />
    </el-table>

    <!-- 📄 分页 -->
    <el-pagination
      class="pagination"
      background
      small
      layout="prev, pager, next"
      :page-size="pageSize"
      :current-page="currentPage"
      :total="filteredLogs.length"
      @current-change="handlePageChange"
    />
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import dayjs from 'dayjs'

// 🔹 defineProps 不写类型，只用普通 JS
const props = defineProps({
  logs: Array
})

const selectedDays = ref(0)
const selectedReason = ref('')
const selectedMessage = ref('')
const selectedKind = ref('')
const selectedName = ref('')
const selectedNamespace = ref('')
const selectedNode = ref('')

// 🔹 各类筛选器选项（唯一值）
const reasonOptions = computed(() =>
  [...new Set(props.logs.map((l) => l.reason))].filter(Boolean)
)
const messageOptions = computed(() =>
  [...new Set(props.logs.map((l) => l.message))].filter(Boolean)
)
const kindOptions = computed(() =>
  [...new Set(props.logs.map((l) => l.kind))].filter(Boolean)
)
const nameOptions = computed(() =>
  [...new Set(props.logs.map((l) => l.name))].filter(Boolean)
)
const namespaceOptions = computed(() =>
  [...new Set(props.logs.map((l) => l.namespace))].filter(Boolean)
)
const nodeOptions = computed(() =>
  [...new Set(props.logs.map((l) => l.node))].filter(Boolean)
)

// 🔹 多字段过滤
const filteredLogs = computed(() => {
  const now = dayjs()
  return props.logs.filter((log) => {
    if (selectedDays.value > 0) {
      const t = dayjs(log.timestamp)
      if (!t.isValid() || now.diff(t, 'day') >= selectedDays.value) { return false }
    }
    if (selectedReason.value && log.reason !== selectedReason.value) { return false }
    if (selectedMessage.value && log.message !== selectedMessage.value) { return false }
    if (selectedKind.value && log.kind !== selectedKind.value) return false
    if (selectedName.value && log.name !== selectedName.value) return false
    if (selectedNamespace.value && log.namespace !== selectedNamespace.value) { return false }
    if (selectedNode.value && log.node !== selectedNode.value) return false
    return true
  })
})

// 🔹 分页逻辑
const currentPage = ref(1)
const pageSize = ref(10)

const handlePageSizeChange = (val) => {
  pageSize.value = val
  currentPage.value = 1
}

const handlePageChange = (page) => {
  currentPage.value = page
}

const pagedLogs = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  return filteredLogs.value.slice(start, start + pageSize.value)
})
</script>

<style scoped>
.log-table-container {
  background: white;
  padding: 20px;
  border-radius: 8px;
  margin-top: 20px;
  box-shadow: 0 2px 6px rgba(145, 54, 54, 0.05);
  min-height: 100px;
}

.log-title {
  margin-bottom: 16px;
}

.log-title h2 {
  font-size: 18px;
  font-weight: bold;
  color: #666;
  margin: 0;
  margin-bottom: 10px;
}

.log-title hr {
  border: none;
  border-top: 2px solid #c6ccd3;
  margin: 0;
  margin-bottom: 12px;
}

.toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.row-size-selector {
  font-size: 14px;
  color: #333;
}

.row-size-dropdown {
  width: 80px;
  margin: 0 6px;
}

.time-filter {
  width: 160px;
}

.pagination {
  margin-top: 20px;
  text-align: right;
}
</style>
