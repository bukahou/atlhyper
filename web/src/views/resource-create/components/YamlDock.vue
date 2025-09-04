<!-- resource-create/components/YamlDock.vue -->
<template>
  <el-card class="yaml-dock" shadow="never">
    <div class="header">
      <span class="title">YAML 预览</span>
      <div class="tools">
        <el-button
          size="mini"
          :disabled="!yaml"
          @click="copyYaml"
        >复制</el-button>
        <el-button
          size="mini"
          :disabled="!yaml"
          @click="downloadYaml"
        >下载</el-button>
      </div>
    </div>

    <div class="body" :style="{ height }">
      <template v-if="yaml && yaml.trim()">
        <pre class="code"><code>{{ yaml }}</code></pre>
      </template>
      <template v-else>
        <!-- 使用普通占位，不依赖 el-empty -->
        <div class="empty-wrap">
          <i class="el-icon-document" />
          <p class="empty-desc">
            暂无 YAML（请在左侧填写名称与镜像，点击“刷新 YAML”）
          </p>
        </div>
      </template>
    </div>

    <div v-if="showResults && results && results.length" class="results">
      <div
        v-for="(r, i) in results"
        :key="i"
        class="result"
        :class="r.type || 'info'"
      >
        <i v-if="r.type === 'success'" class="el-icon-success" />
        <i v-else-if="r.type === 'warning'" class="el-icon-warning-outline" />
        <i v-else-if="r.type === 'error'" class="el-icon-error" />
        <i v-else class="el-icon-info" />
        <span class="msg">{{ r.message }}</span>
      </div>
    </div>
  </el-card>
</template>

<script>
export default {
  name: 'YamlDock',
  props: {
    yaml: { type: String, default: '' },
    results: { type: Array, default: () => [] },
    height: { type: String, default: 'calc(100vh - 220px)' },
    showResults: { type: Boolean, default: true },
    filename: { type: String, default: 'resource.yaml' }
  },
  methods: {
    async copyYaml() {
      try {
        await navigator.clipboard.writeText(this.yaml || '')
        this.$message.success('已复制到剪贴板')
      } catch (e) {
        // 兼容不支持 clipboard 的环境
        const ta = document.createElement('textarea')
        ta.value = this.yaml || ''
        document.body.appendChild(ta)
        ta.select()
        document.execCommand('copy')
        document.body.removeChild(ta)
        this.$message.success('已复制到剪贴板')
      }
    },
    downloadYaml() {
      const blob = new Blob([this.yaml || ''], {
        type: 'text/yaml;charset=utf-8'
      })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = this.filename || 'resource.yaml'
      a.click()
      URL.revokeObjectURL(url)
    }
  }
}
</script>

<style scoped>
.yaml-dock {
  padding: 0;
}
.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 12px;
  border-bottom: 1px solid #ebeef5;
}
.title {
  font-weight: 600;
}
.tools :deep(.el-button) {
  margin-left: 6px;
}
.body {
  overflow: auto;
  padding: 12px;
  background: #0b1021; /* 深色背景更像代码面板 */
}
.code {
  margin: 0;
  white-space: pre;
  color: #e6e6e6;
  font-size: 12px;
  line-height: 1.6;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas,
    "Liberation Mono", "Courier New", monospace;
}

/* 空状态占位样式（替代 el-empty） */
.empty-wrap {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: #909399;
  padding: 36px 12px;
  text-align: center;
}
.empty-wrap i {
  font-size: 28px;
  margin-bottom: 8px;
}
.empty-desc {
  margin: 0;
}

.results {
  padding: 8px 12px;
  border-top: 1px solid #ebeef5;
}
.result {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 0;
}
.result.success {
  color: #67c23a;
}
.result.warning {
  color: #e6a23c;
}
.result.error {
  color: #f56c6c;
}
.result.info {
  color: #909399;
}
.msg {
  word-break: break-all;
}
</style>
