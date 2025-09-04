<template>
  <el-drawer
    :visible="visible"
    :size="width"
    :with-header="false"
    custom-class="pod-describe-drawer"
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
        <span class="pod-name">{{ pod.name }}</span>
        <el-tag size="mini" type="info">{{ pod.namespace }}</el-tag>
        <el-tag size="mini" :type="phaseTagType">{{ pod.phase }}</el-tag>
        <el-tag size="mini">Ready {{ pod.ready }}</el-tag>
        <el-tag size="mini">Restarts {{ pod.restarts }}</el-tag>
        <el-tag size="mini">QoS {{ pod.qosClass }}</el-tag>
        <el-tag size="mini">Node {{ pod.node }}</el-tag>
        <span class="age">Age {{ pod.age }}</span>
      </div>
      <!-- 右侧操作区已移除 -->
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
          <el-menu-item index="containers">容器</el-menu-item>
          <el-menu-item index="network">网络</el-menu-item>
          <el-menu-item index="scheduling">调度</el-menu-item>
          <el-menu-item index="storage">存储</el-menu-item>
          <el-menu-item index="account">账户与策略</el-menu-item>
          <el-menu-item index="raw">原始（JSON/YAML）</el-menu-item>
        </el-menu>
      </div>

      <!-- 右：内容（可滚） -->
      <div class="content" ref="scrollEl" @scroll="onScroll">
        <!-- 概览 -->
        <section ref="overview" data-id="overview" class="section">
          <h3 class="section-title">概览</h3>
          <div class="kv">
            <div>
              <span>名称</span><b>{{ pod.name }}</b>
            </div>
            <div>
              <span>命名空间</span><b>{{ pod.namespace }}</b>
            </div>
            <div>
              <span>控制器</span><b>{{ pod.controller }}</b>
            </div>
            <div>
              <span>阶段</span><b>{{ pod.phase }}</b>
            </div>
            <div>
              <span>就绪</span><b>{{ pod.ready }}</b>
            </div>
            <div>
              <span>重启次数</span><b>{{ pod.restarts }}</b>
            </div>
            <div>
              <span>启动时间</span><b>{{ pod.startTime }}</b>
            </div>
            <div>
              <span>存活时长</span><b>{{ pod.age }}</b>
            </div>
            <div>
              <span>节点</span><b>{{ pod.node }}</b>
            </div>
            <div>
              <span>QoS</span><b>{{ pod.qosClass }}</b>
            </div>
          </div>
        </section>

        <!-- 资源与用量 -->
        <section ref="resource" data-id="resource" class="section">
          <h3 class="section-title">资源与用量</h3>
          <div class="kv">
            <div>
              <span>CPU 使用 / 限制</span>
              <b>{{ pod.cpuUsage }} / {{ normalizedCpuLimit }}</b>
            </div>
            <div class="progress-row">
              <div class="bar">
                <div class="bar-inner" :style="{ width: cpuPercentStr }"></div>
              </div>
              <div class="val">{{ cpuPercentStr }}</div>
            </div>
            <div>
              <span>内存 使用 / 限制</span>
              <b>{{ pod.memUsage }} / {{ pod.memLimit }}</b>
            </div>
            <div class="progress-row">
              <div class="bar">
                <div class="bar-inner" :style="{ width: memPercentStr }"></div>
              </div>
              <div class="val">{{ memPercentStr }}</div>
            </div>
            <div>
              <span>内存利用率</span><b>{{ memPercentStr }}</b>
            </div>
          </div>
        </section>

        <!-- 容器（同一页多个容器分块） -->
        <section ref="containers" data-id="containers" class="section">
          <h3 class="section-title">容器</h3>

          <div v-if="(pod.containers || []).length === 0" class="muted">
            无容器
          </div>
          <div
            v-for="(c, idx) in pod.containers || []"
            :key="idx"
            class="container-block"
          >
            <div class="container-title">
              <b>{{ c.name }}</b>
              <span class="mono">· {{ c.image }}</span>
              <el-tag
                size="mini"
                :type="c.state === 'Running' ? 'success' : 'info'"
              >
                {{ c.state || "-" }}
              </el-tag>
            </div>

            <h4 class="sub">基本</h4>
            <div class="kv">
              <div>
                <span>名称</span><b>{{ c.name }}</b>
              </div>
              <div>
                <span>镜像</span><b class="mono">{{ c.image }}</b>
              </div>
              <div>
                <span>状态</span><b>{{ c.state }}</b>
              </div>
              <div>
                <span>拉取策略</span><b>{{ c.imagePullPolicy }}</b>
              </div>
            </div>

            <h4 class="sub">端口</h4>
            <div v-if="c.ports && c.ports.length">
              <el-tag
                v-for="(p, i) in c.ports"
                :key="i"
                size="mini"
                class="mr8"
              >
                {{ p.containerPort }}/{{ p.protocol }}
              </el-tag>
            </div>
            <div v-else class="muted">无</div>

            <h4 class="sub">资源</h4>
            <div class="kv">
              <div>
                <span>Requests</span>
                <b
                  >CPU {{ (c.requests && c.requests.cpu) || "-" }}, 内存
                  {{ (c.requests && c.requests.memory) || "-" }}</b
                >
              </div>
              <div>
                <span>Limits</span>
                <b
                  >CPU {{ (c.limits && c.limits.cpu) || "-" }}, 内存
                  {{ (c.limits && c.limits.memory) || "-" }}</b
                >
              </div>
            </div>

            <h4 class="sub">探针</h4>
            <div class="kv">
              <div>
                <span>就绪探针</span>
                <b v-if="c.readinessProbe">
                  HTTP {{ c.readinessProbe.httpGet.scheme }}
                  {{ c.readinessProbe.httpGet.path }} :{{
                    c.readinessProbe.httpGet.port
                  }}
                  （init {{ c.readinessProbe.initialDelaySeconds }}s, period
                  {{ c.readinessProbe.periodSeconds }}s）
                </b>
                <b v-else class="muted">无</b>
              </div>
              <div>
                <span>存活探针</span>
                <b v-if="c.livenessProbe">
                  HTTP {{ c.livenessProbe.httpGet.scheme }}
                  {{ c.livenessProbe.httpGet.path }} :{{
                    c.livenessProbe.httpGet.port
                  }}
                  （init {{ c.livenessProbe.initialDelaySeconds }}s, period
                  {{ c.livenessProbe.periodSeconds }}s）
                </b>
                <b v-else class="muted">无</b>
              </div>
            </div>

            <h4 class="sub">环境变量</h4>
            <div v-if="c.envs && c.envs.length" class="kv">
              <div v-for="(e, i) in c.envs" :key="i">
                <span>{{ e.name }}</span
                ><b class="mono">{{ e.value }}</b>
              </div>
            </div>
            <div v-else class="muted">无</div>

            <h4 class="sub">卷挂载</h4>
            <div v-if="c.volumeMounts && c.volumeMounts.length" class="kv">
              <div v-for="(vm, i) in c.volumeMounts" :key="i">
                <span>{{ vm.name }}</span>
                <b
                  >{{ vm.mountPath }}
                  <i class="muted"
                    >（只读：{{ String(vm.readOnly || false) }}）</i
                  ></b
                >
              </div>
            </div>
            <div v-else class="muted">无</div>
          </div>
        </section>

        <!-- 网络 -->
        <section ref="network" data-id="network" class="section">
          <h3 class="section-title">网络</h3>
          <div class="kv">
            <div>
              <span>HostNetwork</span><b>{{ String(pod.hostNetwork) }}</b>
            </div>
            <div>
              <span>DNS 策略</span><b>{{ pod.dnsPolicy }}</b>
            </div>
            <div>
              <span>PodIP</span><b>{{ pod.podIP }}</b>
            </div>
            <div>
              <span>PodIPs</span>
              <b>
                <template v-if="pod.podIPs && pod.podIPs.length">
                  <el-tag
                    v-for="(ip, i) in pod.podIPs"
                    :key="i"
                    size="mini"
                    class="mr8"
                    >{{ ip }}</el-tag
                  >
                </template>
                <template v-else>-</template>
              </b>
            </div>
            <div>
              <span>HostIP</span><b>{{ pod.hostIP }}</b>
            </div>
          </div>
        </section>

        <!-- 调度 -->
        <section ref="scheduling" data-id="scheduling" class="section">
          <h3 class="section-title">调度</h3>
          <h4 class="sub">Tolerations</h4>
          <div v-if="pod.tolerations && pod.tolerations.length" class="kv">
            <div v-for="(t, i) in pod.tolerations" :key="i">
              <span>{{ t.key }}</span>
              <b
                >op={{ t.operator }}；effect={{ t.effect }}；seconds={{
                  t.tolerationSeconds || "-"
                }}</b
              >
            </div>
          </div>
          <div v-else class="muted">无</div>

          <h4 class="sub">Pod Anti-Affinity（preferred）</h4>
          <div v-if="antiPreferred.length" class="kv">
            <div v-for="(p, i) in antiPreferred" :key="i">
              <span>weight={{ p.weight }}</span>
              <b>
                labels：<span class="mono">{{
                  labelSelectorStr(
                    p.podAffinityTerm && p.podAffinityTerm.labelSelector
                  )
                }}</span
                >， topologyKey：{{
                  p.podAffinityTerm && p.podAffinityTerm.topologyKey
                }}
              </b>
            </div>
          </div>
          <div v-else class="muted">无</div>
        </section>

        <!-- 存储 -->
        <section ref="storage" data-id="storage" class="section">
          <h3 class="section-title">存储</h3>
          <div v-if="pod.volumes && pod.volumes.length" class="kv">
            <div v-for="(v, i) in pod.volumes" :key="i">
              <span>{{ v.name }}（{{ v.type }}）</span>
              <b>
                <template v-if="v.type === 'nfs' && v.sourceRaw">
                  server={{ v.sourceRaw.server }}，path={{ v.sourceRaw.path }}
                </template>
                <template v-else-if="v.sourceBrief">{{
                  v.sourceBrief
                }}</template>
                <template v-else>—</template>
              </b>
            </div>
          </div>
          <div v-else class="muted">无</div>
        </section>

        <!-- 账户与策略 -->
        <section ref="account" data-id="account" class="section">
          <h3 class="section-title">账户与策略</h3>
          <div class="kv">
            <div>
              <span>ServiceAccount</span><b>{{ pod.serviceAccountName }}</b>
            </div>
            <div>
              <span>RestartPolicy</span><b>{{ pod.restartPolicy }}</b>
            </div>
            <div>
              <span>优雅终止（秒）</span
              ><b>{{ pod.terminationGracePeriodSeconds }}</b>
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
  name: "PodDetailDrawer",
  props: {
    visible: { type: Boolean, default: false },
    pod: { type: Object, required: true },
    width: { type: String, default: "45%" }, // 右侧检查器：45%；想要更宽可传 60%/70%
  },
  data() {
    return { activeSection: "overview" };
  },
  computed: {
    phaseTagType() {
      const p = (this.pod.phase || "").toLowerCase();
      if (p === "running") return "success";
      if (p === "pending") return "warning";
      if (p === "failed") return "danger";
      return "info";
    },
    prettyJSON() {
      try {
        return JSON.stringify(this.pod, null, 2);
      } catch (e) {
        return "{}";
      }
    },
    antiPreferred() {
      return (
        ((this.pod.affinity || {}).podAntiAffinity || {})
          .preferredDuringSchedulingIgnoredDuringExecution || []
      );
    },
    normalizedCpuLimit() {
      const lim = String(this.pod.cpuLimit || "")
        .trim()
        .toLowerCase();
      if (lim === "700") return "700m"; // 友好显示
      if (lim === "1k" || lim === "1000" || lim === "1000m") return "1000m";
      return this.pod.cpuLimit || "-";
    },
    cpuPercentStr() {
      const u = this.parseCpuToMilli(this.pod.cpuUsage);
      const l = this.parseCpuToMilli(this.normalizedCpuLimit);
      if (!u || !l) return "0%";
      const pct = Math.max(0, Math.min(100, (u / l) * 100));
      return pct.toFixed(0) + "%";
    },
    memPercentStr() {
      if (typeof this.pod.memUtilPct === "number") {
        return Math.max(0, Math.min(100, this.pod.memUtilPct)).toFixed(1) + "%";
      }
      const u = this.parseBytes(this.pod.memUsage);
      const l = this.parseBytes(this.pod.memLimit);
      if (!u || !l) return "0%";
      const pct = Math.max(0, Math.min(100, (u / l) * 100));
      return pct.toFixed(1) + "%";
    },
  },
  methods: {
    // 让 el-drawer 关闭行为同步父级的 :visible.sync
    handleBeforeClose(done) {
      this.$emit("update:visible", false);
      done && done();
    },
    handleClose() {
      this.$emit("update:visible", false);
    },

    labelSelectorStr(sel) {
      if (!sel || !sel.matchLabels) return "-";
      return Object.entries(sel.matchLabels)
        .map(([k, v]) => `${k}=${v}`)
        .join(", ");
    },
    parseCpuToMilli(v) {
      if (v == null) return 0;
      const s = String(v).trim().toLowerCase();
      if (s.endsWith("m")) return parseFloat(s.slice(0, -1)) || 0;
      if (s === "1k" || s === "1000" || s === "1000m") return 1000;
      const num = parseFloat(s);
      return isNaN(num) ? 0 : num * 1000; // 无单位视为核
    },
    parseBytes(v) {
      if (!v) return 0;
      const s = String(v).trim().toLowerCase();
      const map = { ki: 1024, mi: 1024 ** 2, gi: 1024 ** 3, ti: 1024 ** 4 };
      const m = s.match(/^([\d.]+)\s*(ki|mi|gi|ti|k|m|g|t)?$/i);
      if (!m) return parseFloat(s) || 0;
      const val = parseFloat(m[1]);
      const unit = (m[2] || "").toLowerCase();
      if (!unit) return val;
      if (unit === "k") return val * 1000;
      if (unit === "m") return val * 1000 ** 2;
      if (unit === "g") return val * 1000 ** 3;
      if (unit === "t") return val * 1000 ** 4;
      return val * (map[unit] || 1);
    },
    scrollTo(id) {
      const el = this.$refs[id];
      if (!el || !this.$refs.scrollEl) return;
      const top = el.offsetTop - 8; // 上方留空
      this.$refs.scrollEl.scrollTo({ top, behavior: "smooth" });
      this.activeSection = id;
      this.$emit("section-change", id);
    },
    onScroll() {
      const container = this.$refs.scrollEl;
      if (!container) return;
      const sections = [
        "overview",
        "resource",
        "containers",
        "network",
        "scheduling",
        "storage",
        "account",
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
.pod-describe-drawer {
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
.summary-bar .pod-name {
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
.sub {
  margin: 14px 0 6px;
  font-weight: 600;
  color: #555;
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

.container-block {
  padding: 10px 12px;
  border: 1px solid #f1f1f1;
  border-radius: 8px;
  margin-bottom: 12px;
  background: #fff;
}
.container-title {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 6px;
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
