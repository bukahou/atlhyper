<template>
  <div class="user-table-container">
    <div class="table-title">
      <h2>用户列表</h2>
      <hr>
    </div>

    <el-table
      :data="pagedUsers"
      border
      style="width: 100%"
      :header-cell-style="{
        background: '#f5f7fa',
        color: '#333',
        fontWeight: 600,
      }"
      empty-text="暂无用户数据"
    >
      <el-table-column prop="username" label="用户名" min-width="120" />
      <el-table-column prop="displayName" label="显示名" min-width="140" />
      <el-table-column prop="email" label="邮箱" min-width="200">
        <template slot-scope="{ row }">
          <span>{{ row.email || "—" }}</span>
        </template>
      </el-table-column>

      <el-table-column label="角色" width="160">
        <template slot-scope="{ row }">
          <div v-if="editingRow === row.username">
            <el-select
              v-model="row.role"
              placeholder="请选择角色"
              size="mini"
              @change="updateRole(row)"
            >
              <el-option label="超级管理员" value="超级管理员" />
              <el-option label="普通用户" value="普通用户" />
              <el-option label="访客" value="访客" />
            </el-select>
          </div>
          <div v-else>{{ row.role }}</div>
        </template>
      </el-table-column>

      <el-table-column label="操作" width="140">
        <template slot-scope="{ row }">
          <el-button
            v-if="editingRow !== row.username"
            size="mini"
            type="primary"
            plain
            icon="el-icon-edit"
            class="update-btn"
            @click="editingRow = row.username"
          >
            权限更新
          </el-button>

          <el-button
            v-else
            size="mini"
            type="success"
            plain
            icon="el-icon-check"
            class="update-btn"
            @click="confirmRoleUpdate(row)"
          >
            确认修改
          </el-button>
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
      :total="users.length"
      @current-change="handlePageChange"
    />
  </div>
</template>

<script>
export default {
  name: 'UserTable',
  props: {
    users: {
      type: Array,
      required: true
    }
  },
  data() {
    return {
      pageSize: 10,
      currentPage: 1,
      editingRow: null
    }
  },
  computed: {
    pagedUsers() {
      const start = (this.currentPage - 1) * this.pageSize
      return this.users.slice(start, start + this.pageSize)
    }
  },
  methods: {
    handlePageChange(page) {
      this.currentPage = page
    },
    updateRole(row) {
      // v-model 已绑定，无需额外逻辑
    },
    confirmRoleUpdate(row) {
      this.$confirm(
        `确定将用户「${row.username}」的角色修改为「${row.role}」吗？`,
        '确认角色修改',
        {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        }
      )
        .then(() => {
          console.log('角色已确认修改为：', row.role)
          this.editingRow = null
        })
        .catch(() => {
          console.log('取消修改')
        })
    }
  }
}
</script>

<style scoped>
.user-table-container {
  padding: 16px;
}

.table-title {
  margin-bottom: 16px;
}

.update-btn {
  width: 110px;
  height: 32px;
  padding: 0;
  font-size: 13px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 4px;
}
</style>
