<template>
  <div class="card">
    <div class="card-title">Alert Trends</div>
    <div ref="chart" :style="{ width: '100%', height }"></div>
  </div>
</template>

<script>
import * as echarts from "echarts";

export default {
  name: "AlertTrendsChart",
  props: {
    series: { type: Array, default: () => [] }, // [{ ts, critical, warning, info }]
    height: { type: String, default: "320px" },
  },
  data() {
    return { chart: null };
  },
  mounted() {
    this.init();
    window.addEventListener("resize", this.onResize);
  },
  beforeDestroy() {
    window.removeEventListener("resize", this.onResize);
    if (this.chart) this.chart.dispose();
  },
  watch: {
    series: {
      deep: true,
      handler() {
        this.render();
      },
    },
  },
  methods: {
    init() {
      if (!this.chart) this.chart = echarts.init(this.$refs.chart);
      this.render();
    },
    onResize() {
      if (this.chart) this.chart.resize();
    },
    toLine(name, color, data, key) {
      return {
        name,
        type: "line",
        stack: "total",
        areaStyle: {},
        showSymbol: false,
        smooth: true,
        lineStyle: { width: 2 },
        emphasis: { focus: "series" },
        data: data.map((p) => [p.ts, Number(p[key] || 0)]),
        itemStyle: { color },
      };
    },
    render() {
      if (!this.chart) return;
      const data = Array.isArray(this.series) ? this.series : [];
      const option = {
        grid: { left: 48, right: 24, top: 36, bottom: 36 },
        tooltip: {
          trigger: "axis",
          axisPointer: { type: "cross" },
          formatter: (items) => {
            if (!items || !items.length) return "";
            const dt = new Date(items[0].value[0]);
            const hh = String(dt.getHours()).padStart(2, "0");
            const mm = String(dt.getMinutes()).padStart(2, "0");
            const total = items.reduce(
              (s, it) => s + (Number(it.value[1]) || 0),
              0
            );
            const lines = items.map(
              (it) => `${it.marker}${it.seriesName}: ${it.value[1]}`
            );
            return `${hh}:${mm}  (total ${total})<br/>${lines.join("<br/>")}`;
          },
        },
        legend: { top: 6, data: ["Critical", "Warning", "Info"] },
        xAxis: {
          type: "time",
          axisLine: { lineStyle: { color: "#e5e7eb" } },
          axisLabel: { color: "#6b7280" },
          splitLine: { show: false },
        },
        yAxis: {
          type: "value",
          min: 0,
          axisLabel: { color: "#6b7280" },
          splitLine: { lineStyle: { color: "#f3f4f6" } },
        },
        series: [
          this.toLine("Critical", "#EF4444", data, "critical"),
          this.toLine("Warning", "#F59E0B", data, "warning"),
          this.toLine("Info", "#3B82F6", data, "info"),
        ],
      };
      this.chart.setOption(option, true);
    },
  },
};
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
