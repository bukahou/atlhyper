<template>
  <div>
    <el-form-item label="容器名">
      <el-input v-model="m.name" placeholder="不填默认 container" />
    </el-form-item>

    <el-form-item label="镜像" required>
      <el-input
        v-model="m.image"
        placeholder="如 nginx:1.25 或 ghcr.io/org/img:tag"
      />
    </el-form-item>

    <el-form-item label="拉取策略">
      <el-select v-model="m.pullPolicy" placeholder="可选">
        <el-option label="IfNotPresent" value="IfNotPresent" />
        <el-option label="Always" value="Always" />
        <el-option label="Never" value="Never" />
      </el-select>
    </el-form-item>
  </div>
</template>

<script>
export default {
  name: 'BasicFields',
  props: { value: { type: Object, default: () => ({}) }},
  data() {
    return { m: { name: '', image: '', pullPolicy: '', ...this.value }}
  },
  watch: {
    m: {
      deep: true,
      handler(v) {
        this.$emit('input', { ...v })
      }
    },
    value: {
      deep: true,
      handler(v) {
        this.m = { ...this.m, ...v }
      }
    }
  }
}
</script>
