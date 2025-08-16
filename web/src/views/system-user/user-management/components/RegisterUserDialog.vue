<template>
  <el-dialog
    :visible.sync="visible"
    title="注册新用户"
    width="500px"
    :before-close="handleClose"
    class="register-dialog"
  >
    <el-form
      ref="registerForm"
      :model="form"
      :rules="rules"
      label-width="100px"
      autocomplete="off"
    >
      <el-form-item label="用户名" prop="username">
        <el-input
          v-model="form.username"
          autocomplete="new-username"
          placeholder="请输入用户名"
        />
      </el-form-item>

      <el-form-item label="密码" prop="password">
        <el-input
          v-model="form.password"
          type="password"
          placeholder="请输入密码"
          autocomplete="off"
        />
      </el-form-item>

      <el-form-item label="确认密码" prop="confirmPassword">
        <el-input
          v-model="form.confirmPassword"
          type="password"
          placeholder="请再次输入密码"
          autocomplete="off"
        />
      </el-form-item>

      <el-form-item label="昵称" prop="display_name">
        <el-input v-model="form.display_name" placeholder="请输入昵称" />
      </el-form-item>

      <el-form-item label="邮箱" prop="email">
        <el-input v-model="form.email" placeholder="请输入邮箱" />
      </el-form-item>

      <el-form-item label="角色" prop="role">
        <el-select v-model="form.role" placeholder="请选择角色">
          <el-option label="普通用户" :value="1" />
          <el-option label="管理员" :value="2" />
          <el-option label="超级管理员" :value="3" />
        </el-select>
      </el-form-item>
    </el-form>

    <div slot="footer" class="dialog-footer">
      <el-button @click="handleClose">取消</el-button>
      <el-button type="primary" @click="handleRegister">注册</el-button>
    </div>
  </el-dialog>
</template>

<script>
import { register } from '@/api/user' // ✅ 1. 引入注册 API
import { Message } from 'element-ui' // ✅ 2. 引入提示组件

export default {
  name: 'RegisterUserDialog',
  props: {
    visible: Boolean
  },
  data() {
    return {
      form: {
        username: '',
        password: '',
        confirmPassword: '',
        display_name: '',
        email: '',
        role: 1
      },
      rules: {
        username: [
          { required: true, message: '请输入用户名', trigger: 'blur' },
          {
            pattern: /^[A-Za-z]+$/,
            message: '用户名只能包含字母',
            trigger: 'blur'
          }
        ],
        password: [
          { required: true, message: '请输入密码', trigger: 'blur' },
          { min: 6, message: '密码长度不能少于 6 位', trigger: 'blur' }
        ],
        confirmPassword: [
          { required: true, message: '请确认密码', trigger: 'blur' },
          {
            validator: (rule, value, callback) => {
              if (value !== this.form.password) {
                callback(new Error('两次输入的密码不一致'))
              } else {
                callback()
              }
            },
            trigger: 'blur'
          }
        ],
        display_name: [
          { required: true, message: '请输入昵称', trigger: 'blur' }
        ],
        email: [
          { required: true, message: '请输入邮箱', trigger: 'blur' },
          { type: 'email', message: '邮箱格式不正确', trigger: 'blur' }
        ],
        role: [{ required: true, message: '请选择角色', trigger: 'change' }]
      }
    }
  },
  methods: {
    handleClose() {
      this.$emit('update:visible', false)
      this.resetForm()
    },
    resetForm() {
      this.$refs.registerForm.resetFields()
    },
    async handleRegister() {
      this.$refs.registerForm.validate(async(valid) => {
        if (!valid) return

        const { confirmPassword, ...userData } = this.form

        try {
          const res = await register(userData) // ✅ 3. 调用后端 API
          if (res.code === 20000) {
            Message.success('✅ 注册成功')
            this.$emit('register-success', res.data) // 可选：通知父组件刷新用户列表
            this.handleClose()
          } else {
            Message.error(res.message || '注册失败')
          }
        } catch (err) {
          Message.error('注册请求失败：' + (err.message || '未知错误'))
        }
      })
    }
  }
}
</script>

<style scoped>
/* ✅ 增加整体 padding */
.register-dialog >>> .el-dialog__body {
  padding: 24px 30px 12px 30px;
}

/* ✅ 表单项之间间距更一致 */
.register-dialog >>> .el-form-item {
  margin-bottom: 20px;
}

/* ✅ Label 字体增强 */
.register-dialog >>> .el-form-item__label {
  font-weight: 500;
  color: #333;
}

/* ✅ Footer 样式：按钮左偏移一些，更自然 */
.dialog-footer {
  padding: 10px 30px 20px 30px;
  display: flex;
  justify-content: flex-end;
}

.dialog-footer .el-button {
  margin-left: 16px;
  min-width: 80px;
}
</style>
