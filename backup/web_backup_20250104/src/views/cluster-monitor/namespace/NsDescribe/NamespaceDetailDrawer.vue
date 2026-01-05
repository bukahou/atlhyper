<template>
  <el-drawer
    :visible="visible"
    :size="width"
    :with-header="false"
    custom-class="ns-describe-drawer"
    append-to-body
    :destroy-on-close="true"
    :close-on-click-modal="true"
    :before-close="handleBeforeClose"
    @update:visible="$emit('update:visible', $event)"
    @close="handleClose"
  >
    <!-- 顶部摘要栏（吸顶） -->
    <div class="summary-bar">
      <div class="left">
        <span class="ns-name">{{ ns.name }}</span>
        <el-tag size="mini" :type="phaseTagType">{{
          ns.phase || "Unknown"
        }}</el-tag>
        <span class="age">Age {{ ns.age || "-" }}</span>
        <span class="age">Created {{ ns.createdAt || "-" }}</span>
        <el-tag size="mini" type="info">Labels {{ n(ns.labelCount) }}</el-tag>
        <el-tag
          size="mini"
          type="info"
        >Annotations {{ n(ns.annotationCount) }}</el-tag>
      </div>
    </div>

    <!-- 主体：左目录 + 右内容 -->
    <div class="main">
      <!-- 左：目录 -->
      <div class="sidenav">
        <el-menu
          :default-active="activeSection"
          class="menu"
          @select="scrollTo"
        >
          <el-menu-item index="overview">概览</el-menu-item>
          <el-menu-item index="labels">标签</el-menu-item>
          <el-menu-item index="pods">Pods</el-menu-item>
          <el-menu-item index="resources">资源对象</el-menu-item>
          <el-menu-item index="metrics">指标</el-menu-item>
          <el-menu-item index="raw">原始（JSON）</el-menu-item>
        </el-menu>
      </div>

      <!-- 右：内容（可滚） -->
      <div ref="scrollEl" class="content" @scroll="onScroll">
        <!-- 概览 -->
        <section ref="overview" data-id="overview" class="section">
          <h3 class="section-title">概览</h3>
          <div class="kv">
            <div>
              <span>名称</span><b>{{ ns.name }}</b>
            </div>
            <div>
              <span>阶段</span><b>{{ ns.phase || "Unknown" }}</b>
            </div>
            <div>
              <span>创建时间</span><b>{{ ns.createdAt || "-" }}</b>
            </div>
            <div>
              <span>存活时长</span><b>{{ ns.age || "-" }}</b>
            </div>
            <div>
              <span>标签数</span><b>{{ n(ns.labelCount) }}</b>
            </div>
            <div>
              <span>注解数</span><b>{{ n(ns.annotationCount) }}</b>
            </div>
          </div>
        </section>

        <!-- 标签 -->
        <section ref="labels" data-id="labels" class="section">
          <h3 class="section-title">标签</h3>
          <div v-if="labelArray.length" class="labels">
            <el-tag
              v-for="(kv, i) in labelArray"
              :key="i"
              size="mini"
              class="mr8 mono"
            >
              {{ kv.k }}={{ kv.v }}
            </el-tag>
          </div>
          <div v-else class="muted">—</div>
        </section>

        <!-- Pods -->
        <section ref="pods" data-id="pods" class="section">
          <h3 class="section-title">Pods</h3>
          <div class="kv">
            <div>
              <span>总数</span><b>{{ n(ns.pods) }}</b>
            </div>
            <div>
              <span>Running</span><b>{{ n(ns.podsRunning) }}</b>
            </div>
            <div>
              <span>Pending</span><b>{{ n(ns.podsPending) }}</b>
            </div>
            <div>
              <span>Failed</span><b>{{ n(ns.podsFailed) }}</b>
            </div>
            <div>
              <span>Succeeded</span><b>{{ n(ns.podsSucceeded) }}</b>
            </div>
          </div>
          <div class="progress-row">
            <div class="bar">
              <div
                class="bar-inner"
                :style="{ width: podsRunningPctStr }"
              />
            </div>
            <div class="val">Running {{ podsRunningPctStr }}</div>
          </div>
        </section>

        <!-- 资源对象汇总 -->
        <section ref="resources" data-id="resources" class="section">
          <h3 class="section-title">资源对象</h3>
          <div class="kv">
            <div>
              <span>Deployments</span><b>{{ n(ns.deployments) }}</b>
            </div>
            <div>
              <span>StatefulSets</span><b>{{ n(ns.statefulSets) }}</b>
            </div>
            <div>
              <span>DaemonSets</span><b>{{ n(ns.daemonSets) }}</b>
            </div>
            <div>
              <span>Jobs</span><b>{{ n(ns.jobs) }}</b>
            </div>
            <div>
              <span>CronJobs</span><b>{{ n(ns.cronJobs) }}</b>
            </div>
            <div>
              <span>Services</span><b>{{ n(ns.services) }}</b>
            </div>
            <div>
              <span>Ingresses</span><b>{{ n(ns.ingresses) }}</b>
            </div>
            <div>
              <span>ConfigMaps</span><b>{{ n(ns.configMaps) }}</b>
            </div>
            <div>
              <span>Secrets</span><b>{{ n(ns.secrets) }}</b>
            </div>
            <div>
              <span>PVCs</span><b>{{ n(ns.persistentVolumeClaims) }}</b>
            </div>
            <div>
              <span>NetworkPolicies</span><b>{{ n(ns.networkPolicies) }}</b>
            </div>
            <div>
              <span>ServiceAccounts</span><b>{{ n(ns.serviceAccounts) }}</b>
            </div>
          </div>
        </section>

        <!-- 指标 -->
        <section ref="metrics" data-id="metrics" class="section">
          <h3 class="section-title">指标</h3>

          <h4 class="sub">CPU</h4>
          <div class="kv">
            <div>
              <span>使用 / 请求 / 限制</span>
              <b
                class="mono"
              >{{ cpuUsageStr }} / {{ cpuReqStr }} / {{ cpuLimStr }}</b>
            </div>
            <div>
              <span>利用率（基于 {{ cpuUtilBasis }}）</span>
              <b>{{ cpuPctStr }}</b>
            </div>
          </div>
          <div class="progress-row">
            <div class="bar">
              <div class="bar-inner" :style="{ width: cpuPctStr }" />
            </div>
            <div class="val">{{ cpuPctStr }}</div>
          </div>

          <h4 class="sub">内存</h4>
          <div class="kv">
            <div>
              <span>使用 / 请求 / 限制</span>
              <b
                class="mono"
              >{{ memUsageStr }} / {{ memReqStr }} / {{ memLimStr }}</b>
            </div>
            <div>
              <span>利用率（基于 {{ memUtilBasis }}）</span>
              <b>{{ memPctStr }}</b>
            </div>
          </div>
          <div class="progress-row">
            <div class="bar">
              <div class="bar-inner" :style="{ width: memPctStr }" />
            </div>
            <div class="val">{{ memPctStr }}</div>
          </div>
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
export default {
  name: 'NamespaceDetailDrawer',
  props: {
    visible: { type: Boolean, default: false },
    ns: { type: Object, required: true },
    width: { type: String, default: '45%' }
  },
  data() {
    return { activeSection: 'overview' }
  },
  computed: {
    phaseTagType() {
      const p = (this.ns.phase || '').toLowerCase()
      if (p === 'active') return 'success'
      if (p === 'terminating') return 'warning'
      return 'info'
    },
    labelArray() {
      const obj = this.ns.labels || {}
      return Object.keys(obj).map((k) => ({ k, v: obj[k] }))
    },
    podsRunningPctStr() {
      const total = Number(this.ns.pods || 0)
      const run = Number(this.ns.podsRunning || 0)
      const pct = total > 0 ? (run / total) * 100 : 0
      return this.clampPct(pct).toFixed(0) + '%'
    },

    // ---- Metrics（去掉 ?. / ??）----
    cpuUtilBasis() {
      const m =
        this.ns.metrics && this.ns.metrics.cpu ? this.ns.metrics.cpu : {}
      return m.utilBasis || 'limit'
    },
    memUtilBasis() {
      const m =
        this.ns.metrics && this.ns.metrics.memory ? this.ns.metrics.memory : {}
      return m.utilBasis || 'limit'
    },

    cpuUsageStr() {
      const m =
        this.ns.metrics && this.ns.metrics.cpu ? this.ns.metrics.cpu : {}
      return this.fmtCpuMilliStr(m.usage)
    },
    cpuReqStr() {
      const m =
        this.ns.metrics && this.ns.metrics.cpu ? this.ns.metrics.cpu : {}
      return this.fmtCpuMilliStr(m.requests)
    },
    cpuLimStr() {
      const m =
        this.ns.metrics && this.ns.metrics.cpu ? this.ns.metrics.cpu : {}
      return this.fmtCpuMilliStr(m.limits)
    },
    cpuPctStr() {
      const m =
        this.ns.metrics && this.ns.metrics.cpu ? this.ns.metrics.cpu : {}
      if (typeof m.utilPct === 'number') {
        return this.clampPct(m.utilPct).toFixed(1) + '%'
      }
      const usage = this.parseCpuToMilli(m.usage)
      const denom =
        (m.utilBasis || 'limit') === 'request'
          ? this.parseCpuToMilli(m.requests)
          : this.parseCpuToMilli(m.limits)
      const pct = denom > 0 ? (usage / denom) * 100 : 0
      return this.clampPct(pct).toFixed(1) + '%'
    },

    memUsageStr() {
      const m =
        this.ns.metrics && this.ns.metrics.memory ? this.ns.metrics.memory : {}
      return this.fmtBytesStr(m.usage)
    },
    memReqStr() {
      const m =
        this.ns.metrics && this.ns.metrics.memory ? this.ns.metrics.memory : {}
      return this.fmtBytesStr(m.requests)
    },
    memLimStr() {
      const m =
        this.ns.metrics && this.ns.metrics.memory ? this.ns.metrics.memory : {}
      return this.fmtBytesStr(m.limits)
    },
    memPctStr() {
      const m =
        this.ns.metrics && this.ns.metrics.memory ? this.ns.metrics.memory : {}
      if (typeof m.utilPct === 'number') {
        return this.clampPct(m.utilPct).toFixed(1) + '%'
      }
      const usage = this.parseBytes(m.usage)
      const denom =
        (m.utilBasis || 'limit') === 'request'
          ? this.parseBytes(m.requests)
          : this.parseBytes(m.limits)
      const pct = denom > 0 ? (usage / denom) * 100 : 0
      return this.clampPct(pct).toFixed(1) + '%'
    },

    prettyJSON() {
      try {
        return JSON.stringify(this.ns, null, 2)
      } catch (e) {
        return '{}'
      }
    }
  },
  methods: {
    handleBeforeClose(done) {
      this.$emit('update:visible', false)
      done && done()
    },
    handleClose() {
      this.$emit('update:visible', false)
    },

    // ---- helpers ----
    n(v, d = 0) {
      return v == null ? d : v
    },
    clampPct(v) {
      return Math.max(0, Math.min(100, Number(v) || 0))
    },

    // CPU：格式化毫核字符串，如 "5400" / "250m" / "0.5"
    fmtCpuMilliStr(v) {
      const m = this.parseCpuToMilli(v)
      if (m == null) return '-'
      const cores = m / 1000
      const coresStr = (cores < 10 ? cores.toFixed(1) : Math.round(cores))
        .toString()
        .replace(/\.0$/, '')
      return `${m}m (${coresStr} cores)`
    },
    parseCpuToMilli(v) {
      if (v == null || v === '') return 0
      const s = String(v).trim().toLowerCase()
      if (s.endsWith('m')) {
        const n = parseFloat(s.slice(0, -1))
        return Number.isFinite(n) ? n : 0
      }
      const n = parseFloat(s)
      if (!Number.isFinite(n)) return 0
      // 小数视为核
      return s.indexOf('.') >= 0 ? n * 1000 : n
    },

    fmtBytesStr(v) {
      const bytes = this.parseBytes(v)
      if (bytes == null) return '-'
      const units = ['B', 'Ki', 'Mi', 'Gi', 'Ti']
      let i = 0
      let val = bytes
      while (i < units.length - 1 && val >= 1024) {
        val /= 1024
        i++
      }
      const num = val < 10 ? val.toFixed(2) : val.toFixed(1)
      return `${num.replace(/\.0+$/, '').replace(/(\.\d)0$/, '$1')} ${
        units[i]
      }`
    },
    parseBytes(v) {
      if (v == null || v === '') return 0
      const s = String(v).trim().toLowerCase()
      const m = s.match(/^([\d.]+)\s*(ki|mi|gi|ti|k|m|g|t|b)?$/i)
      if (!m) return parseFloat(s) || 0
      const val = parseFloat(m[1])
      const unit = (m[2] || 'b').toLowerCase()
      if (unit === 'b') return val
      if (unit === 'k') return val * 1000
      if (unit === 'm') return val * 1000 ** 2
      if (unit === 'g') return val * 1000 ** 3
      if (unit === 't') return val * 1000 ** 4
      const map = { ki: 1024, mi: 1024 ** 2, gi: 1024 ** 3, ti: 1024 ** 4 }
      return val * (map[unit] || 1)
    },

    // 目录滚动
    scrollTo(id) {
      const el = this.$refs[id]
      if (!el || !this.$refs.scrollEl) return
      const top = el.offsetTop - 8
      this.$refs.scrollEl.scrollTo({ top, behavior: 'smooth' })
      this.activeSection = id
      this.$emit('section-change', id)
    },
    onScroll() {
      const container = this.$refs.scrollEl
      if (!container) return
      const sections = [
        'overview',
        'labels',
        'pods',
        'resources',
        'metrics',
        'raw'
      ]
      let current = sections[0]
      for (const id of sections) {
        const el = this.$refs[id]
        if (el && el.offsetTop - container.scrollTop <= 40) current = id
      }
      this.activeSection = current
    }
  }
}
</script>

<style scoped>
.ns-describe-drawer {
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
.summary-bar .ns-name {
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
  width: 220px;
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
.mr8 {
  margin-right: 8px;
}
.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas,
    "Liberation Mono", monospace;
}

.sub {
  margin: 14px 0 6px;
  font-weight: 600;
  color: #555;
}

.progress-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 8px 0 12px;
}
.progress-row .bar {
  flex: 1;
  height: 8px;
  background: #f2f3f5;
  border-radius: 4px;
  overflow: hidden;
}
.progress-row .bar-inner {
  height: 100%;
  background: #409eff;
}
.progress-row .val {
  min-width: 56px;
  text-align: right;
  color: #666;
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
