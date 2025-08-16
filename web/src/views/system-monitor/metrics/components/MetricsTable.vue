<template>
  <div class="pod-table-container">
    <div class="table-title">
      <h2>Node Metrics List</h2>
      <hr>
    </div>

    <div class="toolbar">
      <div class="row-size-selector">
        Show
        <el-select
          v-model="pageSize"
          class="row-size-dropdown"
          size="small"
          @change="handlePageSizeChange"
        >
          <el-option
            v-for="num in [5, 10, 20, 30]"
            :key="num"
            :label="num"
            :value="num"
          />
        </el-select>
        items
      </div>
    </div>

    <!-- ✅ 包一层用于样式作用域 -->
    <div class="centered-table">
      <el-table
        ref="table"
        :data="pagedRows"
        border
        style="width: 100%"
        :header-cell-style="{
          background: '#f5f7fa',
          color: '#333',
          fontWeight: 600,
          // 不在这里设置 textAlign，交给 CSS 统一处理
        }"
        empty-text="No Metrics data available"
        @sort-change="handleSortChange"
      >
        <el-table-column
          prop="node"
          label="Node"
          width="160"
          sortable="custom"
        />

        <el-table-column
          prop="cpuPercent"
          label="CPU%"
          width="120"
          sortable="custom"
        >
          <template slot-scope="{ row }">
            {{ fmtPct(row.cpuPercent) }}
          </template>
        </el-table-column>

        <el-table-column
          prop="memoryPercent"
          label="Memory%"
          width="120"
          sortable="custom"
        >
          <template slot-scope="{ row }">
            {{ fmtPct(row.memoryPercent) }}
          </template>
        </el-table-column>

        <el-table-column
          prop="cpuTemp"
          label="CPUTemp(°C)"
          width="140"
          sortable="custom"
        >
          <template slot-scope="{ row }">
            {{ fmtTemp(row.cpuTemp) }}
          </template>
        </el-table-column>

        <el-table-column
          prop="diskPercent"
          label="DiskUsed%"
          width="140"
          sortable="custom"
        >
          <template slot-scope="{ row }">
            {{ fmtPct(row.diskPercent) }}
          </template>
        </el-table-column>

        <el-table-column
          prop="eth0Tx"
          label="eth0 出站"
          width="140"
          sortable="custom"
        />
        <el-table-column
          prop="eth0Rx"
          label="eth0 入栈"
          width="140"
          sortable="custom"
        />
        <el-table-column
          prop="topCpuProcess"
          label="Top CPU Process"
          min-width="180"
          sortable="custom"
        />

        <el-table-column
          prop="timestamp"
          label="Timestamp"
          width="230"
          sortable="custom"
        >
          <template slot-scope="{ row }">
            <span :title="row.timestamp">{{ fmtTime(row.timestamp) }}</span>
          </template>
        </el-table-column>

        <el-table-column label="Actions" fixed="right" width="120">
          <template slot-scope="{ row }">
            <div class="action-buttons">
              <el-button
                size="mini"
                type="primary"
                plain
                :style="{ padding: '4px 8px', fontSize: '12px' }"
                icon="el-icon-view"
                @click="$emit('view', row)"
              >
                View
              </el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <el-pagination
      class="pagination"
      background
      small
      layout="prev, pager, next"
      :page-size="pageSize"
      :current-page="currentPage"
      :total="rows.length"
      @current-change="handlePageChange"
    />
  </div>
</template>

