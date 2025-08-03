<template>
  <div class="ingress-table-container">
    <div class="table-title">
      <h2>Ingress 一览表</h2>
      <hr>
    </div>

    <!-- 分页控制 -->
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
      empty-text="暂无 Ingress 数据"
    >
      <el-table-column prop="name" label="名称" min-width="180" />
      <el-table-column prop="namespace" label="命名空间" width="140" />
      <el-table-column prop="host" label="域名 Host" min-width="180" />
      <el-table-column prop="path" label="路由路径" min-width="180" />
      <el-table-column prop="serviceName" label="服务名" width="160" />
      <el-table-column prop="servicePort" label="服务端口" width="120" />
      <el-table-column prop="tls" label="使用 TLS" width="100">
        <template slot-scope="{ row }">
          <span>{{ row.tls ? row.tls : "-" }}</span>
        </template>
      </el-table-column>
      <el-table-column prop="creationTime" label="创建时间" width="200" />
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
  name: 'IngressTable',
  props: {
    ingresses: {
      type: Array,
      required: true
    }
  },
  data() {
    return {
      pageSize: 10,
      currentPage: 1
    }
  },
  computed: {
    pagedIngresses() {
      const start = (this.currentPage - 1) * this.pageSize
      return this.ingresses.slice(start, start + this.pageSize)
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
.ingress-table-container {
  padding: 16px;
}
.table-title {
  margin-bottom: 16px;
}
.toolbar {
  margin-bottom: 12px;
}
</style>
