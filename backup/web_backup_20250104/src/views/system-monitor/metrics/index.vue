<template>
  <div class="page-container">
    <!-- 5s 轮询，仅页面可见时运行 -->
    <AutoPoll
      :interval="5000"
      :visible-only="true"
      :immediate="true"
      :task="loadLatest"
    />

    <!-- 顶部卡片 -->
    <div class="card-row">
      <el-tooltip
        v-for="(item, idx) in statCards"
        :key="idx"
        effect="dark"
        :content="item.tooltip"
        placement="top"
      >
        <div>
          <CardStat
            :icon-bg="item.iconBg"
            :number="item.value"
            number-color="color1"
            :title="item.title"
          >
            <template #icon><i :class="item.iconClass" /></template>
          </CardStat>
        </div>
      </el-tooltip>
    </div>

    <!-- 指标表格 -->
    <MetricsTable :rows="tableRows" @view="handleViewRow" />

    <!-- ▶️ 指标详情抽屉 -->
    <MetricsDetailDrawer
      v-if="drawerVisible"
      :visible.sync="drawerVisible"
      :cluster-id="currentId"
      :node-id="selectedNodeId"
      width="60%"
    />
  </div>
</template>

<script>
import AutoPoll from '@/components/Atlhyper/AutoPoll.vue'
import CardStat from '@/components/Atlhyper/CardStat.vue'
import MetricsTable from './components/MetricsTable.vue'
import MetricsDetailDrawer from './components/MetricsDetailDrawer.vue'
import { getMetricsOverview } from '@/api/metrics'
import { mapState } from 'vuex'

export default {
  name: 'MetricsPage',
  components: { AutoPoll, CardStat, MetricsTable, MetricsDetailDrawer },
  data() {
    return {
      statCards: [
        {
          key: 'cpuAvg',
          title: 'CPU Avg',
          value: '-',
          tooltip: 'CPU 平均使用率',
          iconClass: 'fas fa-microchip',
          iconBg: 'bg2'
        },
        {
          key: 'memAvg',
          title: 'Mem Avg',
          value: '-',
          tooltip: '内存平均使用率',
          iconClass: 'fas fa-memory',
          iconBg: 'bg3'
        },
        {
          key: 'tempMax',
          title: 'Temp Max',
          value: '-',
          tooltip: '最高 CPU 温度',
          iconClass: 'fas fa-thermometer-half',
          iconBg: 'bg4'
        },
        {
          key: 'diskMax',
          title: 'Disk Max',
          value: '-',
          tooltip: '磁盘使用峰值',
          iconClass: 'fas fa-hdd',
          iconBg: 'bg1'
        }
      ],
      tableRows: [],
      // 抽屉
      drawerVisible: false,
      selectedNodeId: ''
    }
  },
  computed: {
    ...mapState('cluster', ['currentId'])
  },
  watch: {
    currentId: {
      immediate: true,
      handler(id) {
        if (id) this.loadLatest()
      }
    }
  },
  methods: {
    async loadLatest() {
      if (!this.currentId) return
      try {
        const res = await getMetricsOverview(this.currentId)
        if (res.code !== 20000) {
          this.$message.error(res.message || '获取集群指标概览失败')
          return
        }
        const { cards = {}, rows = [] } = res.data || {}

        // 顶部卡片
        this.setCard(
          'cpuAvg',
          Number(cards.avgCPUPercent),
          (v) => `${v.toFixed(2)}%`,
          'CPU 平均使用率'
        )
        this.setCard(
          'memAvg',
          Number(cards.avgMemPercent),
          (v) => `${v.toFixed(2)}%`,
          '内存平均使用率'
        )
        this.setCard(
          'tempMax',
          Number(cards.peakTempC),
          (v) => `${v.toFixed(2)}℃`,
          `最高温度节点：${cards.peakTempNode || '-'}`
        )
        this.setCard(
          'diskMax',
          Number(cards.peakDiskPercent),
          (v) => `${v.toFixed(2)}%`,
          `峰值磁盘节点：${cards.peakDiskNode || '-'}`
        )

        // 表格数据
        this.tableRows = rows.map((r) => ({
          node: r.node || '-',
          cpuPercent: this.toNum(r.cpuPercent),
          memoryPercent: this.toNum(r.memPercent),
          cpuTemp: this.toNum(r.cpuTempC),
          diskPercent: this.toNum(r.diskUsedPercent),
          eth0Tx: this.kbpsToString(r.eth0TxKBps),
          eth0Rx: this.kbpsToString(r.eth0RxKBps),
          topCpuProcess: r.topCPUProcess || '-',
          timestamp: r.timestamp || '-'
        }))
      } catch (e) {
        this.$message.error('请求失败：' + (e.message || e))
      }
    },

    setCard(key, raw, fmt, tooltip) {
      const idx = this.statCards.findIndex((c) => c.key === key)
      if (idx < 0) return
      const n = this.toNum(raw)
      this.$set(this.statCards, idx, {
        ...this.statCards[idx],
        value: Number.isFinite(n) ? fmt(n) : '-',
        tooltip
      })
    },

    toNum(v) {
      const n = Number(v)
      return Number.isFinite(n) ? n : NaN
    },
    kbpsToString(v) {
      const n = Number(v)
      return Number.isFinite(n) ? `${n.toFixed(0)} KB/s` : '-'
      // 如需自动换单位，可自行扩展
    },

    // 打开详情抽屉
    handleViewRow(row) {
      this.selectedNodeId = row.node // API 的 NodeID 用节点名
      this.drawerVisible = true
    }
  }
}
</script>

<style scoped>
.page-container {
  padding: 35px 32px;
}
.card-row {
  display: flex;
  flex-wrap: wrap;
  gap: 90px;
  margin-bottom: 24px;
}

/* 保持标题支持换行 */
:deep(.card-stat .right-text .title) {
  white-space: pre-line !important;
  overflow: visible !important;
  text-overflow: unset !important;
  line-height: 1.25;
  opacity: 0.9;
}

/* ✅ 整体左移数字+标题 10px（原本 80px） */
:deep(.card-stat .right-text) {
  margin-left: 70px !important;
}
</style>
