<template>
  <el-drawer
    :visible="visible"
    :size="width"
    :with-header="false"
    custom-class="svc-describe-drawer"
    append-to-body
    :destroy-on-close="true"
    :close-on-click-modal="true"
    @update:visible="$emit('update:visible', $event)"
    :before-close="handleBeforeClose"
    @close="handleClose"
  >
    <!-- 顶部摘要栏（吸顶） -->
    <div class="summary-bar">
      <div class="left">
        <span class="svc-name">{{ svc.name }}</span>
        <el-tag size="mini" type="info">{{ svc.namespace }}</el-tag>
        <el-tag size="mini" :type="typeTagType">{{
          svc.type || "ClusterIP"
        }}</el-tag>
        <span class="age">Age {{ svc.age || "-" }}</span>
        <span class="age">Created {{ svc.createdAt || "-" }}</span>
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
          <el-menu-item index="selector">选择器</el-menu-item>
          <el-menu-item index="ports">端口</el-menu-item>
          <el-menu-item index="backends">后端（Endpoints）</el-menu-item>
          <el-menu-item index="network">网络 / IP</el-menu-item>
          <el-menu-item index="raw">原始（JSON）</el-menu-item>
        </el-menu>
      </div>

      <!-- 右：内容（可滚） -->
      <div class="content" ref="scrollEl" @scroll="onScroll">
        <!-- 概览 -->
        <section ref="overview" data-id="overview" class="section">
          <h3 class="section-title">概览</h3>
          <div class="kv">
            <div>
              <span>名称</span><b>{{ svc.name }}</b>
            </div>
            <div>
              <span>命名空间</span><b>{{ svc.namespace }}</b>
            </div>
            <div>
              <span>类型</span><b>{{ svc.type || "ClusterIP" }}</b>
            </div>
            <div>
              <span>创建时间</span><b>{{ svc.createdAt || "-" }}</b>
            </div>
            <div>
              <span>存活时长</span><b>{{ svc.age || "-" }}</b>
            </div>
            <div>
              <span>会话亲和性</span><b>{{ svc.sessionAffinity || "None" }}</b>
            </div>
            <div>
              <span>内部流量策略</span
              ><b>{{ svc.internalTrafficPolicy || "-" }}</b>
            </div>
            <div>
              <span>IP Family Policy</span
              ><b>{{ svc.ipFamilyPolicy || "-" }}</b>
            </div>
          </div>
        </section>

        <!-- 选择器 -->
        <section ref="selector" data-id="selector" class="section">
          <h3 class="section-title">选择器</h3>
          <div v-if="selectorArray.length" class="labels">
            <el-tag
              v-for="(kv, i) in selectorArray"
              :key="i"
              size="mini"
              class="mr8 mono"
            >
              {{ kv.k }}={{ kv.v }}
            </el-tag>
          </div>
          <div v-else class="muted">—</div>
        </section>

        <!-- 端口 -->
        <section ref="ports" data-id="ports" class="section">
          <h3 class="section-title">端口</h3>
          <div v-if="portsArray.length" class="kv">
            <div v-for="(p, i) in portsArray" :key="i">
              <span>{{ p.name || "—" }}</span>
              <b class="mono"
                >{{ p.protocol || "TCP" }} {{ p.port }} → {{ p.targetPort }}</b
              >
            </div>
          </div>
          <div v-else class="muted">无</div>
        </section>

        <!-- 后端（Endpoints/EndpointSlice 汇总） -->
        <section ref="backends" data-id="backends" class="section">
          <h3 class="section-title">后端（Endpoints）</h3>

          <template v-if="svc.backends">
            <div class="kv">
              <div>
                <span>Ready/Total</span
                ><b
                  >{{ svc.backends.ready || 0 }} /
                  {{ svc.backends.total || 0 }}</b
                >
              </div>
              <div class="progress-row">
                <div class="bar">
                  <div
                    class="bar-inner"
                    :style="{ width: readyPercentStr }"
                  ></div>
                </div>
                <div class="val">{{ readyPercentStr }}</div>
              </div>
              <div>
                <span>EndpointSlices</span><b>{{ svc.backends.slices || 0 }}</b>
              </div>
              <div>
                <span>最近更新</span><b>{{ svc.backends.updated || "-" }}</b>
              </div>
            </div>

            <h4 class="sub">端口</h4>
            <div v-if="backendPorts.length" class="kv">
              <div v-for="(bp, i) in backendPorts" :key="i">
                <span>{{ bp.name || "—" }}</span>
                <b class="mono">{{ bp.protocol || "TCP" }} {{ bp.port }}</b>
              </div>
            </div>
            <div v-else class="muted">无</div>

            <h4 class="sub">Endpoints</h4>
            <div v-if="backendEndpoints.length" class="ep-list">
              <div v-for="(ep, i) in backendEndpoints" :key="i" class="ep-item">
                <el-tag size="mini" :type="ep.ready ? 'success' : 'info'">{{
                  ep.ready ? "Ready" : "NotReady"
                }}</el-tag>
                <span class="mono addr">{{ ep.address }}</span>
                <span class="node mono">node={{ ep.nodeName || "-" }}</span>
                <span class="tref mono" v-if="ep.targetRef">
                  {{ ep.targetRef.kind }}/{{ ep.targetRef.namespace }}/{{
                    ep.targetRef.name
                  }}
                </span>
              </div>
            </div>
            <div v-else class="muted">无</div>
          </template>
          <div v-else class="muted">无</div>
        </section>

        <!-- 网络 / IP -->
        <section ref="network" data-id="network" class="section">
          <h3 class="section-title">网络 / IP</h3>
          <div class="kv">
            <div>
              <span>ClusterIPs</span>
              <b>
                <template v-if="clusterIPs.length">
                  <el-tag
                    v-for="(ip, i) in clusterIPs"
                    :key="i"
                    size="mini"
                    class="mr8 mono"
                    >{{ ip }}</el-tag
                  >
                </template>
                <template v-else>—</template>
              </b>
            </div>
            <div>
              <span>IP Families</span>
              <b>
                <template v-if="ipFamilies.length">
                  <el-tag
                    v-for="(fam, i) in ipFamilies"
                    :key="i"
                    size="mini"
                    class="mr8 mono"
                    >{{ fam }}</el-tag
                  >
                </template>
                <template v-else>—</template>
              </b>
            </div>
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
  name: "ServiceDetailDrawer",
  props: {
    visible: { type: Boolean, default: false },
    svc: { type: Object, required: true },
    width: { type: String, default: "45%" },
  },
  data() {
    return { activeSection: "overview" };
  },
  computed: {
    typeTagType() {
      const t = (this.svc.type || "ClusterIP").toLowerCase();
      if (t === "clusterip") return "primary";
      if (t === "nodeport") return "warning";
      if (t === "loadbalancer") return "success";
      if (t === "externalname") return "info";
      return "info";
    },
    prettyJSON() {
      try {
        return JSON.stringify(this.svc, null, 2);
      } catch {
        return "{}";
      }
    },
    selectorArray() {
      const obj = this.svc.selector || {};
      return Object.keys(obj).map((k) => ({ k, v: obj[k] }));
    },
    portsArray() {
      return Array.isArray(this.svc.ports) ? this.svc.ports : [];
    },
    clusterIPs() {
      return Array.isArray(this.svc.clusterIPs) ? this.svc.clusterIPs : [];
    },
    ipFamilies() {
      return Array.isArray(this.svc.ipFamilies) ? this.svc.ipFamilies : [];
    },
    backendPorts() {
      const p = this.svc.backends && this.svc.backends.ports;
      return Array.isArray(p) ? p : [];
    },
    backendEndpoints() {
      const e = this.svc.backends && this.svc.backends.endpoints;
      return Array.isArray(e) ? e : [];
    },
    readyPercentStr() {
      const r = Number(this.svc.backends?.ready || 0);
      const t = Number(this.svc.backends?.total || 0);
      const pct = t > 0 ? (r / t) * 100 : 0;
      return this.clampPct(pct).toFixed(0) + "%";
    },
  },
  methods: {
    handleBeforeClose(done) {
      this.$emit("update:visible", false);
      done && done();
    },
    handleClose() {
      this.$emit("update:visible", false);
    },
    clampPct(v) {
      return Math.max(0, Math.min(100, Number(v) || 0));
    },

    // 目录滚动
    scrollTo(id) {
      const el = this.$refs[id];
      if (!el || !this.$refs.scrollEl) return;
      const top = el.offsetTop - 8;
      this.$refs.scrollEl.scrollTo({ top, behavior: "smooth" });
      this.activeSection = id;
      this.$emit("section-change", id);
    },
    onScroll() {
      const container = this.$refs.scrollEl;
      if (!container) return;
      const sections = [
        "overview",
        "selector",
        "ports",
        "backends",
        "network",
        "raw",
      ];
      let current = sections[0];
      for (const id of sections) {
        const el = this.$refs[id];
        if (el && el.offsetTop - container.scrollTop <= 40) current = id;
      }
      this.activeSection = current;
    },
  },
};
</script>

<style scoped>
.svc-describe-drawer {
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
.summary-bar .svc-name {
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
.ep-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.ep-item {
  padding: 8px 10px;
  border: 1px solid #f1f1f1;
  border-radius: 8px;
  background: #fff;
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  align-items: center;
}
.ep-item .addr {
  min-width: 140px;
}
.ep-item .node {
  color: #666;
}
.ep-item .tref {
  color: #555;
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
  min-width: 44px;
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
