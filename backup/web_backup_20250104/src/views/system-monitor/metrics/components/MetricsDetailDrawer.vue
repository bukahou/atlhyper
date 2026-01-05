<template>
  <el-drawer
    :visible="visible"
    :size="width"
    :with-header="false"
    custom-class="metrics-detail-drawer"
    append-to-body
    :destroy-on-close="true"
    :close-on-click-modal="true"
    :before-close="handleBeforeClose"
    @update:visible="$emit('update:visible', $event)"
    @close="handleClose"
  >
    <!-- 顶部摘要 -->
    <div class="summary-bar">
      <div class="left">
        <span class="node-name">{{ latest.node || nodeId || "-" }}</span>
        <el-tag
          size="mini"
          type="success"
        >CPU {{ fmtPct(latest.cpuPercent) }}</el-tag>
        <el-tag
          size="mini"
          type="info"
        >Mem {{ fmtPct(latest.memPercent) }}</el-tag>
        <el-tag
          size="mini"
          type="warning"
        >Temp {{ fmtTemp(latest.cpuTempC) }}</el-tag>
        <el-tag
          size="mini"
          type="info"
        >Disk {{ fmtPct(latest.diskUsedPercent) }}</el-tag>
        <el-tag
          size="mini"
          type="info"
        >Tx {{ fmtKBps(latest.eth0TxKBps) }}</el-tag>
        <el-tag
          size="mini"
          type="info"
        >Rx {{ fmtKBps(latest.eth0RxKBps) }}</el-tag>
        <span class="age">更新于 {{ latest.timestamp || "—" }}</span>
      </div>
    </div>

    <!-- 主体 -->
    <div class="main">
      <!-- 左侧目录 -->
      <div class="sidenav">
        <el-menu
          :default-active="activeSection"
          class="menu"
          @select="scrollTo"
        >
          <el-menu-item index="overview">概览</el-menu-item>
          <el-menu-item index="timeline">趋势（CPU/内存/温度）</el-menu-item>
          <el-menu-item index="processes">进程 Top</el-menu-item>
          <el-menu-item index="raw">原始（JSON）</el-menu-item>
        </el-menu>
      </div>

      <!-- 右侧内容 -->
      <div ref="scrollEl" class="content" @scroll="onScroll">
        <!-- 概览 -->
        <section ref="overview" data-id="overview" class="section">
          <h3 class="section-title">概览</h3>
          <div class="kv">
            <div>
              <span>节点</span><b>{{ data.node || nodeId || "-" }}</b>
            </div>
            <div>
              <span>时间范围</span><b>{{ timeRangeStr }}</b>
            </div>
            <div>
              <span>最高温度</span><b>{{ maxTempStr }}</b>
            </div>
            <div>
              <span>CPU 平均</span><b>{{ avgCpuStr }}</b>
            </div>
            <div>
              <span>内存平均</span><b>{{ avgMemStr }}</b>
            </div>
            <div>
              <span>磁盘占用</span><b>{{ lastDiskStr }}</b>
            </div>
            <div>
              <span>Top 进程</span><b class="mono">{{ latest.topCPUProcess || "—" }}</b>
            </div>
          </div>
        </section>

        <!-- 趋势图：CPU / 内存 / 温度（一个图） -->
        <section ref="timeline" data-id="timeline" class="section">
          <h3 class="section-title">趋势（CPU / 内存 / 温度）</h3>
          <div
            v-if="!echartsMissing"
            ref="compositeChart"
            class="chart"
            style="height: 300px"
          />
          <div v-else class="muted">
            未检测到 ECharts，请执行 <code>npm i echarts</code> 后重试。
          </div>
        </section>

        <!-- 进程 Top -->
        <section ref="processes" data-id="processes" class="section">
          <h3 class="section-title">进程 Top</h3>
          <el-table
            :data="procRows"
            border
            size="mini"
            style="width: 100%"
            :header-cell-style="{
              background: '#f5f7fa',
              color: '#333',
              fontWeight: 600,
            }"
            empty-text="无数据"
          >
            <el-table-column prop="pid" label="PID" width="90" />
            <el-table-column prop="user" label="User" width="120" />
            <el-table-column
              prop="command"
              label="Command"
              min-width="160"
              show-overflow-tooltip
            />
            <el-table-column prop="cpuUsage" label="CPU" width="100" />
            <el-table-column prop="memoryUsage" label="Memory" width="120" />
          </el-table>
        </section>

        <!-- 原始（JSON） -->
        <section ref="raw" data-id="raw" class="section">
          <h3 class="section-title">原始（JSON）</h3>
          <pre class="json-viewer">{{ prettyJSON }}</pre>
        </section>
      </div>
    </div>
  </el-drawer>
</template>

