<template>
  <div class="step-basic">
    <el-form label-width="100px">
      <el-form-item label="名称" required>
        <el-input
          v-model="basic.name"
          placeholder="如：media"
          @blur="basic.name = sanitizeName(basic.name)"
        />
        <div class="hint">
          仅小写字母/数字/连字符；不能以连字符开头或结尾（DNS-1123）。
        </div>
      </el-form-item>

      <el-form-item label="命名空间">
        <el-input v-model="basic.namespace" placeholder="default（可选）" />
      </el-form-item>

      <el-form-item label="副本数">
        <el-input-number
          v-model.number="replicas"
          :min="1"
          :step="1"
          controls-position="right"
        />
        <span
          class="hint"
        >（用于 Deployment 的 <code>spec.replicas</code>）</span>
      </el-form-item>
    </el-form>
  </div>
</template>

<script>
import store from '../stores/createForm.store'

export default {
  name: 'BasicInfoStep',
  computed: {
    // 直接与全局 store 绑定
    basic: {
      get() {
        return store.form.basic
      },
      set(v) {
        store.form.basic = { ...(v || {}) }
      }
    },
    replicas: {
      get() {
        const n = Number(store.form.replicas)
        return Number.isFinite(n) && n > 0 ? n : 1
      },
      set(v) {
        const n = Number(v)
        store.form.replicas = Number.isFinite(n) && n > 0 ? n : 1
      }
    }
  },
  methods: {
    // 简易合法化：小写、保留 [a-z0-9-]，去首尾 -
    sanitizeName(s) {
      const v = String(s || '')
        .toLowerCase()
        .replace(/[^a-z0-9-]/g, '')
        .replace(/^-+/, '')
        .replace(/-+$/, '')
        .slice(0, 253)
      return v
    }
  }
}
</script>

<style scoped>
.hint {
  margin-top: 6px;
  color: #909399;
  font-size: 12px;
}
</style>
