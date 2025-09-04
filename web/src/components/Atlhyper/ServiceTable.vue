<template>
  <div class="service-table-container">
    <div class="table-title">
      <h2>Service List</h2>
      <hr />
    </div>

    <!-- 分页控制 -->
    <div class="toolbar">
      <div class="row-size-selector">
        Show
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
        items
      </div>
    </div>

    <!-- 表格展示 -->
    <el-table
      :data="pagedServices"
      border
      style="width: 100%"
      :header-cell-style="{
        background: '#f5f7fa',
        color: '#333',
        fontWeight: 600,
      }"
      empty-text="No Service data available"
    >
      <!-- 名称筛选 -->
      <el-table-column prop="name" label="名称" min-width="160">
        <template #header>
          <el-select
            v-model="selectedName"
            placeholder="Name"
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

      <!-- 命名空间筛选（修正了错误的结束标签） -->
      <el-table-column prop="namespace" label="Namespace" width="120">
        <template #header>
          <el-select
            v-model="selectedNamespace"
            placeholder="Namespace"
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

      <!-- 类型筛选 -->
      <el-table-column prop="type" label="Type" width="120">
        <template #header>
          <el-select
            v-model="selectedType"
            placeholder="Type"
            clearable
            size="small"
            style="width: 100%"
          >
            <el-option
              v-for="item in typeOptions"
              :key="item"
              :label="item"
              :value="item"
            />
          </el-select>
        </template>
      </el-table-column>

      <el-table-column prop="clusterIP" label="Cluster IP" width="160" />
      <el-table-column prop="ports" label="ports" min-width="140" />
      <el-table-column prop="protocol" label="protocol" width="100" />
      <el-table-column prop="selector" label="selector" min-width="180" />
      <el-table-column prop="createTime" label="createTime" width="180" />

      <!-- 操作 -->
      <el-table-column label="Actions" fixed="right" width="120">
        <template slot-scope="{ row }">
          <el-button
            size="mini"
            type="primary"
            plain
            :style="{ padding: '4px 8px', fontSize: '12px' }"
            icon="el-icon-view"
            @click="$emit('view', row)"
          >
            View
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-pagination
      class="pagination"
      background
      small
      layout="prev, pager, next"
      :page-size="pageSize"
      :current-page="currentPage"
      :total="filteredServices.length"
      @current-change="handlePageChange"
    />
  </div>
</template>

<script>
export default {
  name: "ServiceTable",
  props: {
    services: { type: Array, required: true },
  },
  data() {
    return {
      selectedName: "",
      selectedNamespace: "",
      selectedType: "",
      pageSize: 10,
      currentPage: 1,
    };
  },
  computed: {
    nameOptions() {
      return [...new Set(this.services.map((s) => s.name))].filter(Boolean);
    },
    namespaceOptions() {
      return [...new Set(this.services.map((s) => s.namespace))].filter(
        Boolean
      );
    },
    typeOptions() {
      return [...new Set(this.services.map((s) => s.type))].filter(Boolean);
    },
    filteredServices() {
      return this.services.filter((svc) => {
        if (this.selectedName && svc.name !== this.selectedName) return false;
        if (this.selectedNamespace && svc.namespace !== this.selectedNamespace)
          return false;
        if (this.selectedType && svc.type !== this.selectedType) return false;
        return true;
      });
    },
    pagedServices() {
      const start = (this.currentPage - 1) * this.pageSize;
      return this.filteredServices.slice(start, start + this.pageSize);
    },
  },
  methods: {
    handlePageChange(page) {
      this.currentPage = page;
    },
    handlePageSizeChange(size) {
      this.pageSize = size;
      this.currentPage = 1;
    },
  },
};
</script>

<style scoped>
.service-table-container {
  padding: 16px;
}
.table-title {
  margin-bottom: 16px;
}
.toolbar {
  margin-bottom: 12px;
}
</style>
