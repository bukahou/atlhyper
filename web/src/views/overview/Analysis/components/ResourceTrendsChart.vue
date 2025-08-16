<template>
  <div class="card">
    <div class="card-title">Resource Usage Trends</div>
    <div ref="chart" :style="{ width: '100%', height: height }" />
  </div>
</template>

<script>
// 确保已安装 echarts: npm i echarts --save
import * as echarts from 'echarts'

export default {
  name: 'ResourceTrendsChart',
  props: {
    cpu: { type: Array, default: () => [] }, // [[ts, percent], ...]
    mem: { type: Array, default: () => [] }, // [[ts, percent], ...]
    temp: { type: Array, default: () => [] }, // [[ts, celsius], ...]
    height: { type: String, default: '320px' }
  },
  data() {
    return { chart: null }
  },
  watch: {
    cpu: {
      deep: true,
      handler() {
        this.render()
      }
    },
    mem: {
      deep: true,
      handler() {
        this.render()
      }
    },
    temp: {
      deep: true,
      handler() {
        this.render()
      }
    }
  },
  mounted() {
    this.init()
    window.addEventListener('resize', this.handleResize)
  },
  beforeDestroy() {
    window.removeEventListener('resize', this.handleResize)
    if (this.chart) this.chart.dispose()
  },
  methods: {
    init() {
      if (!this.chart) {
        this.chart = echarts.init(this.$refs.chart)
      }
      this.render()
    },
    handleResize() {
      if (this.chart) this.chart.resize()
    },
    toData(arr) {
      // ECharts 支持 [tsMs, val] 直接作为 data；这里做兜底过滤
      return (arr || [])
        .filter(
          (p) =>
            Array.isArray(p) && p.length >= 2 && !isNaN(p[0]) && !isNaN(p[1])
        )
        .map((p) => [Number(p[0]), Number(p[1])])
    },
    render() {
      if (!this.chart) return

      const cpuData = this.toData(this.cpu)
      const memData = this.toData(this.mem)
      const tempData = this.toData(this.temp)

      const option = {
        grid: { left: 48, right: 56, top: 36, bottom: 36 },
        tooltip: {
          trigger: 'axis',
          axisPointer: { type: 'cross' },
          formatter: (items) => {
            if (!items || !items.length) return ''
            const dt = new Date(items[0].value[0])
            const hh = String(dt.getHours()).padStart(2, '0')
            const mm = String(dt.getMinutes()).padStart(2, '0')
            const time = `${hh}:${mm}`
            const lines = items.map((it) => {
              const v = it.value[1]
              const unit = it.seriesName === 'Temperature' ? '℃' : '%'
              return `${it.marker}${it.seriesName}: ${
                typeof v === 'number' ? v.toFixed(2) : v
              }${unit}`
            })
            return `${time}<br/>${lines.join('<br/>')}`
          }
        },
        legend: {
          top: 6,
          data: ['CPU', 'Memory', 'Temperature']
        },
        xAxis: {
          type: 'time',
          axisLine: { lineStyle: { color: '#e5e7eb' }},
          axisLabel: { color: '#6b7280' },
          splitLine: { show: false }
        },
        yAxis: [
          {
            type: 'value',
            name: '%',
            min: 0,
            max: 100,
            axisLabel: { color: '#6b7280', formatter: '{value}%' },
            axisLine: { show: false },
            splitLine: { lineStyle: { color: '#f3f4f6' }}
          },
          {
            type: 'value',
            name: '℃',
            position: 'right',
            axisLabel: { color: '#6b7280', formatter: '{value}℃' },
            axisLine: { show: false },
            splitLine: { show: false }
          }
        ],
        series: [
          {
            name: 'CPU',
            type: 'line',
            smooth: true,
            showSymbol: false,
            yAxisIndex: 0,
            data: cpuData,
            lineStyle: { width: 2 },
            areaStyle: { opacity: 0.05 }
          },
          {
            name: 'Memory',
            type: 'line',
            smooth: true,
            showSymbol: false,
            yAxisIndex: 0,
            data: memData,
            lineStyle: { width: 2 },
            areaStyle: { opacity: 0.1 }
          },
          {
            name: 'Temperature',
            type: 'line',
            smooth: true,
            showSymbol: false,
            yAxisIndex: 1,
            data: tempData,
            lineStyle: { width: 2 }
          }
        ]
      }

      this.chart.setOption(option, true)
    }
  }
}
</script>

<style scoped>
.card {
  background: #fff;
  border-radius: 12px;
  padding: 12px 12px 4px 12px;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.04);
}
.card-title {
  font-size: 14px;
  font-weight: 600;
  color: #111827;
  padding: 4px 4px 8px;
}
</style>
