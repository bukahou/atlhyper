<template>
  <div class="probes-editor">
    <!-- ========== Readiness (HTTP) ========== -->
    <el-form-item label="Readiness · HTTP 路径 / 端口">
      <div class="row">
        <el-input
          v-model="m.readiness.http.path"
          placeholder="/healthz/readiness（留空则不生成）"
          class="mr8"
        />
        <el-input
          v-model.number="m.readiness.http.port"
          placeholder="端口（留空则不生成）"
        />
      </div>
      <div class="row mt8">
        <el-select
          v-model="m.readiness.http.scheme"
          placeholder="协议（默认 HTTP）"
          class="mr8"
        >
          <el-option label="HTTP" value="HTTP" />
          <el-option label="HTTPS" value="HTTPS" />
        </el-select>
        <span class="hint">仅当“路径 + 端口”都填写时才会写入 YAML。</span>
      </div>
    </el-form-item>

    <el-form-item label="Readiness 时序参数（可选）">
      <div class="row">
        <el-input
          v-model.number="m.readiness.initialDelaySeconds"
          placeholder="initialDelaySeconds"
          class="mr8"
        />
        <el-input
          v-model.number="m.readiness.periodSeconds"
          placeholder="periodSeconds"
          class="mr8"
        />
        <el-input
          v-model.number="m.readiness.timeoutSeconds"
          placeholder="timeoutSeconds"
          class="mr8"
        />
        <el-input
          v-model.number="m.readiness.failureThreshold"
          placeholder="failureThreshold"
          class="mr8"
        />
        <el-input
          v-model.number="m.readiness.successThreshold"
          placeholder="successThreshold"
        />
      </div>
    </el-form-item>

    <!-- ========== Liveness (HTTP) ========== -->
    <el-form-item label="Liveness · HTTP 路径 / 端口" class="mt12">
      <div class="row">
        <el-input
          v-model="m.liveness.http.path"
          placeholder="/healthz/liveness（留空则不生成）"
          class="mr8"
        />
        <el-input
          v-model.number="m.liveness.http.port"
          placeholder="端口（留空则不生成）"
        />
      </div>
      <div class="row mt8">
        <el-select
          v-model="m.liveness.http.scheme"
          placeholder="协议（默认 HTTP）"
          class="mr8"
        >
          <el-option label="HTTP" value="HTTP" />
          <el-option label="HTTPS" value="HTTPS" />
        </el-select>
        <span class="hint">仅当“路径 + 端口”都填写时才会写入 YAML。</span>
      </div>
    </el-form-item>

    <el-form-item label="Liveness 时序参数（可选）">
      <div class="row">
        <el-input
          v-model.number="m.liveness.initialDelaySeconds"
          placeholder="initialDelaySeconds"
          class="mr8"
        />
        <el-input
          v-model.number="m.liveness.periodSeconds"
          placeholder="periodSeconds"
          class="mr8"
        />
        <el-input
          v-model.number="m.liveness.timeoutSeconds"
          placeholder="timeoutSeconds"
          class="mr8"
        />
        <el-input
          v-model.number="m.liveness.failureThreshold"
          placeholder="failureThreshold"
        />
      </div>
    </el-form-item>
  </div>
</template>

<script>
const clone = (x) => JSON.parse(JSON.stringify(x || {}));
const trim = (x) => String(x || "").trim();

