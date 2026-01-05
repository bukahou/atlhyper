<template>
  <div class="analysis-page">
    <div class="analysis-grid">
      <!-- 顶部五张卡片 -->
      <div class="col col-12">
        <div class="top-cards">
          <div class="cell">
            <HealthCard v-if="d" :data="d.health_card" />
            <el-skeleton v-else :rows="4" animated />
          </div>
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

      <!-- 左右图表 -->
      <div class="col col-6">
        <ResourceTrendsChart
          v-if="d"
          :cpu="d.cpu_series || []"
          :mem="d.mem_series || []"
          :temp="d.temp_series || []"
        />
        <el-skeleton v-else :rows="6" animated />
      </div>

      <div class="col col-6">
        <AlertTrendsChart
          v-if="d"
          :series="(d.alert_trends && d.alert_trends.series) || []"
        />
        <el-skeleton v-else :rows="6" animated />
      </div>

      <div class="col col-6">
        <RecentAlertsTable
          v-if="d"
          :items="d.recent_alerts || []"
          :loading="loading"
          :show-ns-filter="true"
        />
        <el-skeleton v-else :rows="6" animated />
      </div>

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
import { mapState } from 'vuex'
import { getClusterOverview } from '@/api/analysis'
import HealthCard from './components/HealthCard.vue'
import StatCard from './components/StatCard.vue'
import ResourceTrendsChart from './components/ResourceTrendsChart.vue'
import AlertTrendsChart from './components/AlertTrendsChart.vue'
import RecentAlertsTable from './components/RecentAlertsTable.vue'
import NodeResourceUsage from './components/NodeResourceUsage.vue'