<script>
// 如果还没装：npm i echarts
import { getMetricsdetail } from '@/api/metrics'

export default {
  name: 'MetricsDetailDrawer',
  props: {
    visible: { type: Boolean, default: false },
    clusterId: { type: [String, Number], required: true },
    nodeId: { type: String, required: true }, // 传 row.node
    width: { type: String, default: '60%' }
  },
  data() {
    return {
      activeSection: 'overview',
      data: {}, // 后端返回 data
      latest: {}, // data.latest
      series: {}, // data.series
      processes: [], // data.processes
      timeRange: {}, // data.timeRange
      // echarts
      echarts: null,
      chart: null,
      echartsMissing: false
    }
  },
  computed: {
    prettyJSON() {
      try {
        return JSON.stringify(this.data || {}, null, 2)
      } catch (e) {
        return '{}'
      }
    },
    timeRangeStr() {
      const s = (this.timeRange && this.timeRange.since) || ''
      const u = (this.timeRange && this.timeRange.until) || ''
      return s && u ? `${s} ~ ${u}` : '—'
    },
    procRows() {
      const arr = Array.isArray(this.processes) ? this.processes : []
      return arr.map((p) => ({
        pid: p.pid,
        user: p.user,
        command: p.command,
        cpuUsage:
          p.cpuUsage ||
          (this.isNum(p.cpuPercent)
            ? (p.cpuPercent * 100).toFixed(2) + '%'
            : '-'),
        memoryUsage:
          p.memoryUsage ||
          (this.isNum(p.memoryMB) ? p.memoryMB.toFixed(2) + ' MB' : '-')
      }))
    },
    // 概览里的汇总字符串
    maxTempStr() {
      const t = this.getMax(this.series && this.series.tempC)
      return this.isNum(t) ? `${t.toFixed(1)} ℃` : '—'
    },
    avgCpuStr() {
      const a = this.getAvg(this.series && this.series.cpuPct)
      return this.isNum(a) ? `${a.toFixed(2)}%` : '—'
    },
    avgMemStr() {
      const a = this.getAvg(this.series && this.series.memPct)
      return this.isNum(a) ? `${a.toFixed(2)}%` : '—'
    },
    lastDiskStr() {
      const arr = (this.series && this.series.diskPct) || []
      const last = arr.length ? arr[arr.length - 1] : NaN
      return this.isNum(last) ? `${Number(last).toFixed(1)}%` : '—'
    }
  },
  watch: {
    visible: {
      immediate: true,
      handler(v) {
        if (v) {
          // 打开时拉取
          this.fetchDetail()
          this.$nextTick(() => this.initChart())
        } else {
          this.disposeChart()
        }
      }
    },
    nodeId(val, oldVal) {
      if (this.visible && val && val !== oldVal) {
        this.fetchDetail()
      }
    }
  },
  mounted() {
    window.addEventListener('resize', this.resizeChart)
  },
  beforeDestroy() {
    window.removeEventListener('resize', this.resizeChart)
    this.disposeChart()
  },
  methods: {
    async fetchDetail() {
      if (!this.clusterId || !this.nodeId) return
      try {
        const res = await getMetricsdetail(this.clusterId, this.nodeId)
        if (res.code !== 20000) {
          this.$message.error(res.message || '获取节点指标详情失败')
          return
        }
        const d = res.data || {}
        this.data = d
        this.latest = d.latest || {}
        this.series = d.series || {}
        this.processes = Array.isArray(d.processes) ? d.processes : []
        this.timeRange = d.timeRange || {}
        // 重绘
        this.$nextTick(() => this.renderChart())
      } catch (e) {
        this.$message.error('请求失败：' + (e.message || e))
      }
    },

    // --------- ECharts ----------
    async ensureEcharts() {
      if (this.echarts) return this.echarts
      // 优先用全局（如果你项目里已经挂了 this.$echarts 也可）
      let lib = this.$echarts || window.echarts
      if (!lib) {
        try {
          lib = (await import('echarts')).default || (await import('echarts'))
        } catch (e) {
          this.echartsMissing = true
          return null
        }
      }
      this.echartsMissing = false
      this.echarts = lib
      return lib
    },
    async initChart() {
      const lib = await this.ensureEcharts()
      if (!lib) return
      const el = this.$refs.compositeChart
      if (!el) return
      this.disposeChart()
      this.chart = lib.init(el)
      this.renderChart()
    },
    renderChart() {
      if (!this.chart || !this.series) return
      const at = (this.series && this.series.at) || []
      const cpu = (this.series && this.series.cpuPct) || []
      const mem = (this.series && this.series.memPct) || []
      const temp = (this.series && this.series.tempC) || []
      const x = at.map((t) => this.toHHMM(t))

      const option = {
        tooltip: { trigger: 'axis' },
        legend: { data: ['CPU%', 'Mem%', 'Temp°C'] },
        grid: { left: 40, right: 20, top: 30, bottom: 35 },
        xAxis: { type: 'category', data: x, boundaryGap: false },
        yAxis: [
          {
            type: 'value',
            name: '%',
            position: 'left',
            axisLabel: { formatter: '{value}' }
          },
          {
            type: 'value',
            name: '°C',
            position: 'right',
            axisLabel: { formatter: '{value}' }
          }
        ],
        series: [
          { name: 'CPU%', type: 'line', smooth: true, data: cpu },
          { name: 'Mem%', type: 'line', smooth: true, data: mem },
          {
            name: 'Temp°C',
            type: 'line',
            smooth: true,
            yAxisIndex: 1,
            data: temp
          }
        ]
      }
      this.chart.setOption(option, true)
      this.resizeChart()
    },
    resizeChart() {
      if (this.chart) this.chart.resize()
    },
    disposeChart() {
      if (this.chart) {
        this.chart.dispose()
        this.chart = null
      }
    },

    // --------- 工具 & 显示 ---------
    fmtPct(v) {
      const n = Number(v)
      return Number.isFinite(n) ? `${n.toFixed(1)}%` : '—'
    },
    fmtTemp(v) {
      const n = Number(v)
      return Number.isFinite(n) ? `${n.toFixed(1)}℃` : '—'
    },
    fmtKBps(v) {
      const n = Number(v)
      return Number.isFinite(n) ? `${n.toFixed(0)} KB/s` : '—'
    },
    isNum(v) {
      return Number.isFinite(Number(v))
    },
    getAvg(arr) {
      if (!Array.isArray(arr) || !arr.length) return NaN
      const sum = arr.reduce((a, b) => a + Number(b || 0), 0)
      return sum / arr.length
    },
    getMax(arr) {
      if (!Array.isArray(arr) || !arr.length) return NaN
      return Math.max.apply(
        null,
        arr.map((x) => Number(x || 0))
      )
    },
    toHHMM(iso) {
      const t = Date.parse(iso)
      if (!Number.isFinite(t)) return iso || ''
      const d = new Date(t)
      const p = (n) => String(n).padStart(2, '0')
      return `${p(d.getHours())}:${p(d.getMinutes())}`
    },

    // 目录滚动 & 抽屉关闭
    scrollTo(id) {
      const el = this.$refs[id]
      if (!el || !this.$refs.scrollEl) return
      const top = el.offsetTop - 8
      this.$refs.scrollEl.scrollTo({ top, behavior: 'smooth' })
      this.activeSection = id
    },
    onScroll() {
      const container = this.$refs.scrollEl
      if (!container) return
      const ids = ['overview', 'timeline', 'processes', 'raw']
      let current = ids[0]
      for (let i = 0; i < ids.length; i++) {
        const id = ids[i]
        const el = this.$refs[id]
        if (el && el.offsetTop - container.scrollTop <= 40) current = id
      }
      this.activeSection = current
    },
    handleBeforeClose(done) {
      this.$emit('update:visible', false)
      if (typeof done === 'function') done()
    },
    handleClose() {
      this.$emit('update:visible', false)
    }
  }
}
</script>

