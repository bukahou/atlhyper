<template>
  <el-drawer
    :visible="visible"
    :size="width"
    :with-header="false"
    custom-class="node-describe-drawer"
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
        <span class="node-name">{{ node.name }}</span>
        <el-tag
          size="mini"
          :type="readyTagType"
        >Ready: {{ boolStr(node.ready) }}</el-tag>
        <el-tag
          size="mini"
          :type="schedTagType"
        >Schedulable: {{ boolStr(node.schedulable) }}</el-tag>
        <el-tag size="mini" type="info">{{ node.architecture || "-" }}</el-tag>
        <el-tag size="mini" type="info">{{
          node.osImage || node.os || "-"
        }}</el-tag>
        <el-tag size="mini" type="info">IP {{ node.internalIP || "-" }}</el-tag>
        <span class="age">Age {{ node.age }}</span>
      </div>
      <!-- 无右侧按钮 -->
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
          <el-menu-item index="resource">资源与用量</el-menu-item>
          <el-menu-item index="capacity">配额与可分配</el-menu-item>
          <el-menu-item index="network">网络</el-menu-item>
          <el-menu-item index="components">组件与版本</el-menu-item>
          <el-menu-item index="conditions">条件</el-menu-item>
          <el-menu-item index="labels">标签</el-menu-item>
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
              <span>名称</span><b>{{ node.name }}</b>
            </div>
            <div>
              <span>就绪</span><b>{{ boolStr(node.ready) }}</b>
            </div>
            <div>
              <span>可调度</span><b>{{ boolStr(node.schedulable) }}</b>
            </div>
            <div>
              <span>创建时间</span><b>{{ node.createdAt }}</b>
            </div>
            <div>
              <span>存活时长</span><b>{{ node.age }}</b>
            </div>
            <div>
              <span>主机名</span><b class="mono">{{ node.hostname }}</b>
            </div>
            <div>
              <span>ProviderID</span><b class="mono">{{ node.providerID || "-" }}</b>
            </div>
          </div>
        </section>

        <!-- 资源与用量 -->
        <section ref="resource" data-id="resource" class="section">
          <h3 class="section-title">资源与用量</h3>
          <div class="kv">
            <div>
              <span>CPU 使用 / 可分配</span>
              <b>{{ fmtCores(node.cpuUsageCores) }} /
                {{ fmtCores(node.cpuAllocatableCores) }}</b>
            </div>
            <div class="progress-row">
              <div class="bar">
                <div class="bar-inner" :style="{ width: cpuPercentStr }" />
              </div>
              <div class="val">{{ cpuPercentStr }}</div>
            </div>

            <div>
              <span>内存 使用 / 可分配</span>
              <b>{{ fmtGiB(node.memUsageGiB) }} /
                {{ fmtGiB(node.memAllocatableGiB) }}</b>
            </div>
            <div class="progress-row">
              <div class="bar">
                <div class="bar-inner" :style="{ width: memPercentStr }" />
              </div>
              <div class="val">{{ memPercentStr }}</div>
            </div>

            <div>
              <span>Pods 使用 / 可分配</span>
              <b>{{ node.podsUsed || 0 }} / {{ node.podsAllocatable || 0 }}</b>
            </div>
            <div class="progress-row">
              <div class="bar">
                <div class="bar-inner" :style="{ width: podsPercentStr }" />
              </div>
              <div class="val">{{ podsPercentStr }}</div>
            </div>
          </div>
        </section>

        <!-- 配额与可分配（硬件总量） -->
        <section ref="capacity" data-id="capacity" class="section">
          <h3 class="section-title">配额与可分配</h3>
          <div class="kv">
            <div>
              <span>CPU 容量</span><b>{{ fmtCores(node.cpuCapacityCores) }}</b>
            </div>
            <div>
              <span>CPU 可分配</span><b>{{ fmtCores(node.cpuAllocatableCores) }}</b>
            </div>
            <div>
              <span>内存容量</span><b>{{ fmtGiB(node.memCapacityGiB) }}</b>
            </div>
            <div>
              <span>内存可分配</span><b>{{ fmtGiB(node.memAllocatableGiB) }}</b>
            </div>
            <div>
              <span>临时存储（Ephemeral）</span><b>{{ fmtGiB(node.ephemeralStorageGiB) }}</b>
            </div>
            <div>
              <span>Pods 容量</span><b>{{ node.podsCapacity || 0 }}</b>
            </div>
            <div>
              <span>Pods 可分配</span><b>{{ node.podsAllocatable || 0 }}</b>
            </div>
          </div>
        </section>

        <!-- 网络 -->
        <section ref="network" data-id="network" class="section">
          <h3 class="section-title">网络</h3>
          <div class="kv">
            <div>
              <span>Internal IP</span><b class="mono">{{ node.internalIP }}</b>
            </div>
            <div>
              <span>PodCIDRs</span>
              <b>
                <template v-if="node.podCIDRs && node.podCIDRs.length">
                  <el-tag
                    v-for="(cidr, i) in node.podCIDRs"
                    :key="i"
                    size="mini"
                    class="mr8 mono"
                  >{{ cidr }}</el-tag>
                </template>
                <template v-else>-</template>
              </b>
            </div>
          </div>
        </section>

        <!-- 组件与版本 -->
        <section ref="components" data-id="components" class="section">
          <h3 class="section-title">组件与版本</h3>
          <div class="kv">
            <div>
              <span>OS</span><b>{{ node.os || "-" }}</b>
            </div>
            <div>
              <span>OS Image</span><b>{{ node.osImage || "-" }}</b>
            </div>
            <div>
              <span>架构</span><b>{{ node.architecture || "-" }}</b>
            </div>
            <div>
              <span>内核</span><b class="mono">{{ node.kernel || "-" }}</b>
            </div>
            <div>
              <span>容器运行时</span><b class="mono">{{ node.cri || "-" }}</b>
            </div>
            <div>
              <span>Kubelet</span><b class="mono">{{ node.kubelet || "-" }}</b>
            </div>
            <div>
              <span>Kube-Proxy</span><b class="mono">{{ node.kubeProxy || "-" }}</b>
            </div>
          </div>
        </section>

        <!-- 条件 -->
        <section ref="conditions" data-id="conditions" class="section">
          <h3 class="section-title">条件</h3>
          <div v-if="(node.conditions || []).length" class="cond-list">
            <div v-for="(c, i) in node.conditions" :key="i" class="cond-item">
              <div class="cond-head">
                <el-tag size="mini" :type="condTypeTag(c)">{{ c.type }}</el-tag>
                <el-tag size="mini" :type="condStatusTag(c)">{{
                  c.status
                }}</el-tag>
                <span class="mono reason">{{ c.reason }}</span>
                <span class="time">changed: {{ c.changedAt }}</span>
                <span class="time">hb: {{ c.heartbeat }}</span>
              </div>
              <div class="cond-msg mono">{{ c.message }}</div>
            </div>
          </div>
          <div v-else class="muted">无</div>
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
          <div v-else class="muted">无</div>
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
  name: 'NodeDetailDrawer',
  props: {
    visible: { type: Boolean, default: false },
    node: { type: Object, required: true },
    width: { type: String, default: '45%' }
  },
  data() {
    return { activeSection: 'overview' }
  },
  computed: {
    prettyJSON() {
      try {
        return JSON.stringify(this.node, null, 2)
      } catch (e) {
        return '{}'
      }
    },
    readyTagType() {
      return this.node.ready ? 'success' : 'danger'
    },
    schedTagType() {
      return this.node.schedulable ? 'success' : 'warning'
    },
    cpuPercentStr() {
      if (typeof this.node.cpuUtilPct === 'number') { return this.clampPct(this.node.cpuUtilPct).toFixed(1) + '%' }
      const u = Number(this.node.cpuUsageCores || 0)
      const a = Number(this.node.cpuAllocatableCores || 0)
      const pct = a > 0 ? (u / a) * 100 : 0
      return this.clampPct(pct).toFixed(1) + '%'
    },
    memPercentStr() {
      if (typeof this.node.memUtilPct === 'number') { return this.clampPct(this.node.memUtilPct).toFixed(1) + '%' }
      const u = Number(this.node.memUsageGiB || 0)
      const a = Number(this.node.memAllocatableGiB || 0)
      const pct = a > 0 ? (u / a) * 100 : 0
      return this.clampPct(pct).toFixed(1) + '%'
    },
    podsPercentStr() {
      if (typeof this.node.podsUtilPct === 'number') { return this.clampPct(this.node.podsUtilPct).toFixed(1) + '%' }
      const u = Number(this.node.podsUsed || 0)
      const a = Number(this.node.podsAllocatable || 0)
      const pct = a > 0 ? (u / a) * 100 : 0
      return this.clampPct(pct).toFixed(1) + '%'
    },
    labelArray() {
      const obj = this.node.labels || {}
      return Object.keys(obj).map((k) => ({ k, v: obj[k] }))
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

    // helpers
    boolStr(v) {
      return v ? 'True' : 'False'
    },
    clampPct(v) {
      return Math.max(0, Math.min(100, Number(v) || 0))
    },
    fmtCores(v) {
      const n = Number(v)
      if (!Number.isFinite(n)) return '-'
      return n.toFixed(n < 10 ? 3 : 0).replace(/\.?0+$/, '') + ' cores'
    },
    fmtGiB(v) {
      const n = Number(v)
      if (!Number.isFinite(n)) return '-'
      return n.toFixed(n < 10 ? 3 : 1).replace(/\.?0+$/, '') + ' GiB'
    },
    condTypeTag(c) {
      // 仅做区分：Ready 用 primary，其它 info
      return (c.type || '').toLowerCase() === 'ready' ? 'primary' : 'info'
    },
    condStatusTag(c) {
      const s = (c.status || '').toLowerCase()
      if (s === 'true') return 'success'
      if (s === 'false') return 'danger'
      return 'warning'
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
        'resource',
        'capacity',
        'network',
        'components',
        'conditions',
        'labels',
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
.node-describe-drawer {
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

.progress-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 6px 0 12px;
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
  min-width: 48px;
  text-align: right;
  color: #666;
}

.cond-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.cond-item {
  padding: 10px 12px;
  border: 1px solid #f1f1f1;
  border-radius: 8px;
  background: #fff;
}
.cond-head {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
}
.cond-head .reason {
  color: #666;
}
.cond-head .time {
  color: #999;
  font-size: 12px;
  margin-left: 6px;
}
.cond-msg {
  margin-top: 6px;
  color: #555;
}
.labels {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
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
