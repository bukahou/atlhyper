<template>
  <div class="audit-table-container">
    <div class="table-title">
      <h2>用户审计日志</h2>
      <hr>
    </div>

    <el-table
      :data="pagedLogs"
      border
      style="width: 100%"
      :header-cell-style="{
        background: '#f5f7fa',
        color: '#333',
        fontWeight: 600,
      }"
      empty-text="暂无审计记录"
    >
      <el-table-column prop="username" label="用户名" min-width="120" />
      <el-table-column prop="roleName" label="角色" width="120" />
      <el-table-column prop="action" label="操作行为" min-width="180" />
      <el-table-column prop="success" label="结果" width="100">
        <template slot-scope="{ row }">
          <el-tag :type="row.success ? 'success' : 'danger'" size="mini">
            {{ row.success ? "成功" : "失败" }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="timestamp" label="时间" width="200" />
    </el-table>

    <el-pagination
      class="pagination"
      background
      small
      layout="prev, pager, next"
      :page-size="pageSize"
      :current-page="currentPage"
      :total="logs.length"
      @current-change="handlePageChange"
    />
  </div>
</template>

<script>
export default {
  name: 'UserAuditTable',
  props: {
    logs: {
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
    pagedLogs() {
      const start = (this.currentPage - 1) * this.pageSize
      return this.logs.slice(start, start + this.pageSize)
    }
  },
  methods: {
    handlePageChange(page) {
      this.currentPage = page
    }
  }
}
</script>

<style scoped>
.audit-table-container {
  padding: 16px;
}
.table-title {
  margin-bottom: 16px;
}
</style>
