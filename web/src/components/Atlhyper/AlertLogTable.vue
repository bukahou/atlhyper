<template>
  <div class="alert-log-table">
    <!-- ✅ 表头区域 -->
    <div class="table-controls">
      <div class="table-title">异常告警日志 一览表</div>
      <div class="table-divider" />

      <div class="table-filter-bar">
        <div class="left-side">
          <span>显示</span>
          <el-select v-model="pageSize" size="small" class="page-size-selector">
            <el-option :label="10" :value="10" />
            <el-option :label="20" :value="20" />
            <el-option :label="50" :value="50" />
          </el-select>
          <span>条</span>
        </div>

        <div class="right-side">
          <span>最近</span>
          <el-select
            v-model="dateRange"
            size="small"
            class="date-range-selector"
          >
            <el-option label="1 天" value="1" />
            <el-option label="2 天" value="2" />
            <el-option label="3 天" value="3" />
            <el-option label="4 天" value="4" />
            <el-option label="5 天" value="5" />
            <el-option label="6 天" value="6" />
            <el-option label="7 天" value="7" />
          </el-select>
          <span>内日志</span>

          <!-- ✅ 导出按钮 -->
          <el-button
            type="primary"
            size="small"
            icon="el-icon-download"
            style="margin-left: 16px"
            @click="exportToExcel"
          >
            导出
          </el-button>
        </div>
      </div>
    </div>

    <!-- ✅ 表格区域 -->
    <el-table
      :data="pagedLogs"
      stripe
      border
      style="width: 100%; margin-top: 10px"
      :header-cell-style="{ background: '#f5f7fa', fontWeight: '600' }"
    >
      <!-- 筛选：category -->
      <el-table-column prop="category" label="category" min-width="120">
        <template #header>
          <el-select
            v-model="selectedCategory"
            placeholder="全部 category"
            clearable
            size="small"
            style="width: 100%"
          >
            <el-option
              v-for="item in categoryOptions"
              :key="item"
              :label="item"
              :value="item"
            />
          </el-select>
        </template>
      </el-table-column>

      <el-table-column prop="reason" label="reason" min-width="120" />

      <!-- 筛选：kind -->
      <el-table-column prop="kind" label="kind" min-width="100">
        <template #header>
          <el-select
            v-model="selectedKind"
            placeholder="全部 kind"
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

      <el-table-column prop="name" label="name" min-width="120" />

      <!-- 筛选：namespace -->
      <el-table-column prop="namespace" label="namespace" min-width="120">
        <template #header>
          <el-select
            v-model="selectedNamespace"
            placeholder="全部 namespace"
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

      <!-- 筛选：node -->
      <el-table-column prop="node" label="node" min-width="120">
        <template #header>
          <el-select
            v-model="selectedNode"
            placeholder="全部 node"
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

      <el-table-column
        prop="message"
        label="信息"
        min-width="200"
        show-overflow-tooltip
      />
      <el-table-column prop="timestamp" label="timestamp" min-width="160" />
    </el-table>

    <!-- ✅ 分页 -->
    <div class="pagination-wrapper">
      <el-pagination
        background
        layout="prev, pager, next"
        :total="filteredLogs.length"
        :page-size="pageSize"
        :current-page.sync="currentPage"
        small
      />
    </div>
  </div>
</template>

<script>
import * as XLSX from "xlsx";
import { saveAs } from "file-saver";

export default {
  name: "AlertLogTable",
  props: {
    logs: {
      type: Array,
      required: true,
    },
  },
  data() {
    return {
      pageSize: 10,
      currentPage: 1,
      dateRange: "3",
      selectedCategory: "",
      selectedKind: "",
      selectedNamespace: "",
      selectedNode: "",
    };
  },
  watch: {
    dateRange(newVal) {
      this.$emit("update-date-range", Number(newVal)); // 通知父组件选择了几天
    },
  },

  computed: {
    categoryOptions() {
      return [...new Set(this.logs.map((log) => log.category))].filter(Boolean);
    },
    kindOptions() {
      return [...new Set(this.logs.map((log) => log.kind))].filter(Boolean);
    },
    namespaceOptions() {
      return [...new Set(this.logs.map((log) => log.namespace))].filter(
        Boolean
      );
    },
    nodeOptions() {
      return [...new Set(this.logs.map((log) => log.node))].filter(Boolean);
    },
    filteredLogs() {
      return this.logs.filter((log) => {
        if (this.selectedCategory && log.category !== this.selectedCategory) {
          return false;
        }
        if (this.selectedKind && log.kind !== this.selectedKind) return false;
        if (
          this.selectedNamespace &&
          log.namespace !== this.selectedNamespace
        ) {
          return false;
        }
        if (this.selectedNode && log.node !== this.selectedNode) return false;
        return true;
      });
    },
    pagedLogs() {
      const start = (this.currentPage - 1) * this.pageSize;
      return this.filteredLogs.slice(start, start + this.pageSize);
    },
  },
  methods: {
    exportToExcel() {
      const data = this.pagedLogs.map((log) => ({
        category: log.category,
        reason: log.reason,
        kind: log.kind,
        name: log.name,
        namespace: log.namespace,
        node: log.node,
        message: log.message,
        timestamp: log.timestamp,
      }));

      const worksheet = XLSX.utils.json_to_sheet(data);
      const workbook = XLSX.utils.book_new();
      XLSX.utils.book_append_sheet(workbook, worksheet, "异常告警");

      const excelBuffer = XLSX.write(workbook, {
        bookType: "xlsx",
        type: "array",
      });
      const blob = new Blob([excelBuffer], {
        type: "application/octet-stream",
      });
      saveAs(
        blob,
        `异常告警日志_${new Date().toISOString().slice(0, 10)}.xlsx`
      );
    },
  },
};
</script>

<style scoped>
.alert-log-table {
  background: white;
  padding: 20px;
  border-radius: 12px;
  box-shadow: 0 4px 10px rgba(0, 0, 0, 0.04);
}

.table-controls {
  margin-bottom: 10px;
}

.table-title {
  font-size: 18px;
  font-weight: 600;
}

.table-divider {
  height: 1px;
  background-color: #dcdfe6;
  margin: 8px 0 12px;
}

.table-filter-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
}

.left-side,
.right-side {
  display: flex;
  align-items: center;
  gap: 6px;
}

.page-size-selector,
.date-range-selector {
  width: 80px;
}

.pagination-wrapper {
  margin-top: 16px;
  text-align: right;
}
</style>
