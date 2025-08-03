<template>
  <div class="namespace-table-container">
    <div class="table-title">
      <h2>Namespace 一览表</h2>
      <hr />
    </div>

    <!-- 分页控制 -->
    <div class="toolbar">
      <div class="row-size-selector">
        显示
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
        条
      </div>
    </div>

    <!-- 表格展示 -->
    <el-table
      :data="pagedNamespaces"
      border
      style="width: 100%"
      :header-cell-style="{
        background: '#f5f7fa',
        color: '#333',
        fontWeight: 600,
      }"
      empty-text="暂无 Namespace 数据"
    >
      <el-table-column prop="name" label="名称" min-width="160" />
      <el-table-column prop="status" label="状态" width="100" />
      <el-table-column prop="podCount" label="Pod 数量" width="100" />
      <el-table-column prop="labelCount" label="标签数量" width="100" />
      <el-table-column prop="annotationCount" label="注解数量" width="100" />
      <el-table-column prop="creationTime" label="创建时间" width="180" />

      <!-- 操作列 -->
      <el-table-column label="操作" fixed="right" width="140">
        <template slot-scope="{ row }">
          <div class="action-buttons">
            <el-button
              size="mini"
              type="primary"
              plain
              icon="el-icon-document"
              :style="{ padding: '4px 12px', fontSize: '12px' }"
              @click="
                $router.push({
                  path: '/cluster-monitor/configmap',
                  query: { ns: row.name },
                })
              "
            >
              查看
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
      :total="namespaces.length"
      @current-change="handlePageChange"
    />
  </div>
</template>

<script>
export default {
  name: "NamespaceTable",
  props: {
    namespaces: {
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
    pagedNamespaces() {
      const start = (this.currentPage - 1) * this.pageSize;
      return this.namespaces.slice(start, start + this.pageSize);
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
.namespace-table-container {
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

.action-buttons {
  display: flex;
  justify-content: center;
  gap: 6px;
}
</style>