export default {
  name: "ProbesEditor",
  props: {
    // 父组件 v-model 值：{ readiness: {...} | null, liveness: {...} | null }
    value: {
      type: Object,
      default: () => ({ readiness: null, liveness: null }),
    },
    // 用于智能推断端口（此版本不自动填充，只作为未来扩展）
    ports: { type: Array, default: () => [] },
  },
  data() {
    // 初始化：尽量保留父值；空则给出结构但不设默认值（path='', port=null）
    const r = clone(this.value?.readiness) || {};
    const l = clone(this.value?.liveness) || {};
    return {
      m: {
        readiness: {
          // HTTP 结构固定存在，便于双绑；不填就不写入
          http: {
            path: trim(r?.http?.path || ""),
            port: Number.isFinite(r?.http?.port) ? r.http.port : null,
            scheme: r?.http?.scheme || "HTTP",
          },
          // 时序参数，只有填了才输出
          initialDelaySeconds: Number.isFinite(r?.initialDelaySeconds)
            ? r.initialDelaySeconds
            : undefined,
          periodSeconds: Number.isFinite(r?.periodSeconds)
            ? r.periodSeconds
            : undefined,
          timeoutSeconds: Number.isFinite(r?.timeoutSeconds)
            ? r.timeoutSeconds
            : undefined,
          failureThreshold: Number.isFinite(r?.failureThreshold)
            ? r.failureThreshold
            : undefined,
          successThreshold: Number.isFinite(r?.successThreshold)
            ? r.successThreshold
            : undefined,
        },
        liveness: {
          http: {
            path: trim(l?.http?.path || ""),
            port: Number.isFinite(l?.http?.port) ? l.http.port : null,
            scheme: l?.http?.scheme || "HTTP",
          },
          initialDelaySeconds: Number.isFinite(l?.initialDelaySeconds)
            ? l.initialDelaySeconds
            : undefined,
          periodSeconds: Number.isFinite(l?.periodSeconds)
            ? l.periodSeconds
            : undefined,
          timeoutSeconds: Number.isFinite(l?.timeoutSeconds)
            ? l.timeoutSeconds
            : undefined,
          failureThreshold: Number.isFinite(l?.failureThreshold)
            ? l.failureThreshold
            : undefined,
        },
      },
    };
  },
  methods: {
    normalizeHttpProbe(p, withSuccess) {
      // 只要 path 与 port 其中一个缺失，就返回 null（父级不生成）
      const path = trim(p?.http?.path || "");
      const port = Number(p?.http?.port);
      if (!path || !Number.isFinite(port)) return null;

      const out = {
        http: {
          path,
          port,
          scheme: p?.http?.scheme === "HTTPS" ? "HTTPS" : "HTTP",
        },
      };
      // 可选时序
      if (Number.isFinite(p?.initialDelaySeconds))
        out.initialDelaySeconds = p.initialDelaySeconds;
      if (Number.isFinite(p?.periodSeconds))
        out.periodSeconds = p.periodSeconds;
      if (Number.isFinite(p?.timeoutSeconds))
        out.timeoutSeconds = p.timeoutSeconds;
      if (Number.isFinite(p?.failureThreshold))
        out.failureThreshold = p.failureThreshold;
      if (withSuccess && Number.isFinite(p?.successThreshold))
        out.successThreshold = p.successThreshold;

      return out;
    },
  },
  watch: {
    // 本地编辑 -> 回传父组件（v-model）
    m: {
      deep: true,
      handler(v) {
        this.$emit("input", {
          readiness: this.normalizeHttpProbe(v.readiness, true),
          liveness: this.normalizeHttpProbe(v.liveness, false),
        });
      },
    },
    // 父组件值变化（例如回填）-> 合并到本地（就地更新，避免替换引用）
    value: {
      deep: true,
      handler(nv) {
        const r = nv?.readiness || null;
        const l = nv?.liveness || null;
        if (r) {
          this.m.readiness.http.path = trim(r?.http?.path || "");
          this.m.readiness.http.port = Number.isFinite(r?.http?.port)
            ? r.http.port
            : null;
          this.m.readiness.http.scheme = r?.http?.scheme || "HTTP";
          this.m.readiness.initialDelaySeconds = Number.isFinite(
            r?.initialDelaySeconds
          )
            ? r.initialDelaySeconds
            : undefined;
          this.m.readiness.periodSeconds = Number.isFinite(r?.periodSeconds)
            ? r.periodSeconds
            : undefined;
          this.m.readiness.timeoutSeconds = Number.isFinite(r?.timeoutSeconds)
            ? r.timeoutSeconds
            : undefined;
          this.m.readiness.failureThreshold = Number.isFinite(
            r?.failureThreshold
          )
            ? r.failureThreshold
            : undefined;
          this.m.readiness.successThreshold = Number.isFinite(
            r?.successThreshold
          )
            ? r.successThreshold
            : undefined;
        } else {
          // 父值清空时，不强制清空用户草稿；如需清空，按需启用下面两行
          // this.m.readiness.http.path = ''
          // this.m.readiness.http.port = null
        }

        if (l) {
          this.m.liveness.http.path = trim(l?.http?.path || "");
          this.m.liveness.http.port = Number.isFinite(l?.http?.port)
            ? l.http.port
            : null;
          this.m.liveness.http.scheme = l?.http?.scheme || "HTTP";
          this.m.liveness.initialDelaySeconds = Number.isFinite(
            l?.initialDelaySeconds
          )
            ? l.initialDelaySeconds
            : undefined;
          this.m.liveness.periodSeconds = Number.isFinite(l?.periodSeconds)
            ? l.periodSeconds
            : undefined;
          this.m.liveness.timeoutSeconds = Number.isFinite(l?.timeoutSeconds)
            ? l.timeoutSeconds
            : undefined;
          this.m.liveness.failureThreshold = Number.isFinite(
            l?.failureThreshold
          )
            ? l.failureThreshold
            : undefined;
        } else {
          // 同上：不强行清空草稿
          // this.m.liveness.http.path = ''
          // this.m.liveness.http.port = null
        }
      },
    },
  },
};
</script>

<style scoped>
.row {
  display: flex;
  align-items: center;
}
.mr8 {
  margin-right: 8px;
}
.mt8 {
  margin-top: 8px;
}
.mt12 {
  margin-top: 12px;
}
.hint {
  color: #909399;
  font-size: 12px;
}
</style>
