<template>
  <div class="card" :class="metric">
    <div class="card-title">
      <span>Node Resource Usage</span>
      <div class="actions">
        <el-radio-group v-model="metric" size="mini">
          <el-radio-button label="cpu">CPU</el-radio-button>
          <el-radio-button label="mem">Memory</el-radio-button>
        </el-radio-group>
      </div>
    </div>

    <!-- 列表区域：固定高度（可见行数 × 行高），内部可滚动 -->
    <div
      class="rows"
      ref="rows"
      :style="{ maxHeight: rowsMaxHeight + 'px', overflowY: 'auto' }"
    >
      <div v-for="(n, idx) in paged" :key="n.node_name + '-' + idx" class="row">
        <!-- 左：节点信息 -->
        <div class="left">
          <span class="dot" :class="{ ok: n.ready, bad: !n.ready }"></span>
          <span class="name" :title="n.node_name">{{ n.node_name }}</span>
          <el-tag
            v-if="n.role === 'control-plane'"
            size="mini"
            type="info"
            effect="plain"
            >control-plane</el-tag
          >
        </div>

        <!-- 中：条形百分比 -->
        <div class="meter" :class="[metric, levelClass(val(n))]">
          <div class="fill" :style="{ width: clampPct(val(n)) + '%' }"></div>
          <div class="ticks"><i></i><i></i><i></i></div>
        </div>

        <!-- 右：数字百分比 -->
        <span class="val" :class="levelClass(val(n))">{{
          fmtPct(val(n))
        }}</span>
      </div>

      <div v-if="!paged.length" class="empty">No nodes</div>
    </div>

    <div class="footer" v-if="pages > 1">
      <el-button
        type="text"
        icon="el-icon-arrow-left"
        @click="prev"
        :disabled="page <= 1"
      />
      <span class="page">{{ page }} / {{ pages }}</span>
      <el-button
        type="text"
        icon="el-icon-arrow-right"
        @click="next"
        :disabled="page >= pages"
      />
    </div>
  </div>
</template>

<script>
export default {
  name: "NodeResourceUsage",
  props: {
    items: { type: Array, default: () => [] }, // node_usages
    pageSize: { type: Number, default: 5 }, // Top N
    visibleRows: { type: Number, default: 5 }, // 与 RecentAlertsTable 对齐
  },
  data() {
    return {
      metric: "cpu",
      page: 1,
      rowsMaxHeight: 5 * 44, // 初值，mounted 后按真实行高计算
    };
  },
  watch: {
    metric() {
      this.page = 1;
      this.$nextTick(this.computeRowsMaxHeight);
    },
    items: {
      deep: true,
      handler() {
        if (this.page > this.pages) this.page = 1;
        this.$nextTick(this.computeRowsMaxHeight);
      },
    },
    pageSize() {
      this.$nextTick(this.computeRowsMaxHeight);
    },
    visibleRows() {
      this.$nextTick(this.computeRowsMaxHeight);
    },
  },
  mounted() {
    this.$nextTick(() => {
      this.computeRowsMaxHeight();
      // 再次延迟，确保首屏真实行高
      setTimeout(this.computeRowsMaxHeight, 0);
      window.addEventListener("resize", this.computeRowsMaxHeight, {
        passive: true,
      });
    });
  },
  beforeDestroy() {
    window.removeEventListener("resize", this.computeRowsMaxHeight);
  },
  computed: {
    sorted() {
      const arr = (this.items || []).slice();
      const get = (n) =>
        (this.metric === "cpu"
          ? Number(n.cpu_percent)
          : Number(n.memory_percent)) || 0;
      return arr.sort((a, b) => get(b) - get(a));
    },
    pages() {
      const len = this.sorted.length;
      return len ? Math.ceil(len / this.pageSize) : 1;
    },
    paged() {
      const start = (this.page - 1) * this.pageSize;
      return this.sorted.slice(start, start + this.pageSize);
    },
  },
  methods: {
    computeRowsMaxHeight() {
      // 参考 RecentAlertsTable：用实际行高来计算可见区域上限
      const container = this.$refs.rows;
      if (!container) return;

      // 找到一行的真实高度
      const anyRow = container.querySelector(".row");
      let rowH = anyRow ? anyRow.offsetHeight : 0;
      if (!rowH || rowH < 36) rowH = 44; // 合理的保底行高

      this.rowsMaxHeight = this.visibleRows * rowH;
    },
    val(n) {
      return this.metric === "cpu" ? n.cpu_percent : n.memory_percent;
    },
    fmtPct(v) {
      if (v === null || v === undefined || isNaN(v)) return "--";
      return `${Number(v).toFixed(2)}%`;
    },
    clampPct(v) {
      const x = Number(v);
      if (isNaN(x)) return 0;
      return Math.max(0, Math.min(100, x));
    },
    levelClass(v) {
      const x = Number(v) || 0;
      if (x >= 85) return "lv-high";
      if (x >= 60) return "lv-mid";
      return "lv-low";
    },
    prev() {
      if (this.page > 1) this.page--;
    },
    next() {
      if (this.page < this.pages) this.page++;
    },
  },
};
</script>

