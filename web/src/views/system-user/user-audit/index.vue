<template>
  <div class="audit-log-page">
    <!-- ✅ 顶部统计卡片 -->
    <el-row :gutter="20" class="card-row">
      <el-col :xs="24" :sm="12" :md="6">
        <CardStat
          icon-bg="bg1"
          :number="stats.totalOps"
          number-color="color1"
          title="总操作次数"
        >
          <template #icon><i class="el-icon-document" /></template>
        </CardStat>
      </el-col>
      <el-col :xs="24" :sm="12" :md="6">
        <CardStat
          icon-bg="bg2"
          :number="stats.successOps"
          number-color="color2"
          title="成功操作数"
        >
          <template #icon><i class="el-icon-check" /></template>
        </CardStat>
      </el-col>
      <el-col :xs="24" :sm="12" :md="6">
        <CardStat
          icon-bg="bg3"
          :number="stats.failOps"
          number-color="color3"
          title="失败操作数"
        >
          <template #icon><i class="el-icon-close" /></template>
        </CardStat>
      </el-col>
      <el-col :xs="24" :sm="12" :md="6">
        <CardStat
          icon-bg="bg4"
          :number="stats.activeUsers"
          number-color="color4"
          title="活跃用户数"
        >
          <template #icon><i class="el-icon-user-solid" /></template>
        </CardStat>
      </el-col>
    </el-row>

    <!-- ✅ 下方占位区域：可替换为审计表格 -->
    <el-card shadow="hover">
      <UserAuditTable :logs="auditLogs" />
    </el-card>
  </div>
</template>

<script>
import CardStat from '@/components/Atlhyper/CardStat.vue'
import UserAuditTable from '@/components/Atlhyper/UserAuditTable.vue'

export default {
  name: 'AuditLogView',
  components: {
    CardStat,
    UserAuditTable
  },
  data() {
    return {
      stats: {
        totalOps: 208,
        successOps: 192,
        failOps: 16,
        activeUsers: 9
      },
      auditLogs: [
        {
          username: 'admin',
          roleName: '超级管理员',
          action: '更新用户权限',
          success: true,
          timestamp: '2025-07-30 15:32:10'
        },
        {
          username: 'wuxiafeng',
          roleName: '普通用户',
          action: '尝试访问系统配置',
          success: false,
          timestamp: '2025-07-30 14:21:40'
        },
        {
          username: 'editor',
          roleName: '访客',
          action: '查看用户列表',
          success: true,
          timestamp: '2025-07-30 14:12:00'
        }
      ]
    }
  }
}
</script>

<style scoped>
.audit-log-page {
  padding: 20px;
}

.card-row {
  margin-bottom: 30px;
}

.log-table-section {
  padding: 10px;
}

.placeholder {
  padding: 40px 0;
  text-align: center;
  color: #999;
  font-size: 14px;
}
</style>
