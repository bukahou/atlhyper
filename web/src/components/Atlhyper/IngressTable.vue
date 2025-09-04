<template>
  <div class="ingress-table-container">
    <div class="table-title">
      <h2>Ingress List</h2>
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
      :data="pagedIngresses"
      border
      style="width: 100%"
      :header-cell-style="{
        background: '#f5f7fa',
        color: '#333',
        fontWeight: 600,
      }"
      empty-text="No Ingress data available"
    >
      <el-table-column prop="name" label="Name" min-width="180" />
      <el-table-column prop="namespace" label="Namespace" width="140" />
      <el-table-column prop="host" label="Host" min-width="180" />
      <el-table-column prop="path" label="Path" min-width="180" />
      <el-table-column prop="serviceName" label="Service Name" width="160" />
      <el-table-column prop="servicePort" label="Service Port" width="120" />
      <el-table-column prop="tls" label="TLS" width="120">
        <template slot-scope="{ row }">
          <span>{{ row.tls ? row.tls : "-" }}</span>
        </template>
      </el-table-column>
      <el-table-column prop="creationTime" label="Creation Time" width="200" />

      <!-- 操作列：只派发事件，不做路由跳转 -->
      <el-table-column label="Actions" fixed="right" width="120">
        <template slot-scope="{ row }">
          <div class="action-buttons">
            <el-button
              size="mini"
              type="primary"
              plain
              icon="el-icon-view"
              :style="{ padding: '4px 12px', fontSize: '12px' }"
              @click.stop="$emit('view', row)"
            >
              View
            </el-button>
          </div>
        </template>
      </el-table-column>
    </el-table>

    <!-- 分页器 -->
    <el-pagination
      class="pagination"
      background
      small
      layout="prev, pager, next"
      :page-size="pageSize"
      :current-page="currentPage"
      :total="ingresses.length"
      @current-change="handlePageChange"
    />
  </div>
</template>

<script>
export default {
  name: "IngressTable",
  props: {
    ingresses: { type: Array, required: true },
  },
  data() {
    return {
      pageSize: 10,
      currentPage: 1,
    };
  },
  computed: {
    pagedIngresses() {
      const start = (this.currentPage - 1) * this.pageSize;
      return this.ingresses.slice(start, start + this.pageSize);
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
.ingress-table-container {
  padding: 16px;
}
.table-title {
  margin-bottom: 16px;
}
.toolbar {
  margin-bottom: 12px;
}
.action-buttons {
  display: flex;
  justify-content: center;
  gap: 6px;
}
</style>
