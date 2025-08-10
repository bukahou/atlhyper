<template>
  <div class="node-table-container">
    <div class="table-title">
      <h2>Node List</h2>
      <hr />
    </div>

    <!-- 分页控制 -->
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

    <!-- 表格展示 -->
    <el-table
      :data="pagedNodes"
      border
      style="width: 100%"
      :header-cell-style="{
        background: '#f5f7fa',
        color: '#333',
        fontWeight: 600,
      }"
      empty-text="No Node data available"
    >
      <el-table-column prop="name" label="Name" min-width="160" />

      <!-- ✅ Ready 状态使用 tag 形式展示 -->
      <el-table-column label="Ready" width="100">
        <template slot-scope="{ row }">
          <el-tag
            :type="row.ready ? 'success' : 'danger'"
            size="mini"
            disable-transitions
          >
            {{ row.ready ? "Ready" : "NotReady" }}
          </el-tag>
        </template>
      </el-table-column>

      <el-table-column prop="internalIP" label="Internal IP" width="160" />
      <el-table-column prop="osImage" label="OS Image" min-width="200" />
      <el-table-column prop="architecture" label="Architecture" width="120" />
      <el-table-column prop="cpu" label="CPU Cores" width="120" />
      <el-table-column prop="memory" label="Memory (GiB)" width="140" />

      <!-- ✅ 新增调度状态列 -->
      <el-table-column label="Schedulable" width="140">
        <template slot-scope="{ row }">
          <el-tag
            :type="row.unschedulable ? 'danger' : 'success'"
            size="mini"
            disable-transitions
          >
            {{ row.unschedulable ? "NotSchedulable" : "Schedulable" }}
          </el-tag>
        </template>
      </el-table-column>

      <!-- 操作列 -->
      <el-table-column label="Actions" fixed="right" width="220">
        <template slot-scope="{ row }">
          <div class="action-buttons">
            <!-- 查看按钮 -->
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

            <!-- 封锁/解封按钮 -->
            <el-button
              size="mini"
              :type="row.unschedulable ? 'success' : 'danger'"
              plain
              :style="{ padding: '4px 8px', fontSize: '12px' }"
              :icon="row.unschedulable ? 'el-icon-unlock' : 'el-icon-lock'"
              @click="$emit('toggle', row)"
            >
              {{ row.unschedulable ? "Unblock" : "Block" }}
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
      :total="nodes.length"
      @current-change="handlePageChange"
    />
  </div>
</template>

<script>
export default {
  name: "NodeTable",
  props: {
    nodes: {
      type: Array,
      required: true,
    },
  },
  data() {
    return {
      pageSize: 10,
      currentPage: 1,
    };
  },
  computed: {
    pagedNodes() {
      const start = (this.currentPage - 1) * this.pageSize;
      return this.nodes.slice(start, start + this.pageSize);
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
  },
};
</script>

<style scoped>
.node-table-container {
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
