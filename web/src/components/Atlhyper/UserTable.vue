<template>
  <div class="user-table-container">
    <div class="table-header">
      <h2 class="table-title">用户列表</h2>
      <el-button
        class="register-button"
        type="primary"
        size="medium"
        plain
        round
        @click="$emit('open-register')"
      >
        注册用户
      </el-button>
    </div>
    <hr>

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
      <el-table-column label="启用状态" width="120">
        <template slot-scope="{ row }">
          <el-switch
            v-model="row.enabledSwitch"
            :active-value="true"
            :inactive-value="false"
            active-color="#13ce66"
            inactive-color="#dcdfe6"
            @change="() => toggleUserEnable(row)"
          />
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
              <el-option label="普通用户" value="普通用户" />
              <el-option label="管理员" value="管理员" />
              <el-option label="超级管理员" value="超级管理员" />
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
import { updateUserRole } from '@/api/user'

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
      return this.users.slice(start, start + this.pageSize).map((user) => {
        const enabled =
          typeof user.role === 'number'
            ? user.role > 0
            : ['普通用户', '管理员', '超级管理员'].includes(user.role)

        return {
          ...user,
          enabledSwitch: enabled
        }
      })
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
        .then(async() => {
          const roleMap = {
            普通用户: 1,
            管理员: 2,
            超级管理员: 3
          }
          const newRoleNum = roleMap[row.role]

          try {
            const res = await updateUserRole({ id: row.id, role: newRoleNum })
            if (res.code === 20000) {
              this.$message.success('✅ 角色更新成功')
              this.editingRow = null
            } else {
              this.$message.error('❌ 更新失败：' + res.message)
            }
          } catch (err) {
            this.$message.error('❌ 请求失败：' + err.message)
          }
        })
        .catch(() => {
          console.log('取消修改')
        })
    },
    async toggleUserEnable(row) {
      const isEnabled = row.enabledSwitch
      const newRole = isEnabled ? 1 : 0
      const action = isEnabled ? '启用' : '禁用'

      this.$confirm(`确定${action}用户「${row.username}」吗？`, '确认操作', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: isEnabled ? 'info' : 'warning'
      })
        .then(async() => {
          try {
            const res = await updateUserRole({ id: row.id, role: newRole })
            if (res.code === 20000) {
              row.role = newRole
              this.$message.success(`✅ 已${action}`)
            } else {
              row.enabledSwitch = !isEnabled
              this.$message.error(`❌ ${action}失败：` + res.message)
            }
          } catch (err) {
            row.enabledSwitch = !isEnabled
            this.$message.error('❌ 请求失败：' + err.message)
          }
        })
        .catch(() => {
          row.enabledSwitch = !isEnabled
          console.log('取消操作')
        })
    }
  }
}
</script>

<style scoped>
.user-table-container {
  padding: 16px;
}

.table-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.table-title {
  font-size: 20px;
  font-weight: bold;
  margin: 0;
}
.register-button {
  margin-right: 20px;
  padding: 6px 18px;
  font-size: 14px;
  font-weight: 400;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(64, 158, 255, 0.2);
  transition: all 0.3s ease;
}
.register-button:hover {
  background-color: #409eff;
  color: #fff;
  box-shadow: 0 4px 12px rgba(64, 158, 255, 0.4);
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