<style scoped>
.metrics-detail-drawer {
  overflow: hidden;
}
.summary-bar {
  position: sticky;
  top: 0;
  z-index: 2;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 16px;
  background: #fff;
  border-bottom: 1px solid #eee;
}
.summary-bar .left {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
}
.summary-bar .node-name {
  font-weight: 600;
  font-size: 16px;
}
.summary-bar .age {
  color: #666;
  margin-left: 6px;
}

.main {
  display: flex;
  height: calc(100vh - 60px);
}
.sidenav {
  width: 240px;
  border-right: 1px solid #f0f0f0;
  padding: 8px 0;
  background: #fafafa;
}
.sidenav .menu {
  border-right: none;
}
.content {
  flex: 1;
  overflow: auto;
  padding: 12px 16px;
}

.section {
  margin-bottom: 20px;
}
.section-title {
  font-weight: 600;
  margin: 4px 0 10px;
}
.kv > div {
  display: flex;
  justify-content: space-between;
  padding: 6px 0;
  border-bottom: 1px dashed #f0f0f0;
}
.kv > div:last-child {
  border-bottom: none;
}
.kv span {
  color: #666;
  margin-right: 12px;
}
.muted {
  color: #999;
}
.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas,
    "Liberation Mono", monospace;
}
.chart {
  width: 100%;
}
.json-viewer {
  padding: 12px;
  background: #0e1116;
  color: #d5e5ff;
  border-radius: 6px;
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 12px;
}
</style>
