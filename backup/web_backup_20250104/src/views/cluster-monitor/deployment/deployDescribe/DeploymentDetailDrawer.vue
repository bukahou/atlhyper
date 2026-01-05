<template>
  <el-drawer
    :visible="visible"
    :size="width"
    :with-header="false"
    custom-class="dep-describe-drawer"
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
        <span class="dep-name">{{ dep.name }}</span>
        <el-tag size="mini" type="info">{{ dep.namespace }}</el-tag>
        <el-tag size="mini" type="success">{{
          dep.strategy || spec.strategyType || "RollingUpdate"
        }}</el-tag>
        <el-tag size="mini">
          Replicas {{ dep.replicas }} (ready {{ dep.ready }} / updated
          {{ dep.updated }} / available {{ dep.available }})
        </el-tag>
        <span class="age">Created {{ dep.createdAt || "-" }}</span>
        <span class="age">Age {{ dep.age || "-" }}</span>
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
          <el-menu-item index="replicas">副本与状态</el-menu-item>
          <el-menu-item index="strategy">更新策略</el-menu-item>
          <el-menu-item index="selectors">Selector / 标签</el-menu-item>
          <el-menu-item index="template">Pod 模板（容器）</el-menu-item>
          <el-menu-item index="replicasets">历史 ReplicaSets</el-menu-item>
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
              <span>名称</span><b>{{ dep.name }}</b>
            </div>
            <div>
              <span>命名空间</span><b>{{ dep.namespace }}</b>
            </div>
            <div>
              <span>策略</span><b>{{ dep.strategy || spec.strategyType || "-" }}</b>
            </div>
            <div>
              <span>Selector</span><b class="mono">{{ dep.selector || selectorStr }}</b>
            </div>
            <div>
              <span>创建时间</span><b>{{ dep.createdAt || "-" }}</b>
            </div>
            <div>
              <span>存活时长</span><b>{{ dep.age || "-" }}</b>
            </div>
          </div>
        </section>

        <!-- 副本与状态 -->
        <section ref="replicas" data-id="replicas" class="section">
          <h3 class="section-title">副本与状态</h3>
          <div class="kv">
            <div>
              <span>期望副本</span>
              <b>{{
                spec.replicas != null
                  ? spec.replicas
                  : dep.replicas != null
                    ? dep.replicas
                    : "-"
              }}</b>
            </div>
            <div>
              <span>状态副本</span>
              <b>
                ready {{ (status && status.readyReplicas) || 0 }} / updated
                {{ (status && status.updatedReplicas) || 0 }} / available
                {{ (status && status.availableReplicas) || 0 }} / total
                {{ (status && status.replicas) || 0 }}
              </b>
            </div>
            <div>
              <span>Rollout 阶段</span><b>{{ (dep.rollout && dep.rollout.phase) || "—" }}</b>
            </div>
          </div>

          <h4 class="sub">Conditions</h4>
          <div v-if="(dep.conditions || []).length" class="kv">
            <div v-for="(c, i) in dep.conditions" :key="i">
              <span>{{ c.type }} ({{ c.status }})</span>
              <b>
                {{ c.reason || "-" }} — {{ c.message || "-" }}
                <i
                  class="muted"
                >updated {{ c.lastUpdateTime || "-" }}, transition
                  {{ c.lastTransitionTime || "-" }}</i>
              </b>
            </div>
          </div>
          <div v-else class="muted">—</div>
        </section>

        <!-- 更新策略 -->
        <section ref="strategy" data-id="strategy" class="section">
          <h3 class="section-title">更新策略</h3>
          <div class="kv">
            <div>
              <span>Strategy</span><b>{{ spec.strategyType || dep.strategy || "-" }}</b>
            </div>
            <div>
              <span>Max Surge</span><b>{{ spec.maxSurge || "—" }}</b>
            </div>
            <div>
              <span>Max Unavailable</span><b>{{ spec.maxUnavailable || "—" }}</b>
            </div>
            <div>
              <span>RevisionHistoryLimit</span><b>{{
                spec.revisionHistoryLimit != null
                  ? spec.revisionHistoryLimit
                  : "—"
              }}</b>
            </div>
            <div>
              <span>ProgressDeadlineSeconds</span><b>{{
                spec.progressDeadlineSeconds != null
                  ? spec.progressDeadlineSeconds
                  : "—"
              }}</b>
            </div>
          </div>
        </section>

        <!-- Selector / 标签 -->
        <section ref="selectors" data-id="selectors" class="section">
          <h3 class="section-title">Selector / 标签</h3>
          <div class="kv">
            <div>
              <span>MatchLabels</span>
              <b class="mono">
                <template v-if="Object.keys(spec.matchLabels || {}).length">
                  {{ selectorStr }}
                </template>
                <template v-else>—</template>
              </b>
            </div>
            <div>
              <span>Template Labels</span>
              <b>
                <template v-if="Object.keys(tpl.labels || {}).length">
                  <el-tag
                    v-for="(v, k) in tpl.labels"
                    :key="k"
                    size="mini"
                    class="mr8 mono"
                  >{{ k }}={{ v }}</el-tag>
                </template>
                <template v-else>—</template>
              </b>
            </div>
          </div>
        </section>

        <!-- Pod 模板（容器） -->
        <section ref="template" data-id="template" class="section">
          <h3 class="section-title">Pod 模板（容器）</h3>
          <div class="kv">
            <div>
              <span>DNS Policy</span><b>{{ tpl.dnsPolicy || "-" }}</b>
            </div>
            <div>
              <span>ImagePullSecrets</span>
              <b>
                <template v-if="imagePullSecrets.length">
                  <el-tag
                    v-for="(s, i) in imagePullSecrets"
                    :key="i"
                    size="mini"
                    class="mr8 mono"
                  >{{ s }}</el-tag>
                </template>
                <template v-else>—</template>
              </b>
            </div>
          </div>

          <div v-if="containers.length === 0" class="muted">无容器</div>
          <div
            v-for="(c, idx) in containers"
            :key="idx"
            class="container-block"
          >
            <div class="container-title">
              <b>{{ c.name }}</b>
              <span class="mono">· {{ c.image }}</span>
              <el-tag size="mini" type="info">{{
                c.imagePullPolicy || "IfNotPresent"
              }}</el-tag>
            </div>

            <h4 class="sub">端口</h4>
            <div v-if="(c.ports || []).length">
              <el-tag
                v-for="(p, i) in c.ports"
                :key="i"
                size="mini"
                class="mr8"
              >
                {{ p.containerPort }}/{{ p.protocol || "TCP" }}
              </el-tag>
            </div>
            <div v-else class="muted">无</div>

            <h4 class="sub">资源</h4>
            <div class="kv">
              <div>
                <span>Requests</span>
                <b>
                  CPU
                  {{
                    (c.resources &&
                      c.resources.requests &&
                      c.resources.requests.cpu) ||
                      "-"
                  }}, 内存
                  {{
                    (c.resources &&
                      c.resources.requests &&
                      c.resources.requests.memory) ||
                      "-"
                  }}
                </b>
              </div>
              <div>
                <span>Limits</span>
                <b>
                  CPU
                  {{
                    (c.resources &&
                      c.resources.limits &&
                      c.resources.limits.cpu) ||
                      "-"
                  }}, 内存
                  {{
                    (c.resources &&
                      c.resources.limits &&
                      c.resources.limits.memory) ||
                      "-"
                  }}
                </b>
              </div>
            </div>

            <h4 class="sub">环境变量</h4>
            <div v-if="(c.env || []).length" class="kv">
              <div v-for="(e, i) in c.env" :key="i">
                <span class="mono">{{ e.name }}</span><b class="mono">{{ e.value }}</b>
              </div>
            </div>
            <div v-else class="muted">无</div>
          </div>
        </section>

        <!-- 历史 ReplicaSets -->
        <section ref="replicasets" data-id="replicasets" class="section">
          <h3 class="section-title">历史 ReplicaSets</h3>
          <div v-if="(dep.replicaSets || []).length === 0" class="muted">—</div>
          <div v-else class="rs-list">
            <div v-for="(rs, i) in dep.replicaSets" :key="i" class="rs-item">
              <div class="kv">
                <div>
                  <span>名称</span><b class="mono">{{ rs.name }}</b>
                </div>
                <div>
                  <span>Revision</span><b>{{ rs.revision }}</b>
                </div>
                <div>
                  <span>Replicas</span><b>ready {{ rs.ready }}/{{ rs.replicas }}, available
                    {{ rs.available }}</b>
                </div>
                <div>
                  <span>创建时间</span><b>{{ rs.createdAt || "-" }}</b>
                </div>
                <div>
                  <span>Age</span><b>{{ rs.age || "-" }}</b>
                </div>
              </div>
            </div>
          </div>
        </section>

        <!-- 注解 -->
        <section ref="annotations" data-id="annotations" class="section">
          <h3 class="section-title">注解</h3>
          <div v-if="annotationArray.length" class="kv">
            <div v-for="(a, i) in annotationArray" :key="i">
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
  name: 'DeploymentDetailDrawer',
  props: {
    visible: { type: Boolean, default: false },
    dep: { type: Object, required: true },
    width: { type: String, default: '55%' }
  },
  data() {
    return { activeSection: 'overview' }
  },
  computed: {
    spec() {
      return this.dep.spec || {}
    },
    tpl() {
      return this.dep.template || {}
    },
    status() {
      return this.dep.status || {}
    },
    selectorStr() {
      const m = (this.spec && this.spec.matchLabels) || {}
      const pairs = Object.keys(m).map((k) => k + '=' + m[k])
      return pairs.join(', ')
    },
    containers() {
      return Array.isArray(this.tpl.containers) ? this.tpl.containers : []
    },
    imagePullSecrets() {
      // 兼容 string[] 或 {name:string}[]
      const ips = this.tpl.imagePullSecrets || []
      return ips
        .map((s) => (typeof s === 'string' ? s : (s && s.name) || ''))
        .filter(Boolean)
    },
    annotationArray() {
      const obj = this.dep.annotations || {}
      return Object.keys(obj).map((k) => ({ k, v: obj[k] }))
    },
    prettyJSON() {
      try {
        return JSON.stringify(this.dep, null, 2)
      } catch (e) {
        return '{}'
      }
    }
  },
  methods: {
    handleBeforeClose(done) {
      this.$emit('update:visible', false)
      if (typeof done === 'function') done()
    },
    handleClose() {
      this.$emit('update:visible', false)
    },
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
      const ids = [
        'overview',
        'replicas',
        'strategy',
        'selectors',
        'template',
        'replicasets',
        'annotations',
        'raw'
      ]
      let current = ids[0]
      for (let i = 0; i < ids.length; i++) {
        const id = ids[i]
        const el = this.$refs[id]
        if (el && el.offsetTop - container.scrollTop <= 40) current = id
      }
      this.activeSection = current
    }
  }
}
</script>

<style scoped>
.dep-describe-drawer {
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
.summary-bar .dep-name {
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
  width: 240px;
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

.container-block {
  padding: 10px 12px;
  border: 1px solid #f0f0f0;
  border-radius: 10px;
  margin: 10px 0;
  background: #fff;
}
.container-title {
  margin-bottom: 6px;
  display: flex;
  gap: 8px;
  align-items: center;
}

.rs-list .rs-item {
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
