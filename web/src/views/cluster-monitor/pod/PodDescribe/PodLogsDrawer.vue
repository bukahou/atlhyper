<template>
  <el-drawer
    :visible="visible"
    :size="width"
    :with-header="false"
    custom-class="pod-logs-drawer"
    append-to-body
    :destroy-on-close="true"
    :close-on-click-modal="true"
    @open="onOpen"
    @close="handleClose"
  >
    <!-- 顶部摘要栏 -->
    <div class="summary-bar">
      <div class="left">
        <span class="pod-name">{{ podName }}</span>
        <el-tag size="mini" type="info">{{ namespace }}</el-tag>
        <el-tag size="mini">Tail {{ tailLines }} lines</el-tag>
      </div>
      <div class="right">
        <el-select
          v-model="tailLines"
          size="mini"
          style="width: 120px; margin-right: 8px"
          @change="fetchLogs"
        >
          <el-option
            v-for="n in [50, 100, 200, 500, 1000]"
            :key="n"
            :label="n + ' lines'"
            :value="n"
          />
        </el-select>
        <el-button
          size="mini"
          icon="el-icon-refresh"
          @click="fetchLogs"
        >刷新</el-button>
      </div>
    </div>

    <!-- 主体内容 -->
    <div class="main">
      <div class="content">
        <div class="log-toolbar">
          <el-input
            v-model="keyword"
            size="mini"
            clearable
            placeholder="过滤关键字（本地前端过滤）"
            style="width: 240px"
          />
          <el-button
            size="mini"
            icon="el-icon-document-copy"
            @click="copyLogs"
          >复制</el-button>
          <el-button
            size="mini"
            icon="el-icon-download"
            @click="downloadLogs"
          >下载</el-button>
        </div>

        <div ref="logBox" class="log-box">
          <pre class="log-pre" :class="{ dim: loading }">{{
            filteredLogs
          }}</pre>
        </div>
      </div>
    </div>
  </el-drawer>
</template>

<script>
import { getPodLogs } from '@/api/pod'

export default {
  name: 'PodLogsDrawer',
  props: {
    visible: { type: Boolean, default: false },
    clusterId: { type: [String, Number], required: true },
    namespace: { type: String, required: true },
    podName: { type: String, required: true },
    defaultTailLines: { type: Number, default: 50 }, // ✅ 改为 50
    width: { type: String, default: '60%' }
  },
  data() {
    return {
      logsStr: '',
      tailLines: this.defaultTailLines,
      loading: false,
      keyword: ''
    }
  },
  computed: {
    filteredLogs() {
      if (!this.keyword) return this.logsStr || ''
      const k = this.keyword.toLowerCase()
      return (this.logsStr || '')
        .split('\n')
        .filter((l) => l.toLowerCase().includes(k))
        .join('\n')
    }
  },
  watch: {
    // 抽屉在已打开状态下切换 ns/pod 时，自动刷新
    podName() {
      if (this.visible) this.fetchLogs()
    },
    namespace() {
      if (this.visible) this.fetchLogs()
    }
  },
  methods: {
    // ✅ 首次打开时拉取（解决首次看不到的问题）
    onOpen() {
      this.tailLines = this.defaultTailLines
      this.fetchLogs()
    },
    async fetchLogs() {
      if (!this.clusterId || !this.namespace || !this.podName) return
      this.loading = true
      try {
        const res = await getPodLogs(
          this.clusterId,
          this.namespace,
          this.podName,
          this.tailLines
        )
        if (res.code !== 20000) {
          this.$message.error(res.message || '获取 Pod 日志失败')
          return
        }
        this.logsStr = (res.data && res.data.logs) || ''
        this.$nextTick(() => this.scrollToBottom())
      } catch (e) {
        this.$message.error('请求失败：' + (e.message || e))
      } finally {
        this.loading = false
      }
    },
    scrollToBottom() {
      const el = this.$refs.logBox
      if (el) el.scrollTop = el.scrollHeight
    },
    copyLogs() {
      const text = this.filteredLogs || ''
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard
          .writeText(text)
          .then(() => this.$message.success('日志已复制'))
      } else {
        const ta = document.createElement('textarea')
        ta.value = text
        document.body.appendChild(ta)
        ta.select()
        document.execCommand('copy')
        document.body.removeChild(ta)
        this.$message.success('日志已复制')
      }
    },
    downloadLogs() {
      const blob = new Blob([this.filteredLogs || ''], {
        type: 'text/plain;charset=utf-8'
      })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      const ts = new Date().toISOString().replace(/[:.]/g, '-')
      a.download = `${this.namespace}_${this.podName}_${this.tailLines}lines_${ts}.log`
      a.click()
      URL.revokeObjectURL(url)
    },
    handleClose() {
      // 遮罩/关闭按钮：同步父组件 .sync，避免“关上又弹开”
      this.$emit('update:visible', false)
    }
  }
}
</script>

<style scoped>
.pod-logs-drawer ::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}
.pod-logs-drawer ::-webkit-scrollbar-thumb {
  background: #c0c4cc;
  border-radius: 4px;
}
.pod-logs-drawer ::-webkit-scrollbar-thumb:hover {
  background: #a8abb2;
}

.summary-bar {
  position: sticky;
  top: 0;
  z-index: 10;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 14px;
  background: #fff;
  border-bottom: 1px solid #ebeef5;
}
.summary-bar .left {
  display: flex;
  align-items: center;
  gap: 8px;
}
.pod-name {
  font-weight: 600;
  font-size: 14px;
  margin-right: 6px;
}

.main {
  display: flex;
  height: calc(100vh - 120px);
}
.content {
  flex: 1;
  padding: 12px 14px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.log-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}
.log-box {
  flex: 1;
  background: #0b1021;
  color: #cde2ff;
  border-radius: 6px;
  border: 1px solid #232a39;
  padding: 10px;
  overflow: auto;
  line-height: 1.4;
}
.log-pre {
  white-space: pre-wrap;
  word-break: break-word;
  margin: 0;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas,
    "Liberation Mono", monospace;
  font-size: 12px;
}
.dim {
  opacity: 0.6;
}
</style>
