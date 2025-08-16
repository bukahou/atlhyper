<template>
  <div class="analysis-page">
    <div class="analysis-grid">
      <!-- 顶部五张卡片：5 等分子网格，等宽等高 -->
      <div class="col col-12">
        <div class="top-cards">
          <!-- Health -->
          <div class="cell">
            <HealthCard v-if="d" :data="d.health_card" />
            <el-skeleton v-else :rows="4" animated />
          </div>

          <!-- Nodes -->
          <div class="cell">
            <StatCard
              v-if="d"
              title="Nodes"
              :value="`${d.nodes_card.ready_nodes} / ${d.nodes_card.total_nodes}`"
              :sub-text="`Ready: ${pct(d.nodes_card.node_ready_pct)}`"
              icon="el-icon-s-grid"
              :percent="d.nodes_card.node_ready_pct"
              accent="#6366F1"
            />
            <el-skeleton v-else :rows="2" animated />
          </div>

          <!-- CPU -->
          <div class="cell">
            <StatCard
              v-if="d"
              title="CPU Usage"
              :value="d.cpu_card.percent"
              unit="%"
              icon="el-icon-cpu"
              :percent="d.cpu_card.percent"
              accent="#F97316"
            />
            <el-skeleton v-else :rows="2" animated />
          </div>

          <!-- Memory -->
          <div class="cell">
            <StatCard
              v-if="d"
              title="Memory Usage"
              :value="d.mem_card.percent"
              unit="%"
              icon="el-icon-pie-chart"
              :percent="d.mem_card.percent"
              accent="#10B981"
            />
            <el-skeleton v-else :rows="2" animated />
          </div>

          <!-- Alerts -->
          <div class="cell">
            <StatCard
              v-if="d"
              title="Alerts"
              :value="d.alerts_total"
              :sub-text="'24h'"
              icon="el-icon-bell"
              accent="#EF4444"
            />
            <el-skeleton v-else :rows="2" animated />
          </div>
        </div>
      </div>

      <!-- Resource Usage Trends（左上，占 6 栅格） -->
      <div class="col col-6">
        <ResourceTrendsChart
          v-if="d"
          :cpu="d.cpu_series || []"
          :mem="d.mem_series || []"
          :temp="d.temp_series || []"
        />
        <el-skeleton v-else :rows="6" animated />
      </div>

      <!-- Alert Trends（右上，占 6 栅格） -->
      <div class="col col-6">
        <AlertTrendsChart
          v-if="d"
          :series="(d.alert_trends && d.alert_trends.series) || []"
        />
        <el-skeleton v-else :rows="6" animated />
      </div>

      <!-- Recent Alerts（左下，占 6 栅格） -->
      <div class="col col-6">
        <RecentAlertsTable
          v-if="d"
          :items="d.recent_alerts || []"
          :loading="loading"
          :show-ns-filter="true"
        />
        <el-skeleton v-else :rows="6" animated />
      </div>

      <!-- Node Resource Usage Top5（右下，占 6 栅格） -->
      <div class="col col-6">
        <NodeResourceUsage
          v-if="d"
          :items="d.node_usages || []"
          :page-size="5"
        />
        <el-skeleton v-else :rows="6" animated />
      </div>
    </div>
  </div>
</template>

<script>
import { getClusterOverview } from '@/api/analysis'
import HealthCard from './components/HealthCard.vue'
import StatCard from './components/StatCard.vue'
import ResourceTrendsChart from './components/ResourceTrendsChart.vue'
import AlertTrendsChart from './components/AlertTrendsChart.vue'
import RecentAlertsTable from './components/RecentAlertsTable.vue'
import NodeResourceUsage from './components/NodeResourceUsage.vue'

export default {
  name: 'AnalysisIndex',
  components: {
    HealthCard,
    StatCard,
    ResourceTrendsChart,
    AlertTrendsChart,
    RecentAlertsTable,
    NodeResourceUsage
  },
  data() {
    return {
      d: null,
      loading: false
    }
  },
  created() {
    this.fetchData()
  },
  methods: {
    async fetchData() {
      this.loading = true
      try {
        const res = await getClusterOverview()
        if (res.code === 20000) {
          this.d = res.data
        } else {
          this.$message.error(res.message || '获取集群概览失败')
        }
      } catch (e) {
        this.$message.error('请求失败：' + e)
      } finally {
        this.loading = false
      }
    },
    pct(v) {
      return `${(v ?? 0).toFixed(2)}%`
    }
  }
}
</script>

<style scoped>
/* 页面整体留白（顶部 & 左右 & 底部） */
.analysis-page {
  padding: 16px;
}
@media (min-width: 1280px) {
  .analysis-page {
    padding: 20px 24px;
  }
}
@media (min-width: 1600px) {
  .analysis-page {
    padding: 24px 28px;
  }
}

/* 主网格：分离行/列间距，让上下区块更舒展 */
.analysis-grid {
  display: grid;
  grid-template-columns: repeat(12, 1fr);
  column-gap: 12px; /* 左右间距 */
  row-gap: 18px; /* 上下间距（比列间距略大） */
}

.col {
  min-width: 0;
}
.col-6 {
  grid-column: span 6;
}
.col-12 {
  grid-column: span 12;
}

/* 顶部 5 等分子网格（等宽） */
.top-cards {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 12px; /* 同时作用于行/列 */
}
.top-cards .cell {
  min-width: 0;
}

/* 卡片等高：父网格与子网格都拉满 */
.col :deep(.card),
.top-cards .cell :deep(.card) {
  height: 100%;
}
</style>
