<template>
  <el-drawer
    :visible="visible"
    :size="width"
    :with-header="false"
    custom-class="ing-describe-drawer"
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
        <span class="ing-name">{{ ing.name }}</span>
        <el-tag size="mini" type="info">{{ ing.namespace }}</el-tag>
        <el-tag v-if="ing.tlsEnabled" size="mini" type="success">TLS</el-tag>
        <el-tag v-else size="mini" type="warning">No TLS</el-tag>
        <el-tag size="mini" type="info">Class {{ ing.class || "-" }}</el-tag>
        <el-tag
          size="mini"
          type="info"
        >Controller {{ ing.controller || "-" }}</el-tag>
        <span class="age">Created {{ ing.createdAt || "-" }}</span>
        <span class="age">Age {{ ing.age || "-" }}</span>
        <span v-if="lbIPsStr" class="age">LB {{ lbIPsStr }}</span>
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
          <el-menu-item index="hosts">主机/域名</el-menu-item>
          <el-menu-item index="rules">路由规则</el-menu-item>
          <el-menu-item index="tls">TLS</el-menu-item>
          <el-menu-item index="status">状态/LB</el-menu-item>
          <el-menu-item index="annotations">注解</el-menu-item>
          <el-menu-item index="raw">原始（JSON）</el-menu-item>
        </el-menu>
      </div>

      <!-- 右：内容 -->
      <div ref="scrollEl" class="content" @scroll="onScroll">
        <!-- 概览 -->
        <section ref="overview" data-id="overview" class="section">
          <h3 class="section-title">概览</h3>
          <div class="kv">
            <div>
              <span>名称</span><b>{{ ing.name }}</b>
            </div>
            <div>
              <span>命名空间</span><b>{{ ing.namespace }}</b>
            </div>
            <div>
              <span>Class</span><b>{{ ing.class || "-" }}</b>
            </div>
            <div>
              <span>Controller</span><b>{{ ing.controller || "-" }}</b>
            </div>
            <div>
              <span>TLS</span><b>{{ ing.tlsEnabled ? "Enabled" : "Disabled" }}</b>
            </div>
            <div>
              <span>创建时间</span><b>{{ ing.createdAt || "-" }}</b>
            </div>
            <div>
              <span>存活时长</span><b>{{ ing.age || "-" }}</b>
            </div>
          </div>
        </section>

        <!-- 主机/域名 -->
        <section ref="hosts" data-id="hosts" class="section">
          <h3 class="section-title">主机/域名</h3>
          <div v-if="hostList.length">
            <el-tag
              v-for="(h, i) in hostList"
              :key="i"
              size="mini"
              class="mr8 mono"
            >{{ h }}</el-tag>
          </div>
          <div v-else class="muted">—</div>
        </section>

        <!-- 路由规则 -->
        <section ref="rules" data-id="rules" class="section">
          <h3 class="section-title">路由规则</h3>

          <div v-if="specRules.length === 0" class="muted">无规则</div>
          <div
            v-for="(r, ri) in specRules"
            :key="'rule-' + ri"
            class="rule-block"
          >
            <div class="rule-head">
              <b>Host:</b> <span class="mono">{{ r.host || "-" }}</span>
            </div>

            <div v-if="(r.paths || []).length === 0" class="muted">无路径</div>
            <div
              v-for="(p, pi) in r.paths"
              :key="'path-' + ri + '-' + pi"
              class="path-row"
            >
              <div class="kv">
                <div>
                  <span>Path</span><b class="mono">{{ p.path || "/" }}</b>
                </div>
                <div>
                  <span>PathType</span><b>{{ p.pathType || "-" }}</b>
                </div>
                <div>
                  <span>Backend</span><b>{{ backendTypeStr(p.backend) }}</b>
                </div>
                <div v-if="p.backend && p.backend.service">
                  <span>Service</span>
                  <b class="mono">
                    {{ p.backend.service.name || "-" }} :
                    {{
                      p.backend.service.portNumber != null
                        ? p.backend.service.portNumber
                        : "-"
                    }}
                  </b>
                </div>
              </div>
            </div>
          </div>
        </section>

        <!-- TLS -->
        <section ref="tls" data-id="tls" class="section">
          <h3 class="section-title">TLS</h3>
          <div v-if="specTLS.length === 0" class="muted">未配置</div>
          <div v-for="(t, ti) in specTLS" :key="'tls-' + ti" class="tls-block">
            <div class="kv">
              <div>
                <span>Secret</span><b class="mono">{{ t.secretName || "-" }}</b>
              </div>
              <div>
                <span>Hosts</span>
                <b>
                  <el-tag
                    v-for="(h, i) in t.hosts || []"
                    :key="'tlsh-' + ti + '-' + i"
                    size="mini"
                    class="mr8 mono"
                  >{{ h }}</el-tag>
                  <template v-if="!t.hosts || t.hosts.length === 0">—</template>
                </b>
              </div>
            </div>
          </div>
        </section>

        <!-- 状态/LB -->
        <section ref="status" data-id="status" class="section">
          <h3 class="section-title">状态 / LoadBalancer</h3>
          <div class="kv">
            <div>
              <span>LoadBalancer IPs</span>
              <b>
                <template v-if="statusLB.length">
                  <el-tag
                    v-for="(ip, i) in statusLB"
                    :key="'lb-' + i"
                    size="mini"
                    class="mr8 mono"
                  >{{ ip }}</el-tag>
                </template>
                <template v-else>—</template>
              </b>
            </div>
          </div>
        </section>

        <!-- 注解 -->
        <section ref="annotations" data-id="annotations" class="section">
          <h3 class="section-title">注解</h3>
          <div v-if="annotationArray.length" class="kv">
            <div v-for="(a, i) in annotationArray" :key="'anno-' + i">
              <span class="mono">{{ a.k }}</span><b class="mono">{{ a.v }}</b>
            </div>
          </div>
          <div v-else class="muted">—</div>
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
  name: 'IngressDetailDrawer',
  props: {
    visible: { type: Boolean, default: false },
    ing: { type: Object, required: true },
    width: { type: String, default: '50%' }
  },
  data() {
    return { activeSection: 'overview' }
  },
  computed: {
    lbIPsStr() {
      const arr = Array.isArray(this.ing.loadBalancer)
        ? this.ing.loadBalancer
        : []
      return arr.join(', ')
    },
    hostList() {
      // 优先用 data.hosts；若为空，尝试从 spec.rules 提取 host
      const top = Array.isArray(this.ing.hosts) ? this.ing.hosts : []
      if (top.length) return top
      const rules =
        this.ing.spec && Array.isArray(this.ing.spec.rules)
          ? this.ing.spec.rules
          : []
      const set = new Set()
      rules.forEach((r) => r && r.host && set.add(r.host))
      return Array.from(set)
    },
    specRules() {
      const s = this.ing.spec
      return s && Array.isArray(s.rules) ? s.rules : []
    },
    specTLS() {
      const s = this.ing.spec
      return s && Array.isArray(s.tls) ? s.tls : []
    },
    statusLB() {
      // 兼容 data.status.loadBalancer: ["ip"]
      if (this.ing.status && Array.isArray(this.ing.status.loadBalancer)) { return this.ing.status.loadBalancer }
      const top = Array.isArray(this.ing.loadBalancer)
        ? this.ing.loadBalancer
        : []
      return top
    },
    annotationArray() {
      const obj = this.ing.annotations || {}
      return Object.keys(obj).map((k) => ({ k, v: obj[k] }))
    },
    prettyJSON() {
      try {
        return JSON.stringify(this.ing, null, 2)
      } catch (e) {
        return '{}'
      }
    }
  },
  methods: {
    backendTypeStr(b) {
      if (!b) return '-'
      if (b.type) return b.type
      if (b.service) return 'Service'
      return 'Backend'
    },
    handleBeforeClose(done) {
      this.$emit('update:visible', false)
      done && done()
    },
    handleClose() {
      this.$emit('update:visible', false)
    },
    // 目录滚动 & scrollspy
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
        'hosts',
        'rules',
        'tls',
        'status',
        'annotations',
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
.ing-describe-drawer {
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
.summary-bar .ing-name {
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

.rule-block {
  padding: 8px 10px;
  border: 1px solid #f0f0f0;
  border-radius: 8px;
  margin-bottom: 10px;
  background: #fff;
}
.rule-head {
  margin-bottom: 6px;
}
.path-row {
  padding: 6px 0 2px;
}
.tls-block {
  padding: 8px 10px;
  border: 1px dashed #eaeaea;
  border-radius: 8px;
  margin-bottom: 10px;
  background: #fff;
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
