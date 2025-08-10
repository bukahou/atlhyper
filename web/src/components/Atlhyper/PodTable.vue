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
      <el-table-column prop="namespace" label="Namespace" width="140">
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
      <el-table-column prop="deployment" label="Deployment" width="140">
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

      <el-table-column prop="name" label="Pod Name" min-width="160" />
      <el-table-column label="Ready" width="80">
        <template slot-scope="{ row }">
          <el-tag :type="row.ready ? 'success' : 'info'" size="small">
            {{ row.ready ? "Yes" : "No" }}
          </el-tag>
        </template>
      </el-table-column>

      <el-table-column prop="phase" label="Phase" width="100" />
      <el-table-column prop="restartCount" label="Restart Count" width="120" />
      <el-table-column prop="startTime" label="Start Time" width="180" />
      <el-table-column prop="podIP" label="Pod IP" width="150" />
      <el-table-column prop="nodeName" label="Node" width="140" />

      <!-- 操作按钮 -->
      <el-table-column label="Actions" fixed="right" width="160">
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
    pods: {
      type: Array,
      required: true,
    },
  },
  data() {
    return {
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
        if (
          this.selectedNamespace &&
          pod.namespace !== this.selectedNamespace
        ) {
          return false;
        }
        if (
          this.selectedDeployment &&
          pod.deployment !== this.selectedDeployment
        ) {
          return false;
        }
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
}
</style>
