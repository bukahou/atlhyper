<template>
  <div class="audit-log-page">
    <!-- 顶部统计卡片 -->
    <el-row :gutter="20" class="card-row">
      <el-col :xs="24" :sm="12" :md="6">
        <CardStat
          icon-bg="bg1"
          :number="stats.total"
          number-color="color1"
          title="日志数量"
        >
          <template #icon><i class="el-icon-document" /></template>
        </CardStat>
      </el-col>

      <el-col :xs="24" :sm="12" :md="6">
        <CardStat
          icon-bg="bg4"
          :number="stats.fail"
          number-color="color4"
          title="失败数量"
        >
          <template #icon><i class="el-icon-close" /></template>
        </CardStat>
      </el-col>

      <el-col :xs="24" :sm="12" :md="6">
        <CardStat
          icon-bg="bg3"
          :number="stats.err4xx"
          number-color="color3"
          title="400系错误"
        >
          <template #icon><i class="el-icon-warning" /></template>
        </CardStat>
      </el-col>

      <el-col :xs="24" :sm="12" :md="6">
        <CardStat
          icon-bg="bg2"
          :number="stats.err5xx"
          number-color="color2"
          title="500系错误"
        >
          <template #icon><i class="el-icon-warning-outline" /></template>
        </CardStat>
      </el-col>
    </el-row>

    <!-- 刷新按钮 -->
    <div class="toolbar">
      <el-button
        type="primary"
        size="mini"
        :loading="loading"
        @click="fetchLogs"
      >
        刷新
      </el-button>
      <span
        v-if="lastUpdated"
        class="updated-at"
      >最后更新：{{ lastUpdated }}</span>
    </div>

    <!-- 明细表格 -->
    <el-card shadow="hover">
      <UserAuditTable :logs="auditLogs" />
    </el-card>
  </div>
</template>

<script>
import CardStat from '@/components/Atlhyper/CardStat.vue'
import UserAuditTable from '@/components/Atlhyper/UserAuditTable.vue'
import { listUserAuditLogs } from '@/api/user'

export default {
  name: 'AuditLogView',
  components: { CardStat, UserAuditTable },
  data() {
    return {
      loading: false,
      auditLogs: [],
      stats: {
        total: 0,
        fail: 0,
        err4xx: 0,
        err5xx: 0
      },
      lastUpdated: ''
    }
  },
  created() {
    this.fetchLogs()
  },
  methods: {
    isSuccess(v) {
      // 兼容 bool / 0/1 / "0"/"1"
      return v === true || v === 1 || v === '1'
    },
    fetchLogs() {
      this.loading = true
      listUserAuditLogs()
        .then((res) => {
          // 兼容两种后端返回：data 为数组 或 data.list
          const list = Array.isArray(res.data)
            ? res.data
            : res.data && Array.isArray(res.data.list)
              ? res.data.list
              : []

          this.auditLogs = list

          // 统计
          const total = list.length
          let fail = 0
          let err4xx = 0
          let err5xx = 0

          list.forEach((item) => {
            const success = this.isSuccess(item.success ?? item.Success)
            const statusRaw = item.status ?? item.Status
            const status = Number(statusRaw ?? 0) || 0

            if (!success) fail += 1
            if (status >= 400 && status <= 499) err4xx += 1
            if (status >= 500 && status <= 599) err5xx += 1
          })

          this.stats = { total, fail, err4xx, err5xx }
          this.lastUpdated = this.formatNow()
        })
        .catch((e) => {
          this.$message.error('获取用户审计日志失败')
          // 清空并重置统计
          this.auditLogs = []
          this.stats = { total: 0, fail: 0, err4xx: 0, err5xx: 0 }
        })
        .finally(() => {
          this.loading = false
        })
    },
    formatNow() {
      const d = new Date()
      const pad = (n) => (n < 10 ? '0' + n : '' + n)
      return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(
        d.getDate()
      )} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
    }
  }
}
</script>

<style scoped>
.audit-log-page {
  padding: 20px;
}
.card-row {
  margin-bottom: 16px;
}
.toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin: 8px 0 16px;
}
.updated-at {
  color: #909399;
  font-size: 12px;
}
</style>
