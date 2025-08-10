<template>
  <div class="deployment-table-container">
    <div class="table-title">
      <h2>Deployment List</h2>
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
      :data="pagedDeployments"
      border
      style="width: 100%"
      :header-cell-style="{
        background: '#f5f7fa',
        color: '#333',
        fontWeight: 600,
      }"
      empty-text="No Deployment data available"
    >
      <!-- ✅ 命名空间筛选列 -->
      <el-table-column prop="namespace" label="Namespace" width="160">
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
        <template slot-scope="{ row }">
          {{ row.namespace }}
        </template>
      </el-table-column>

      <el-table-column prop="name" label="Name" min-width="160" />
      <el-table-column prop="image" label="Image" min-width="220">
        <template slot-scope="{ row }">
          <el-input
            v-if="isEditing(row)"
            v-model="editCache[rowKey(row)].image"
            size="mini"
            placeholder="New Image Name"
          />
          <span v-else>{{ row.image }}</span>
        </template>
      </el-table-column>

      <el-table-column prop="replicas" label="Replicas" width="140">
        <template slot-scope="{ row }">
          <el-select
            v-if="isEditing(row)"
            v-model="editCache[rowKey(row)].replicas"
            size="mini"
            placeholder="Select replicas"
          >
            <el-option v-for="n in 10" :key="n" :label="n" :value="n" />
          </el-select>
          <span v-else>{{ row.replicas }}</span>
        </template>
      </el-table-column>

      <el-table-column prop="labelCount" label="Labels" width="100" />
      <el-table-column prop="annotationCount" label="Annotations" width="120" />
      <el-table-column prop="creationTime" label="创建时间" width="180" />

      <!-- 操作列 -->
      <el-table-column label="操作" fixed="right" width="160">
        <template slot-scope="{ row }">
          <div class="action-buttons">
            <el-button
              size="mini"
              type="primary"
              plain
              icon="el-icon-view"
              @click="$emit('view', row)"
            >
              View
            </el-button>

            <el-button
              size="mini"
              type="warning"
              plain
              icon="el-icon-edit"
              @click="startEdit(row)"
              v-if="!isEditing(row)"
            >
              Edit
            </el-button>

            <el-button
              size="mini"
              type="success"
              plain
              icon="el-icon-check"
              @click="confirmEdit(row)"
              v-if="isEditing(row)"
            >
              Apply
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
      :total="filteredDeployments.length"
      @current-change="handlePageChange"
    />
  </div>
</template>

<script>
export default {
  name: "DeploymentTable",
  props: {
    deployments: {
      type: Array,
      required: true,
    },
  },
  data() {
    return {
      selectedNamespace: "",
      pageSize: 10,
      currentPage: 1,
      editCache: {},
    };
  },
  computed: {
    namespaceOptions() {
      return [...new Set(this.deployments.map((d) => d.namespace))].filter(
        Boolean
      );
    },
    filteredDeployments() {
      return this.deployments.filter((d) => {
        if (this.selectedNamespace && d.namespace !== this.selectedNamespace) {
          return false;
        }
        return true;
      });
    },
    pagedDeployments() {
      const start = (this.currentPage - 1) * this.pageSize;
      return this.filteredDeployments.slice(start, start + this.pageSize);
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
    rowKey(row) {
      return `${row.namespace}/${row.name}`;
    },

    isEditing(row) {
      return !!this.editCache[this.rowKey(row)];
    },

    startEdit(row) {
      this.$set(this.editCache, this.rowKey(row), {
        image: row.image,
        replicas: parseInt(row.replicas.split("/")[1]) || 1, // 使用总副本数
      });
    },

    confirmEdit(row) {
      const key = this.rowKey(row);
      const { image, replicas } = this.editCache[key];

      const imageOriginal = row.image;
      const replicasOriginal = parseInt(row.replicas.split("/")[1]) || 1;

      // ✅ 判断是否有实际修改
      if (image === imageOriginal && replicas === replicasOriginal) {
        this.$message({
          message: "本次无修改",
          type: "info",
          duration: 1000,
        });
        this.$delete(this.editCache, key); // 清除编辑状态
        return;
      }

      this.$confirm("确定要修改该 Deployment 的副本数和镜像吗？", "确认修改", {
        confirmButtonText: "确定",
        cancelButtonText: "取消",
        type: "warning",
      })
        .then(() => {
          this.$emit("update", {
            namespace: row.namespace,
            name: row.name,
            image,
            replicas,
          });
          this.$delete(this.editCache, key); // 清除编辑状态
        })
        .catch(() => {
          // 用户取消
        });
    },
  },
};
</script>

<style scoped>
.deployment-table-container {
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
  justify-content: center;
  gap: 6px;
}
.action-buttons .el-button {
  padding: 2px 6px;
  font-size: 11px;
  min-width: 50px;
}
</style>
