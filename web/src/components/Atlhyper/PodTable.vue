<template>
  <div class="pod-table-container">
    <div class="table-title">
      <h2>Pod Resource List</h2>
      <hr />
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

    <el-table
      :data="pagedPods"
      border
      style="width: 100%"
      :header-cell-style="{
        background: '#f5f7fa',
        color: '#333',
        fontWeight: 600,
      }"
      empty-text="No Pod data available"
    >
      <!-- Namespace 筛选 -->
      <el-table-column
        prop="namespace"
        label="Namespace"
        :width="colWidth.namespace"
      >
        <template slot="header">
          <el-select
            v-model="selectedNamespace"
            placeholder="All Namespaces"
            clearable
            size="small"
            style="width: 100%"
          >
            <el-option
              v-for="item in namespaceOptions"
              :key="item"
              :label="item"
              :value="item"
            />
          </el-select>
        </template>
      </el-table-column>

      <!-- Deployment 筛选 -->
      <el-table-column
        prop="deployment"
        label="Deployment"
        :width="colWidth.deployment"
      >
        <template slot="header">
          <el-select
            v-model="selectedDeployment"
            placeholder="All Deployments"
            clearable
            size="small"
            style="width: 100%"
          >
            <el-option
              v-for="item in deploymentOptions"
              :key="item"
              :label="item"
              :value="item"
            />
          </el-select>
        </template>
      </el-table-column>

      <!-- Pod Name：用 min-width + 省略号提示 -->
      <el-table-column
        prop="name"
        label="Pod Name"
        :min-width="colWidth.name"
        show-overflow-tooltip
      />

      <el-table-column label="Ready" :width="colWidth.ready">
        <template slot-scope="{ row }">
          <el-tag :type="row.ready ? 'success' : 'info'" size="small">
            {{ row.ready ? "Yes" : "No" }}
          </el-tag>
        </template>
      </el-table-column>

      <el-table-column prop="phase" label="Phase" :width="colWidth.phase" />
      <el-table-column
        prop="restartCount"
        label="Restart Count"
        :width="colWidth.restartCount"
      />

      <!-- ✅ Start Time：格式化显示 + 悬浮原始 ISO -->
      <el-table-column
        prop="startTime"
        label="Start Time"
        :width="colWidth.startTime"
      >
        <template slot-scope="{ row }">
          <span :title="row.startTime">{{ fmtTime(row.startTime) }}</span>
        </template>
      </el-table-column>

      <el-table-column prop="podIP" label="Pod IP" :width="colWidth.podIP" />
      <el-table-column
        prop="nodeName"
        label="Node"
        :width="colWidth.nodeName"
      />

      <!-- 操作按钮 -->
      <el-table-column label="Actions" fixed="right" :width="colWidth.actions">
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

            <el-button
              size="mini"
              type="danger"
              plain
              :style="{ padding: '4px 8px', fontSize: '12px' }"
              icon="el-icon-delete"
              @click="emitRestart(row)"
            >
              Restart
            </el-button>
          </div>
        </template>
      </el-table-column>
    </el-table>

    <el-pagination
      class="pagination"
      background
      small
      layout="prev, pager, next"
      :page-size="pageSize"
      :current-page="currentPage"
      :total="filteredPods.length"
      @current-change="handlePageChange"
    />
  </div>
</template>

<script>
export default {
  name: "PodTable",
  props: {
    pods: { type: Array, required: true },
  },
  data() {
    return {
      // 统一管理列宽
      colWidth: {
        namespace: 150,
        deployment: 150,
        name: 220, // min-width
        ready: 80,
        phase: 110,
        restartCount: 130,
        startTime: 200, // 稍加宽，容纳格式化后的时间
        podIP: 160,
        nodeName: 150,
        actions: 180,
      },

      selectedNamespace: "",
      selectedDeployment: "",
      pageSize: 10,
      currentPage: 1,
    };
  },
  computed: {
    namespaceOptions() {
      return [...new Set(this.pods.map((p) => p.namespace))].filter(Boolean);
    },
    deploymentOptions() {
      return [...new Set(this.pods.map((p) => p.deployment))].filter(Boolean);
    },
    filteredPods() {
      return this.pods.filter((pod) => {
        if (this.selectedNamespace && pod.namespace !== this.selectedNamespace)
          return false;
        if (
          this.selectedDeployment &&
          pod.deployment !== this.selectedDeployment
        )
          return false;
        return true;
      });
    },
    pagedPods() {
      const start = (this.currentPage - 1) * this.pageSize;
      return this.filteredPods.slice(start, start + this.pageSize);
    },
  },
  methods: {
    handlePageChange(page) {
      this.currentPage = page;
    },
    handlePageSizeChange(size) {
      this.pageSize = size;
      this.currentPage = 1;
    },
    emitRestart(row) {
      this.$emit("restart", row);
    },

    // ===== 时间格式化（本地时区） =====
    fmtTime(ts) {
      const ms = this.parseIsoToMs(ts);
      if (!Number.isFinite(ms)) return ts || "-";
      const d = new Date(ms);
      const pad = (n, w = 2) => String(n).padStart(w, "0");
      const yyyy = d.getFullYear();
      const MM = pad(d.getMonth() + 1);
      const DD = pad(d.getDate());
      const hh = pad(d.getHours());
      const mm = pad(d.getMinutes());
      const ss = pad(d.getSeconds());
      return `${yyyy}-${MM}-${DD} ${hh}:${mm}:${ss}`; // e.g. 2025-06-29 19:19:42
    },
    // 兼容：2025-06-29T19:19:42+09:00 / 带毫秒或纳秒 / Z 结尾
    parseIsoToMs(ts) {
      if (typeof ts !== "string") return NaN;
      const m = ts.match(
        /^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})(\.(\d+))?([Zz]|[+-]\d{2}:\d{2})?$/
      );
      if (!m) {
        const t = Date.parse(ts);
        return Number.isFinite(t) ? t : NaN;
      }
      const base = m[1];
      const frac = m[3] || "";
      const tz = m[4] || "Z";
      const ms3 = (frac + "000").slice(0, 3); // 只保留毫秒
      const iso = `${base}.${ms3}${tz}`;
      const t = Date.parse(iso);
      return Number.isFinite(t) ? t : NaN;
    },
  },
};
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
.action-buttons {
  display: flex;
  gap: 6px;
  white-space: nowrap;
}
</style>
