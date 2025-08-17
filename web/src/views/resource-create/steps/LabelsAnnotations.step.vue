<template>
  <div class="step-labels-annotations">
    <!-- Labels -->
    <div class="box mb12">
      <div class="box-title">
        Labels
        <el-button
          type="text"
          size="mini"
          @click="quickAddLabel('app', basicName)"
          >+ app={{ basicName || "..." }}</el-button
        >
        <el-button
          type="text"
          size="mini"
          @click="quickAddLabel('version', 'v1')"
          >+ version=v1</el-button
        >
      </div>
      <el-table :data="labels" border size="mini" style="width: 100%">
        <el-table-column label="Key" width="220">
          <template slot-scope="{ row }">
            <el-input
              v-model="row.key"
              placeholder="k/v 需为字母数字和 - . _"
            />
          </template>
        </el-table-column>
        <el-table-column label="Value">
          <template slot-scope="{ row }">
            <el-input v-model="row.value" />
          </template>
        </el-table-column>
        <el-table-column width="90" label="操作">
          <template slot-scope="{ $index }">
            <el-button size="mini" type="text" @click="labels.splice($index, 1)"
              >删</el-button
            >
          </template>
        </el-table-column>
      </el-table>
      <div class="mt8">
        <el-button size="mini" @click="labels.push({ key: '', value: '' })"
          >+ 添加</el-button
        >
      </div>
    </div>

    <!-- Annotations -->
    <div class="box mb12">
      <div class="box-title">
        Annotations
        <el-button
          type="text"
          size="mini"
          @click="quickAddAnno('kubectl.kubernetes.io/restartedAt', nowIso)"
          >+ restartedAt</el-button
        >
      </div>
      <el-table :data="annotations" border size="mini" style="width: 100%">
        <el-table-column label="Key" width="260">
          <template slot-scope="{ row }">
            <el-input
              v-model="row.key"
              placeholder="如：kubectl.kubernetes.io/restartedAt"
            />
          </template>
        </el-table-column>
        <el-table-column label="Value">
          <template slot-scope="{ row }">
            <el-input v-model="row.value" />
          </template>
        </el-table-column>
        <el-table-column width="90" label="操作">
          <template slot-scope="{ $index }">
            <el-button
              size="mini"
              type="text"
              @click="annotations.splice($index, 1)"
              >删</el-button
            >
          </template>
        </el-table-column>
      </el-table>
      <div class="mt8">
        <el-button size="mini" @click="annotations.push({ key: '', value: '' })"
          >+ 添加</el-button
        >
      </div>
    </div>

    <!-- ✅ 作用范围选择 -->
    <div class="box mt12">
      <div class="box-title">作用范围（Scope）</div>
      <el-checkbox-group v-model="scopeSelected">
        <el-checkbox label="metadata">资源 metadata</el-checkbox>
        <el-checkbox label="podTemplate">Pod 模板</el-checkbox>
        <el-checkbox label="service">Service</el-checkbox>
        <el-checkbox label="ingress">Ingress</el-checkbox>
      </el-checkbox-group>
      <div class="hint">
        不勾选则默认应用到 <code>metadata</code> 与
        <code>podTemplate</code>（builder 已按此策略生成）。
      </div>
    </div>
  </div>
</template>

<script>
import store from "../stores/createForm.store";

function stripOuterQuotes(s = "") {
  const t = String(s).trim();
  if (
    (t.startsWith('"') && t.endsWith('"')) ||
    (t.startsWith("'") && t.endsWith("'"))
  ) {
    return t.slice(1, -1);
  }
  return t;
}

function normalizeKV(list = []) {
  const out = [];
  const seen = new Set();
  (list || []).forEach(({ key = "", value = "" }) => {
    const k = String(key).trim();
    if (!k) return;
    const v = stripOuterQuotes(value);
    if (seen.has(k)) return;
    seen.add(k);
    out.push({ key: k, value: v });
  });
  return out;
}

export default {
  name: "LabelsAnnotationsStep",
  data() {
    const s = store.form;
    return {
      labels: normalizeKV(s.labels || []),
      annotations: normalizeKV(s.annotations || []),
      scopeSelected:
        Array.isArray(s.scope) && s.scope.length
          ? [...s.scope]
          : ["metadata", "podTemplate"],
    };
  },
  computed: {
    basicName() {
      return store.form?.basic?.name || "";
    },
    nowIso() {
      const d = new Date();
      const tz = -d.getTimezoneOffset();
      const sign = tz >= 0 ? "+" : "-";
      const pad = (n) => String(Math.floor(Math.abs(n))).padStart(2, "0");
      const hh = pad(tz / 60);
      const mm = pad(tz % 60);
      return d.toISOString().replace("Z", `${sign}${hh}:${mm}`);
    },
  },
  watch: {
    labels: {
      deep: true,
      handler(v) {
        store.form.labels = normalizeKV(v);
      },
    },
    annotations: {
      deep: true,
      handler(v) {
        store.form.annotations = normalizeKV(v);
      },
    },
    scopeSelected(v) {
      store.form.scope = Array.isArray(v) ? [...v] : [];
    },
  },
  mounted() {
    if (Array.isArray(store.form.annotations)) {
      store.form.annotations = normalizeKV(store.form.annotations);
    }
    if (Array.isArray(store.form.labels)) {
      store.form.labels = normalizeKV(store.form.labels);
    }
  },
  methods: {
    quickAddLabel(k, v) {
      if (!k) return;
      const key = String(k).trim();
      const val = stripOuterQuotes(v);
      const idx = this.labels.findIndex((x) => x.key === key);
      if (idx >= 0) this.$set(this.labels, idx, { key, value: val });
      else this.labels.push({ key, value: val });
    },
    quickAddAnno(k, v) {
      if (!k) return;
      const key = String(k).trim();
      const val = stripOuterQuotes(v);
      const idx = this.annotations.findIndex((x) => x.key === key);
      if (idx >= 0) this.$set(this.annotations, idx, { key, value: val });
      else this.annotations.push({ key, value: val });
    },
  },
};
</script>

<style scoped>
.box {
  border: 1px solid #ebeef5;
  border-radius: 6px;
  padding: 8px;
}
.box-title {
  font-weight: 600;
  margin: 4px 0 8px;
  display: flex;
  align-items: center;
  gap: 8px;
}
.mt8 {
  margin-top: 8px;
}
.mb12 {
  margin-bottom: 12px;
}
.mt12 {
  margin-top: 12px;
}
.hint {
  margin-top: 6px;
  color: #909399;
  font-size: 12px;
}
</style>
