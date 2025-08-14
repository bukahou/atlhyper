<template>
  <div class="card health-card">
    <div class="header">
      <div class="title">Cluster Health</div>
      <el-tag :type="tagType" size="small" effect="dark">{{
        data.status
      }}</el-tag>
    </div>

    <div class="reason" :title="data.reason">{{ data.reason || "—" }}</div>

    <div class="meter">
      <div class="meter-row">
        <span class="meter-label">Node Ready</span>
        <span class="meter-val">{{ fmtPct(data.node_ready_pct) }}</span>
      </div>
      <el-progress
        :percentage="clampPct(data.node_ready_pct)"
        :stroke-width="10"
        :status="progressStatus(data.node_ready_pct)"
        :show-text="false"
      />
    </div>

    <div class="meter">
      <div class="meter-row">
        <span class="meter-label">Pod Healthy</span>
        <span class="meter-val">{{ fmtPct(data.pod_healthy_pct) }}</span>
      </div>
      <el-progress
        :percentage="clampPct(data.pod_healthy_pct)"
        :stroke-width="10"
        :status="progressStatus(data.pod_healthy_pct)"
        :show-text="false"
      />
    </div>
  </div>
</template>

<script>
export default {
  name: "HealthCard",
  props: {
    data: {
      type: Object,
      required: true,
      // 结构：{ status, reason, node_ready_pct, pod_healthy_pct }
    },
  },
  computed: {
    tagType() {
      const s = (this.data.status || "").toLowerCase();
      if (s === "healthy") return "success";
      if (s === "degraded") return "warning";
      return "danger";
    },
  },
  methods: {
    fmtPct(v) {
      if (v === null || v === undefined || isNaN(v)) return "--";
      return `${(+v).toFixed(2)}%`;
    },
    clampPct(v) {
      const x = Number(v);
      if (isNaN(x)) return 0;
      return Math.max(0, Math.min(100, x));
    },
    progressStatus(v) {
      const x = Number(v);
      if (x >= 98) return "success";
      if (x >= 90) return ""; // 默认蓝色
      return "exception";
    },
  },
};
</script>

<style scoped>
.card {
  background: #fff;
  border-radius: 12px;
  padding: 14px 16px;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.04);
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-height: 160px; /* 统一高度 */
  height: 100%;
  border: 1px solid #f0f3f7;
  transition: all 0.2s ease-in-out;
}
.card:hover {
  box-shadow: 0 6px 18px rgba(0, 0, 0, 0.08);
  transform: translateY(-1px);
}

.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.title {
  font-size: 15px;
  font-weight: 600;
  color: #0f172a;
  letter-spacing: 0.2px;
}

.reason {
  font-size: 13px;
  color: #64748b;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.meter {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.meter-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 13px;
  color: #334155;
}
.meter-label {
  color: #475569;
}
.meter-val {
  color: #0f172a;
  font-variant-numeric: tabular-nums;
}

/* 让进度条更圆润 */
:deep(.el-progress-bar__outer) {
  border-radius: 6px;
}
:deep(.el-progress-bar__inner) {
  border-radius: 6px;
}
</style>
