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
      <el-table-column prop="action" label="操作行为" min-width="220" />
      <el-table-column prop="method" label="方法" width="100" />
      <el-table-column prop="status" label="状态码" width="100" />
      <el-table-column prop="ip" label="IP" min-width="140" />

      <el-table-column prop="success" label="结果" width="100">
        <template slot-scope="{ row }">
          <el-tag :type="row.success ? 'success' : 'danger'" size="mini">
            {{ row.success ? "成功" : "失败" }}
          </el-tag>
        </template>
      </el-table-column>

      <el-table-column prop="timestamp" label="时间" width="200">
        <template slot-scope="{ row }">
          {{ formatTs(row.timestamp) }}
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
      :total="normalizedLogs.length"
      @current-change="handlePageChange"
    />
  </div>
</template>

<script>
export default {
  name: 'UserAuditTable',
  props: {
    // 后端返回的列表。支持两种字段风格：
    // 1) 大写: { ID, UserID, Username, Role, Action, Success, IP, Method, Status, Timestamp }
    // 2) 小写: { id, user_id, username, role, action, success, ip, method, status, timestamp }
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
    // 统一/兼容化字段
    normalizedLogs() {
      const roleNameMap = { 1: 'Viewer', 2: 'Operator', 3: 'Admin' }
      const normIP = (ip) => (ip === '::1' ? '127.0.0.1' : ip || '')

      return (this.logs || []).map((item) => {
        // 兼容两种命名
        const id = item.id ?? item.ID
        const userId = item.user_id ?? item.UserID
        const username = item.username ?? item.Username
        const role = item.role ?? item.Role
        const action = item.action ?? item.Action
        const success =
          (item.success ?? item.Success) === true ||
          (item.success ?? item.Success) === 1
        const ip = normIP(item.ip ?? item.IP)
        const method = item.method ?? item.Method
        const status = item.status ?? item.Status
        const timestamp = item.timestamp ?? item.Timestamp

        return {
          id,
          userId,
          username,
          role,
          roleName: roleNameMap[Number(role)] || String(role || ''),
          action,
          success,
          ip,
          method,
          status: Number(status ?? 0) || 0,
          timestamp
        }
      })
    },
    pagedLogs() {
      const start = (this.currentPage - 1) * this.pageSize
      return this.normalizedLogs.slice(start, start + this.pageSize)
    }
  },
  methods: {
    handlePageChange(page) {
      this.currentPage = page
    },
    // 本地时间格式化（保留到秒）
    formatTs(ts) {
      if (!ts) return ''
      const d = new Date(ts)
      if (isNaN(d.getTime())) return ts // 兜底：无法解析就原样显示
      const pad = (n) => (n < 10 ? '0' + n : '' + n)
      const Y = d.getFullYear()
      const M = pad(d.getMonth() + 1)
      const D = pad(d.getDate())
      const h = pad(d.getHours())
      const m = pad(d.getMinutes())
      const s = pad(d.getSeconds())
      return `${Y}-${M}-${D} ${h}:${m}:${s}`
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
.pagination {
  margin-top: 12px;
  text-align: right;
}
</style>
