<template>
  <el-card class="todos-card two-third" shadow="never">
    <div class="card-header">
      <div class="title">我的代办</div>
      <div>
        <el-button
          type="primary"
          size="mini"
          @click="openCreate"
        >新增</el-button>
        <el-button
          size="mini"
          :loading="loading"
          @click="fetchMine"
        >刷新</el-button>
      </div>
    </div>

    <el-table
      v-loading="loading"
      :data="todos"
      size="mini"
      stripe
      border
      height="320"
    >
      <el-table-column type="index" width="46" />
      <el-table-column
        prop="title"
        label="标题"
        min-width="180"
        show-overflow-tooltip
      />
      <el-table-column prop="priority" label="优先级" width="90">
        <template slot-scope="scope">
          <el-tag :type="priorityType(scope.row.priority)" size="mini">
            P{{ scope.row.priority || 2 }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="due_date" label="截止" width="120">
        <template slot-scope="scope">{{ scope.row.due_date || "-" }}</template>
      </el-table-column>

      <el-table-column prop="is_done" label="状态" width="90">
        <template slot-scope="scope">
          <el-switch
            v-model="scope.row.is_done"
            :active-value="1"
            :inactive-value="0"
            :disabled="scope.row._updating === true"
            @change="(val) => toggleDone(scope.row, val)"
          />
        </template>
      </el-table-column>

      <el-table-column label="操作" width="160" fixed="right">
        <template slot-scope="scope">
          <el-button
            type="text"
            size="mini"
            @click="editTodo(scope.row)"
          >view</el-button>
          <el-divider direction="vertical" />
          <el-popconfirm
            title="确认删除该代办吗？"
            @confirm="removeTodo(scope.row.id)"
          >
            <el-button
              slot="reference"
              type="text"
              size="mini"
            >delete</el-button>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog
      :visible.sync="dialogVisible"
      :title="editMode ? '编辑代办' : '新增代办'"
      width="480px"
    >
      <el-form
        ref="form"
        :model="form"
        label-width="80px"
        size="small"
        :rules="rules"
      >
        <el-form-item label="标题" prop="title">
          <el-input v-model.trim="form.title" maxlength="80" show-word-limit />
        </el-form-item>
        <el-form-item label="内容">
          <el-input v-model.trim="form.content" type="textarea" :rows="3" />
        </el-form-item>
        <el-form-item label="优先级">
          <el-radio-group v-model="form.priority">
            <el-radio :label="1">P1</el-radio>
            <el-radio :label="2">P2</el-radio>
            <el-radio :label="3">P3</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="截止日期">
          <el-date-picker
            v-model="form.due_date"
            type="date"
            value-format="yyyy-MM-dd"
            clearable
            style="width: 200px"
          />
          <el-button
            type="text"
            class="ml8"
            @click="form.due_date = ''"
          >清空</el-button>
        </el-form-item>
        <el-form-item label="状态">
          <el-switch
            v-model="form.is_done"
            :active-value="1"
            :inactive-value="0"
          />
        </el-form-item>
      </el-form>
      <span slot="footer">
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button
          type="primary"
          :loading="saving"
          @click="saveTodo"
        >保存</el-button>
      </span>
    </el-dialog>
  </el-card>
</template>

<script>
import {
  getUserTodosByUsername,
  createUserTodo,
  updateUserTodo,
  deleteUserTodo
} from '@/api/user'

export default {
  name: 'TodosMyCard',
  props: { username: { type: String, default: 'admin' }},
  data() {
    return {
      loading: false,
      saving: false,
      todos: [],
      dialogVisible: false,
      editMode: false,
      form: {
        id: null,
        username: this.username,
        title: '',
        content: '',
        priority: 2,
        is_done: 0,
        due_date: '',
        category: '个人'
      },
      rules: {
        title: [{ required: true, message: '请输入标题', trigger: 'blur' }]
      }
    }
  },
  created() {
    this.fetchMine()
  },
  methods: {
    async fetchMine() {
      this.loading = true
      try {
        const res = await getUserTodosByUsername(this.username)
        if (res.code === 20000 && res.data) {
          this.todos = res.data.items || []
        } else {
          this.$message.error(res.message || '加载失败')
        }
      } catch (e) {
        this.$message.error(e.message || '请求异常')
      } finally {
        this.loading = false
      }
    },
    openCreate() {
      this.editMode = false
      this.form = {
        username: this.username,
        title: '',
        content: '',
        priority: 2,
        is_done: 0,
        due_date: '',
        category: '个人'
      }
      this.dialogVisible = true
    },
    editTodo(row) {
      this.editMode = true
      this.form = Object.assign(
        {
          username: this.username,
          priority: 2,
          is_done: 0,
          due_date: '',
          category: '个人'
        },
        row
      )
      this.dialogVisible = true
    },
    async saveTodo() {
      this.$refs.form.validate(async(ok) => {
        if (!ok) return
        this.saving = true
        try {
          if (this.editMode) {
            const res = await updateUserTodo(this.clean(this.form))
            if (res.code === 20000) this.$message.success('更新成功')
            else return this.$message.error(res.message || '更新失败')
          } else {
            const res = await createUserTodo(this.clean(this.form))
            if (res.code === 20000) this.$message.success('新增成功')
            else return this.$message.error(res.message || '新增失败')
          }
          this.dialogVisible = false
          this.fetchMine()
        } catch (e) {
          this.$message.error(e.message || '保存失败')
        } finally {
          this.saving = false
        }
      })
    },
    async removeTodo(id) {
      try {
        await this.$confirm('确认删除该代办吗？', '提示', { type: 'warning' })
      } catch (_) {
        return
      }
      try {
        const res = await deleteUserTodo(id)
        if (res.code === 20000) {
          this.$message.success('删除成功')
          this.fetchMine()
        } else {
          this.$message.error(res.message || '删除失败')
        }
      } catch (e) {
        this.$message.error(e.message || '删除失败')
      }
    },

    async toggleDone(row, val) {
      const prev = row.is_done // 0/1
      row._updating = true
      try {
        const res = await updateUserTodo({ id: row.id, is_done: val })
        if (res.code === 20000) {
          this.$message.success('状态已更新')
        } else {
          row.is_done = prev // 回滚
          this.$message.error(res.message || '更新失败')
        }
      } catch (e) {
        row.is_done = prev // 回滚
        this.$message.error(e.message || '更新失败')
      } finally {
        row._updating = false
      }
    },

    priorityType(val) {
      return val === 1 ? 'danger' : val === 2 ? 'warning' : 'info'
    },
    clean(obj) {
      const out = {}
      Object.keys(obj || {}).forEach((k) => {
        const v = obj[k]
        if (v !== undefined) out[k] = v // 保留 "" 与 null
      })
      return out
    }
  }
}
</script>

<style scoped>
.todos-card.two-third {
  width: 100%;
  min-width: 480px;
  display: inline-block;
  vertical-align: top;
  box-sizing: border-box;
}
@media (max-width: 1200px) {
  .todos-card.two-third {
    width: 100%;
    min-width: 360px;
  }
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 6px;
}
.title {
  font-weight: 600;
}
.ml8 {
  margin-left: 8px;
}
</style>