<style scoped>
.card {
  background: #fff;
  border-radius: 12px;
  padding: 12px;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.04);
  border: 1px solid #eef2f6;
  display: flex;
  flex-direction: column;
  /* 不强制等高，保持与 RecentAlertsTable 一致，由 rows 的 max-height 控制视觉高度 */
}
.card-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 14px;
  font-weight: 600;
  color: #111827;
  padding: 2px 4px 10px;
}
.actions {
  display: flex;
  gap: 8px;
  align-items: center;
}

/* 列表区域：固定上限高度，内部滚动 */
.rows {
  display: flex;
  flex-direction: column;
  gap: 10px;
  flex: 0 1 auto;
  min-height: 0;
}

/* 行：三列布局：左(信息) | 中(条形百分比) | 右(数字百分比) */
.row {
  display: grid;
  grid-template-columns: minmax(180px, 0.45fr) minmax(140px, 1fr) auto;
  align-items: center;
  gap: 12px;
  padding: 6px 8px;
  border-radius: 8px;
  transition: background-color 0.15s ease;
}
.row:hover {
  background: #f8fafc;
}

/* 左：节点信息 */
.left {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.name {
  font-size: 13px;
  color: #111827;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 中：条形百分比 */
.meter {
  position: relative;
  height: 10px;
  background: #f1f5f9;
  border: 1px solid #e5eaf0;
  border-radius: 999px;
  overflow: hidden;
  min-width: 140px;
}
.meter .fill {
  height: 100%;
  width: 0%;
  transition: width 0.35s ease;
  border-radius: 999px;
}
.meter .ticks {
  position: absolute;
  inset: 0;
  pointer-events: none;
}
.meter .ticks i {
  position: absolute;
  top: 0;
  bottom: 0;
  width: 1px;
  background: rgba(0, 0, 0, 0.06);
}
.meter .ticks i:nth-child(1) {
  left: 25%;
}
.meter .ticks i:nth-child(2) {
  left: 50%;
}
.meter .ticks i:nth-child(3) {
  left: 75%;
}

/* 指标主色：CPU 橙 / Memory 绿 */
.card.cpu .meter .fill {
  background: linear-gradient(90deg, #fdba74, #f97316);
}
.card.mem .meter .fill {
  background: linear-gradient(90deg, #86efac, #10b981);
}

/* 阈值叠加边框颜色（低/中/高） */
.meter.lv-low {
  border-color: #bbf7d0;
}
.meter.lv-mid {
  border-color: #fed7aa;
}
.meter.lv-high {
  border-color: #fecaca;
}

/* 右：数字百分比（徽标） */
.val {
  font-size: 12px;
  font-weight: 600;
  text-align: right;
  padding: 4px 8px;
  border-radius: 8px;
  min-width: 76px;
  font-variant-numeric: tabular-nums;
  background: #f8fafc;
  border: 1px solid #eef2f6;
  color: #0f172a;
}
.val.lv-low {
  background: #f0fdf4;
  border-color: #dcfce7;
  color: #065f46;
}
.val.lv-mid {
  background: #fff7ed;
  border-color: #fde68a;
  color: #7c2d12;
}
.val.lv-high {
  background: #fef2f2;
  border-color: #fecaca;
  color: #7f1d1d;
}

/* 状态点 */
.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #d1d5db;
  display: inline-block;
  box-shadow: inset 0 0 0 1px rgba(0, 0, 0, 0.04);
}
.dot.ok {
  background: #10b981;
}
.dot.bad {
  background: #ef4444;
}

/* 底部分页（有分页时整体略高一点，与表格相差不大；需要绝对对齐可把 rowsMaxHeight 适当减去 24px） */
.footer {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 8px;
  padding-top: 8px;
  color: #6b7280;
}
.page {
  font-size: 12px;
  min-width: 48px;
  text-align: center;
}
.empty {
  color: #9ca3af;
  font-size: 12px;
  text-align: center;
  padding: 8px 0;
}
</style>
