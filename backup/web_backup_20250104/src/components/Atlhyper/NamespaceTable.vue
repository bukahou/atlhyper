<template>
  <div class="namespace-table-container">
    <div class="table-title">
      <h2>Namespace List</h2>
      <hr>
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
      :data="pagedNamespaces"
      border
      style="width: 100%"
      :header-cell-style="{
        background: '#f5f7fa',
        color: '#333',
        fontWeight: 600,
      }"
      empty-text="No Namespace data available"
    >
      <el-table-column prop="name" label="Name" min-width="160" />
      <el-table-column prop="status" label="Status" width="100" />
      <el-table-column prop="podCount" label="Pod Count" width="100" />
      <el-table-column prop="labelCount" label="Label Count" width="100" />
      <el-table-column
        prop="annotationCount"
        label="Annotation Count"
        width="120"
      />
      <el-table-column prop="creationTime" label="Creation Time" width="180" />

      <!-- 操作列：仅派发事件，不做路由跳转 -->
      <el-table-column label="Actions" fixed="right" width="200">
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
            <el-button
              size="mini"
              plain
              type="success"
              icon="el-icon-collection"
              :style="{ padding: '4px 12px', fontSize: '12px' }"
              @click.stop="$emit('configmap', row)"
            >
              ConfigMap
            </el-button>
          </div>
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
      :total="namespaces.length"
      @current-change="handlePageChange"
    />
  </div>
</template>

<script>
export default {
  name: 'NamespaceTable',
  props: {
    namespaces: { type: Array, required: true }
  },
  data() {
    return {
      pageSize: 10,
      currentPage: 1
    }
  },
  computed: {
    pagedNamespaces() {
      const start = (this.currentPage - 1) * this.pageSize
      return this.namespaces.slice(start, start + this.pageSize)
    }
  },
  methods: {
    handlePageChange(page) {
      this.currentPage = page
    },
    handlePageSizeChange(size) {
      this.pageSize = size
      this.currentPage = 1
    }
  }
}
</script>

<style scoped>
.namespace-table-container {
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
