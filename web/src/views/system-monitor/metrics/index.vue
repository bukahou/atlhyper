<template>
  <div class="page-container">
    <!-- 🔁 5s 自动轮询 -->
    <AutoPoll
      :interval="5000"
      :visible-only="true"
      :immediate="true"
      :task="loadLatest"
    />

    <!-- ✅ 顶部状态卡片区域 -->
    <div class="card-row">
      <!-- 可选：tooltip，让内容再长也能完整查看 -->
      <el-tooltip
        v-for="(item, index) in statCards"
        :key="index"
        effect="dark"
        :content="`${item.plainTitle}（${item.node || '-'}）`"
        placement="top"
      >
        <!-- el-tooltip 必须只有一个子节点 -->
        <div>
          <CardStat
            :icon-bg="item.iconBg"
            :number="item.value"
            :number-color="'color1'"
            :title="item.title"
          >
            <template #icon>
              <i :class="item.iconClass" />
            </template>
          </CardStat>
        </div>
      </el-tooltip>
    </div>

    <!-- ✅ 指标表格组件 -->
    <MetricsTable :rows="tableRows" @view="handleViewRow" />
  </div>
</template>

<script>
import AutoPoll from '@/components/Atlhyper/AutoPoll.vue'
import CardStat from '@/components/Atlhyper/CardStat.vue'
import MetricsTable from './components/MetricsTable.vue'
import { getLatestMetrics } from '@/api/metrics'

export default {
  name: 'MetricsPage',
  components: { AutoPoll, CardStat, MetricsTable },
  data() {
    return {
      statCards: [
        {
          plainTitle: 'CPU', // 纯文本标题（用在 tooltip）
          title: 'CPU', // 展示用，下面会变成两行
          value: '-',
          node: '-',
          iconClass: 'fas fa-microchip',
          iconBg: 'bg2'
        },
        {
          plainTitle: 'Mem',
          title: 'Mem',
          value: '-',
          node: '-',
          iconClass: 'fas fa-memory',
          iconBg: 'bg3'
        },
        {
          plainTitle: 'Temp',
          title: 'Temp',
          value: '-',
          node: '-',
          iconClass: 'fas fa-thermometer-half',
          iconBg: 'bg4'
        },
        {
          plainTitle: 'Disk',
          title: 'Disk',
          value: '-',
          node: '-',
          iconClass: 'fas fa-hdd',
          iconBg: 'bg1'
        }
      ],
      tableRows: []
    }
  },
  methods: {
    async loadLatest() {
      try {
        const res = await getLatestMetrics()
        const raw = res?.data ?? res
        const { stats, rows } = this.adaptForView(raw)

        // 数值
        this.statCards[0].value = stats.maxCpu.value.toFixed(2) + '%'
        this.statCards[1].value = stats.maxMem.value.toFixed(2) + '%'
        this.statCards[2].value = stats.maxTemp.value.toFixed(2) + '°C'
        this.statCards[3].value = stats.maxDisk.value.toFixed(2) + '%'

        // 节点 + 两行标题（不改 CardStat，也能换行）
        this.statCards[0].node = stats.maxCpu.node
        this.statCards[1].node = stats.maxMem.node
        this.statCards[2].node = stats.maxTemp.node
        this.statCards[3].node = stats.maxDisk.node

        // 让 title 变为两行：第一行标题，第二行（节点）
        this.statCards.forEach((c) => {
          c.title = `${c.plainTitle}\n（${c.node || '-'}）`
        })

        // 表格
        this.tableRows = rows
      } catch (e) {
        console.warn('[Metrics] loadLatest failed:', e)
      }
    },

    adaptForView(payload) {
      const data = payload?.data ?? payload ?? {}
      const nodes = Object.values(data)

      const toNumberPercent = (v, fallback = 0) => {
        if (v == null) return fallback
        if (typeof v === 'string' && v.endsWith('%')) return parseFloat(v)
        const n = Number(v)
        if (!Number.isFinite(n)) return fallback
        return n <= 1 && n >= 0 ? n * 100 : n
      }

      let maxCpu = { value: 0, node: '-' }
      let maxMem = { value: 0, node: '-' }
      let maxTemp = { value: -Infinity, node: '-' }
      let maxDisk = { value: 0, node: '-' }

      const rows = nodes.map((n) => {
        const node = n?.nodeName || '-'
        const cpuPct = toNumberPercent(
          n?.cpu?.usagePercent,
          toNumberPercent(n?.cpu?.usage)
        )
        const memPct = toNumberPercent(
          n?.memory?.usagePercent,
          n?.memory?.usage
        )
        const cpuTemp = Number(n?.temperature?.cpuDegrees ?? NaN)
        const firstDisk =
          Array.isArray(n?.disk) && n.disk.length > 0 ? n.disk[0] : null
        const diskPct = toNumberPercent(
          firstDisk?.usagePercent,
          firstDisk?.usage
        )
        const eth0 =
          (Array.isArray(n?.network) ? n.network : []).find(
            (i) => i?.interface === 'eth0'
          ) || {}
        const tx = eth0?.txSpeed ?? '-'
        const rx = eth0?.rxSpeed ?? '-'
        const topProc =
          Array.isArray(n?.topCPUProcesses) && n.topCPUProcesses.length > 0
            ? n.topCPUProcesses[0]
            : null
        const topCmd = topProc?.command || '-'

        if (cpuPct > maxCpu.value) maxCpu = { value: cpuPct, node }
        if (memPct > maxMem.value) maxMem = { value: memPct, node }
        if (Number.isFinite(cpuTemp) && cpuTemp > maxTemp.value) { maxTemp = { value: cpuTemp, node } }
        if (diskPct > maxDisk.value) maxDisk = { value: diskPct, node }

        return {
          node,
          cpuPercent: cpuPct,
          memoryPercent: memPct,
          cpuTemp,
          diskPercent: diskPct,
          eth0Tx: tx,
          eth0Rx: rx,
          topCpuProcess: topCmd,
          timestamp: n?.timestamp || '-'
        }
      })

      if (!nodes.length) maxTemp.value = 0

      return { stats: { maxCpu, maxMem, maxTemp, maxDisk }, rows }
    },

    handleViewRow(row) {
      this.$message.info(`Node: ${row.node}\nTime: ${row.timestamp}`)
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