const CURR_KEY = 'atlhyper_cluster_id'

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
      loading: false,
      currentClusterId: '' // 本页正在使用的集群ID
    }
  },
  computed: {
    // 登录时存的 clusterIds，用于兜底默认
    ...mapState('user', ['clusterIds'])
  },
  created() {
    // 1) 选择一个集群ID：localStorage 优先，其次 clusterIds[0]
    const saved = localStorage.getItem(CURR_KEY)
    if (saved && (!this.clusterIds.length || this.clusterIds.includes(saved))) {
      this.currentClusterId = saved
    } else if (this.clusterIds && this.clusterIds.length > 0) {
      this.currentClusterId = this.clusterIds[0]
      localStorage.setItem(CURR_KEY, this.currentClusterId)
    }
    // 2) 拉取数据
    this.fetchData()
  },
  mounted() {
    // 监听 Navbar 派发的集群变更事件
    window.addEventListener('cluster-changed', this.onClusterChanged)
  },
  beforeDestroy() {
    window.removeEventListener('cluster-changed', this.onClusterChanged)
  },
  methods: {
    async fetchData() {
      this.loading = true
      try {
        // ✅ 从 localStorage 或 Vuex 里取当前集群ID
        const clusterId = localStorage.getItem('atlhyper_cluster_id')
        if (!clusterId) {
          this.$message.error('当前没有选中的集群ID')
          return
        }

        const res = await getClusterOverview(clusterId) // ✅ 传参
        if (res.code === 20000) {
          this.d = this.transformOverview(res.data)
        } else {
          this.$message.error(res.message || '获取集群概览失败')
        }
      } catch (e) {
        this.$message.error('请求失败：' + e)
      } finally {
        this.loading = false
      }
    },
    onClusterChanged(e) {
      const id = e && e.detail
      if (id && id !== this.currentClusterId) {
        this.currentClusterId = id
        this.fetchData()
      }
    },
    pct(v) {
      return `${(v ?? 0).toFixed(2)}%`
    },
    // 把后端返回结构映射到页面组件使用的字段
    transformOverview(data) {
      const cards = data.cards || {}
      const trends = data.trends || {}
      const alerts = data.alerts || {}
      // const nodes = data.nodes || {};

      // 顶部卡片
      const health_card = {
        status: cards.clusterHealth?.status ?? 'Unknown',
        // 新返回体没有 reason，就留空，组件会显示 "—"
        reason: cards.clusterHealth?.reason ?? '',
        node_ready_pct: Number(cards.clusterHealth?.nodeReadyPercent ?? 0),
        pod_healthy_pct: Number(cards.clusterHealth?.podReadyPercent ?? 0)
      }
      const nodes_card = {
        total_nodes: Number(cards.nodeReady?.total ?? 0),
        ready_nodes: Number(cards.nodeReady?.ready ?? 0),
        node_ready_pct: Number(cards.nodeReady?.percent ?? 0)
      }
      const cpu_card = { percent: Number(cards.cpuUsage?.percent ?? 0) }
      const mem_card = { percent: Number(cards.memUsage?.percent ?? 0) }
      const alerts_total = Number(cards.events24h ?? 0)

      // 资源趋势
      const cpu_series = (trends.resourceUsage || []).map((p) => [
        new Date(p.at).getTime(),
        Number((p.cpuPeak ?? 0) * 100) // 0~1 → 百分比
      ])

      const mem_series = (trends.resourceUsage || []).map((p) => [
        new Date(p.at).getTime(),
        Number((p.memPeak ?? 0) * 100)
      ])

      const temp_series = (trends.resourceUsage || []).map((p) => [
        new Date(p.at).getTime(),
        Number(p.tempPeak ?? 0)
      ])

      // 告警趋势
      // AnalysisIndex.vue -> transformOverview(data)
      const trendList = alerts.trend || []
      const alert_trends = {
        series: trendList
          .map((p) => {
            const ts = new Date(p.at).getTime()
            return {
              ts, // ✅ 毫秒时间戳
              critical: Number(p.critical ?? 0),
              warning: Number(p.warning ?? 0),
              info: Number(p.info ?? 0)
            }
          })
          .filter((it) => Number.isFinite(it.ts)), // 过滤非法时间
        totals: {
          critical: Number(alerts.totals?.critical ?? 0),
          warning: Number(alerts.totals?.warning ?? 0),
          info: Number(alerts.totals?.info ?? 0)
        }
      }

      // 最近告警
      // AnalysisIndex.vue -> transformOverview(data) 里
      const recent_alerts = (data.alerts?.recent || []).map((x) => ({
        time: x.Timestamp, // ✅ 表格用的是 time
        severity: x.Severity, // ✅ severity
        kind: x.Kind, // ✅ kind
        namespace: x.Namespace, // ✅ namespace
        message: x.Message, // ✅ message
        reason: x.ReasonCode, // ✅ reason（用于回退显示）
        name: x.Name, // 备查
        node: x.Node // 备查
      }))

      // 节点用量
      // 助手函数（你前面已有类似的 n / pctClamp，也可以复用）
      const n = (v, d = 0) => (Number.isFinite(Number(v)) ? Number(v) : d)
      const pctClamp = (v) => Math.max(0, Math.min(100, n(v)))

      // ...
      const node_usages = (data.nodes?.usage || []).map((it) => ({
        node_name: it.node, // ✅ 组件需要 node_name
        // 如果后端暂时没给，就先兜底：ready=true、role=''（或 'worker'）
        ready: true,
        role: '', // 或 'worker' / 'control-plane'（若后端有就透传）

        // ✅ 组件需要百分比字段（0-100）
        cpu_percent: pctClamp(it.cpuUsage), // 后端已是百分比数值，钳一下更稳
        memory_percent: pctClamp(it.memUsage)
      }))

      return {
        cluster_id: data.clusterId,
        health_card,
        nodes_card,
        cpu_card,
        mem_card,
        alerts_total,
        cpu_series,
        mem_series,
        temp_series,
        alert_trends,
        recent_alerts,
        node_usages
      }
    }
  }
}
</script>

<style scoped>
/* 你的样式保持不变 */
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
.analysis-grid {
  display: grid;
  grid-template-columns: repeat(12, 1fr);
  column-gap: 12px;
  row-gap: 18px;
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
.top-cards {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 12px;
}
.top-cards .cell {
  min-width: 0;
}
.col :deep(.card),
.top-cards .cell :deep(.card) {
  height: 100%;
}
</style>
