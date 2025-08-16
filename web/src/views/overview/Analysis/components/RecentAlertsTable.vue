<template>
  <div class="card">
    <div class="card-title">
      <span>Recent Alerts</span>
      <div class="actions">
        <el-select
          v-if="showNsFilter"
          v-model="ns"
          size="mini"
          clearable
          placeholder="Namespace"
        >
          <el-option label="All" :value="''" />
          <el-option v-for="n in namespaces" :key="n" :label="n" :value="n" />
        </el-select>
      </div>
    </div>

    <el-table
      ref="tbl"
      v-loading="loading"
      :data="filtered"
      size="mini"
      stripe
      :border="false"
      :max-height="tableHeight"
      class="alerts-table"
    >
      <el-table-column label="Time" width="150">
        <template slot-scope="{ row }">
          {{ fmtTime(row.time) }}
        </template>
      </el-table-column>

      <el-table-column label="Severity" width="110">
        <template slot-scope="{ row }">
          <el-tag :type="sevType(row.severity)" effect="dark" size="small">
            {{ cap(row.severity) }}
          </el-tag>
        </template>
      </el-table-column>

      <el-table-column label="Source" prop="kind" width="120" />
      <el-table-column label="Namespace" prop="namespace" width="140" />

      <el-table-column label="Message" min-width="260" show-overflow-tooltip>
        <template slot-scope="{ row }">
          <span class="msg">{{ row.message || row.reason }}</span>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script>
export default {
  name: 'RecentAlertsTable',
  props: {
    items: { type: Array, default: () => [] },
    loading: { type: Boolean, default: false },
    namespace: { type: String, default: '' },
    showNsFilter: { type: Boolean, default: true },
    visibleRows: { type: Number, default: 5 }
  },
  data() {
    return {
      ns: this.namespace || '',
      tableHeight: 320 // 初值，mounted 后会被 computeHeight 覆盖
    }
  },
  computed: {
    namespaces() {
      const set = new Set();
      (this.items || []).forEach((it) => it.namespace && set.add(it.namespace))
      return Array.from(set).sort()
    },
    sorted() {
      const arr = (this.items || []).slice()
      const toTs = (s) => {
        const t = Date.parse(s)
        return isNaN(t) ? 0 : t
      }
      return arr.sort((a, b) => toTs(b.time) - toTs(a.time))
    },
    filtered() {
      if (!this.ns) return this.sorted
      return this.sorted.filter((it) => it.namespace === this.ns)
    }
  },
  watch: {
    namespace(v) {
      this.ns = v || ''
      this.$nextTick(this.computeHeight)
    },
    items: {
      deep: true,
      handler() {
        this.$nextTick(this.computeHeight)
      }
    }
  },
  mounted() {
    this.$nextTick(() => {
      this.computeHeight()
      // 延迟一次，确保首次渲染的行高拿到真实值
      setTimeout(this.computeHeight, 0)
      window.addEventListener('resize', this.computeHeight, { passive: true })
    })
  },
  beforeDestroy() {
    window.removeEventListener('resize', this.computeHeight)
  },
  methods: {
    computeHeight() {
      const tableVm = this.$refs.tbl
      const el = tableVm && tableVm.$el
      if (!el) return

      // 表头容器
      const header = el.querySelector('.el-table__header-wrapper')
      const headerH = header ? header.offsetHeight : 40

      // 任意一行（优先找 body 里的）
      const body = el.querySelector('.el-table__body-wrapper')
      const anyRow = body && body.querySelector('.el-table__row')
      let rowH = anyRow ? anyRow.offsetHeight : 0

      // 如果此时没有行（如 items 为空或还没渲染），给个更接近 mini 尺寸的保守值
      if (!rowH || rowH < 24) rowH = 28 // mini 行高大约 28

      this.tableHeight = headerH + this.visibleRows * rowH
    },
    fmtTime(s) {
      const t = new Date(s)
      if (isNaN(t)) return s || '--'
      const hh = String(t.getHours()).padStart(2, '0')
      const mm = String(t.getMinutes()).padStart(2, '0')
      return `${hh}:${mm}`
    },
    sevType(s) {
      const k = String(s || '').toLowerCase()
      if (k === 'critical' || k === 'error') return 'danger'
      if (k === 'warning' || k === 'warn') return 'warning'
      return 'info'
    },
    cap(s) {
      if (!s) return '--'
      const k = String(s).toLowerCase()
      return k.charAt(0).toUpperCase() + k.slice(1)
    }
  }
}
</script>

<style scoped>
.card {
  background: #fff;
  border-radius: 12px;
  padding: 12px;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.04);
  display: flex;
  flex-direction: column;
  /* 防止父容器拉伸子元素高度：留空即可，由 el-table 自己的 max-height 控制滚动 */
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
.alerts-table ::v-deep .el-table__header th {
  background: #fafafa;
}
.msg {
  color: #374151;
}
</style>