<script>
export default {
  name: 'MetricsTable',
  props: { rows: { type: Array, required: true }},
  data() {
    return {
      pageSize: 10,
      currentPage: 1,
      sortProp: '',
      sortOrder: '',
      _lastClickedProp: ''
    }
  },
  computed: {
    sortedRows() {
      const arr = this.rows.slice()
      const { sortProp: prop, sortOrder: order } = this
      if (!prop || !order) return arr

      const dir = order === 'ascending' ? 1 : -1
      return arr.sort((a, b) => {
        const va = this.getSortValue(a, prop)
        const vb = this.getSortValue(b, prop)

        const aInvalid = va == null || Number.isNaN(va)
        const bInvalid = vb == null || Number.isNaN(vb)
        if (aInvalid && bInvalid) return 0
        if (aInvalid) return 1
        if (bInvalid) return -1

        if (va > vb) return dir
        if (va < vb) return -dir
        return 0
      })
    },
    pagedRows() {
      const start = (this.currentPage - 1) * this.pageSize
      return this.sortedRows.slice(start, start + this.pageSize)
    }
  },
  methods: {
    handlePageChange(page) {
      this.currentPage = page
    },
    handlePageSizeChange(size) {
      this.pageSize = size
      this.currentPage = 1
    },
    handleSortChange({ prop, order }) {
      if (!prop) {
        this.sortProp = ''
        this.sortOrder = ''
        return
      }
      if (this._lastClickedProp !== prop) {
        this._lastClickedProp = prop
        if (order === 'ascending') {
          this.$nextTick(() => {
            this.$refs.table && this.$refs.table.sort(prop, 'descending')
          })
          return
        }
      }
      this.sortProp = prop
      this.sortOrder = order || ''
      this.currentPage = 1
    },
    getSortValue(row, prop) {
      switch (prop) {
        case 'cpuPercent':
        case 'memoryPercent':
        case 'cpuTemp':
        case 'diskPercent':
          return this.toNum(row[prop])
        case 'eth0Tx':
        case 'eth0Rx':
          return this.parseNetSpeed(row[prop])
        case 'timestamp':
          return this.parseIsoToMs(row.timestamp)
        case 'node':
        case 'topCpuProcess':
          return (row[prop] || '').toString().toLowerCase()
        default:
          return this.toNum(row[prop])
      }
    },
    toNum(v) {
      if (v == null || v === '') return NaN
      if (typeof v === 'string' && v.trim().endsWith('%')) return parseFloat(v)
      const n = Number(v)
      return Number.isFinite(n) ? n : NaN
    },
    parseNetSpeed(v) {
      if (v == null || v === '' || v === '-') return NaN
      if (typeof v === 'number') return v
      const s = String(v).trim()
      const m = s.match(/^([\d.]+)\s*([KMG]?B)\/s$/i)
      if (!m) return NaN
      const num = parseFloat(m[1])
      if (!Number.isFinite(num)) return NaN
      const unit = m[2].toUpperCase()
      if (unit === 'B') return num / 1024
      if (unit === 'KB') return num
      if (unit === 'MB') return num * 1024
      if (unit === 'GB') return num * 1024 * 1024
      return NaN
    },
    fmtTime(ts) {
      const ms = this.parseIsoToMs(ts)
      if (!Number.isFinite(ms)) return '-'
      const d = new Date(ms)
      const pad = (n, w = 2) => String(n).padStart(w, '0')
      const yyyy = d.getFullYear()
      const MM = pad(d.getMonth() + 1)
      const DD = pad(d.getDate())
      const hh = pad(d.getHours())
      const mm = pad(d.getMinutes())
      const ss = pad(d.getSeconds())
      const mss = String(d.getMilliseconds()).padStart(3, '0')
      return `${yyyy}-${MM}-${DD} ${hh}:${mm}:${ss}.${mss}`
    },
    parseIsoToMs(ts) {
      if (typeof ts !== 'string') return NaN
      const m = ts.match(
        /^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})(\.(\d+))?(Z|[+-]\d{2}:\d{2})?$/
      )
      if (!m) {
        const t = Date.parse(ts)
        return Number.isFinite(t) ? t : NaN
      }
      const base = m[1]
      const frac = m[3] || ''
      const tz = m[4] || 'Z'
      const ms3 = (frac + '000').slice(0, 3)
      const iso = `${base}.${ms3}${tz}`
      const t = Date.parse(iso)
      return Number.isFinite(t) ? t : NaN
    },
    fmtPct(val) {
      if (val == null || val === '') return '-'
      if (typeof val === 'string') { return val.endsWith('%') ? val : `${Number(val).toFixed(2)}%` }
      const n = Number(val)
      return Number.isFinite(n) ? n.toFixed(2) + '%' : '-'
    },
    fmtTemp(val) {
      const n = Number(val)
      return Number.isFinite(n) ? n.toFixed(2) : '-'
    }
  }
}
</script>

<style scoped>
.pod-table-container {
  padding: 16px;
}
.table-title {
  margin-bottom: 16px;
}
.toolbar {
  margin-bottom: 12px;
}

/* 操作按钮容器也居中 */
.action-buttons {
  display: flex;
  gap: 6px;
  justify-content: center;
}

/* ✅ 统一表头与单元格文本居中（兼容多种 deep 语法） */
.centered-table >>> .el-table .cell,
.centered-table >>> .el-table th .cell,
.centered-table /deep/ .el-table .cell,
.centered-table /deep/ .el-table th .cell,
.centered-table ::v-deep .el-table .cell,
.centered-table ::v-deep .el-table th .cell {
  text-align: center !important;
}
</style>
