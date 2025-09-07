<template>
  <el-drawer
    :visible="visible"
    :size="width"
    :with-header="false"
    custom-class="cm-drawer"
    append-to-body
    :destroy-on-close="true"
    :close-on-click-modal="true"
    @close="$emit('update:visible', false)"
  >
    <!-- 顶部摘要栏 -->
    <div class="summary-bar">
      <div class="left">
        <span class="ns-name">{{ namespace }}</span>
        <el-tag size="mini" type="info">ConfigMaps {{ items.length }}</el-tag>
        <span v-if="activeItem" class="muted">当前：{{ activeItem.name }}</span>
      </div>
    </div>

    <!-- 主体 -->
    <div class="main">
      <!-- 左侧列表 -->
      <div class="sidenav">
        <div class="list-toolbar">
          <el-input
            v-model="q"
            size="mini"
            placeholder="搜索 ConfigMap 名称"
            clearable
          />
        </div>

        <el-menu :default-active="activeName" class="menu" @select="selectItem">
          <el-menu-item v-for="cm in filtered" :key="cm.name" :index="cm.name">
            <div class="menu-row">
              <span class="title" :title="cm.name">{{ cm.name }}</span>
              <span class="meta">{{ cm.keys }} keys</span>
            </div>
          </el-menu-item>
        </el-menu>

        <div v-if="!filtered.length && !loading" class="empty">无匹配项</div>
      </div>

      <!-- 右侧详情 -->
      <div v-loading="loading" class="content">
        <template v-if="activeItem">
          <section class="section">
            <h3 class="section-title">概览</h3>
            <div class="kv">
              <div>
                <span>名称</span><b>{{ activeItem.name }}</b>
              </div>
              <div>
                <span>命名空间</span><b>{{ activeItem.namespace }}</b>
              </div>
              <div>
                <span>创建时间</span><b>{{ activeItem.createdAt || "-" }}</b>
              </div>
              <div>
                <span>Age</span><b>{{ activeItem.age || "-" }}</b>
              </div>
              <div>
                <span>键数量</span><b>{{ activeItem.keys }}</b>
              </div>
              <div>
                <span>二进制键数量</span><b>{{ activeItem.binaryKeys }}</b>
              </div>
              <div>
                <span>合计大小</span><b>{{ fmtBytes(activeItem.totalSizeBytes) }}</b>
              </div>
              <div>
                <span>二进制合计大小</span><b>{{ fmtBytes(activeItem.binaryTotalSizeBytes) }}</b>
              </div>
            </div>
          </section>

          <section class="section">
            <h3 class="section-title">数据</h3>

            <el-table
              :data="activeItem.data || []"
              size="mini"
              border
              style="width: 100%"
              empty-text="No data"
            >
              <el-table-column prop="key" label="Key" min-width="180" />
              <el-table-column prop="size" label="Size(Bytes)" width="120" />
              <el-table-column label="Preview" min-width="300">
                <template slot-scope="{ row }">
                  <div class="preview">
                    <pre>{{ coalesce(row.preview, "(binary or empty)") }}</pre>
                    <span v-if="row.truncated" class="tag">... truncated</span>
                  </div>
                </template>
              </el-table-column>
              <el-table-column label="Actions" width="140" fixed="right">
                <template slot-scope="{ row }">
                  <el-button
                    type="text"
                    size="mini"
                    icon="el-icon-document-copy"
                    @click="copy(row.preview || '')"
                  >复制</el-button>
                  <el-button
                    type="text"
                    size="mini"
                    icon="el-icon-download"
                    @click="downloadKey(activeItem.name, row)"
                  >下载</el-button>
                </template>
              </el-table-column>
            </el-table>
          </section>

          <section class="section">
            <h3 class="section-title">YAML（快速导出）</h3>
            <el-button
              size="mini"
              icon="el-icon-download"
              @click="downloadAsYaml(activeItem)"
            >
              下载为 YAML
            </el-button>
          </section>
        </template>

        <div v-else-if="!loading" class="empty">请选择左侧一个 ConfigMap</div>
      </div>
    </div>
  </el-drawer>
</template>

<script>
export default {
  name: 'ConfigMapDrawer',
  props: {
    visible: { type: Boolean, default: false },
    namespace: { type: String, required: true },
    items: { type: Array, default: () => [] },
    loading: { type: Boolean, default: false },
    width: { type: String, default: '60%' }
  },
  data() {
    return {
      q: '',
      activeName: ''
    }
  },
  computed: {
    filtered() {
      const k = this.q.trim().toLowerCase()
      if (!k) return this.items
      return this.items.filter(
        (i) => i.name && i.name.toLowerCase().includes(k)
      )
    },
    activeItem() {
      if (!this.activeName) return null
      return this.items.find((x) => x.name === this.activeName) || null
    }
  },
  watch: {
    // 打开时默认选第一个
    visible(v) {
      if (v) {
        this.$nextTick(() => {
          if (!this.activeName && this.items && this.items.length) {
            this.activeName = this.items[0].name
          }
        })
      } else {
        // 关闭时可选择保留选择，也可以清空；这里保留
      }
    },
    // 数据变更后，若当前选中项不存在则回退为第一个
    items() {
      if (!this.items || !this.items.length) {
        this.activeName = ''
        return
      }
      if (!this.items.find((x) => x.name === this.activeName)) {
        this.activeName = this.items[0].name
      }
    }
  },
  methods: {
    selectItem(name) {
      this.activeName = name
    },
    coalesce(v, d) {
      return v == null || v === '' ? d : v
    },
    copy(text) {
      if (!text) {
        this.$message.info('内容为空')
        return
      }
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(text).then(() => {
          this.$message.success('已复制')
        })
      } else {
        const ta = document.createElement('textarea')
        ta.value = text
        document.body.appendChild(ta)
        ta.select()
        document.execCommand('copy')
        document.body.removeChild(ta)
        this.$message.success('已复制')
      }
    },
    downloadKey(cmName, row) {
      const content = row.preview || ''
      const blob = new Blob([content], { type: 'text/plain;charset=utf-8' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `${this.namespace}_${cmName}_${row.key}.txt`
      a.click()
      URL.revokeObjectURL(url)
    },
    downloadAsYaml(cm) {
      // 仅做快速导出：将 data 拼成简单 YAML（不处理复杂转义）
      const lines = []
      lines.push('apiVersion: v1')
      lines.push('kind: ConfigMap')
      lines.push(`metadata:`)
      lines.push(`  name: ${cm.name}`)
      lines.push(`  namespace: ${cm.namespace}`)
      lines.push(`data:`);
      (cm.data || []).forEach((kv) => {
        const key = kv.key
        const val = (kv.preview || '').replace(/\r?\n/g, '\n    ')
        // 多行使用 | 块标量，简单处理
        if (/\n/.test(kv.preview || '')) {
          lines.push(`  ${key}: |`)
          lines.push(`    ${val}`)
        } else {
          const safe = (kv.preview || '').replace(/"/g, '\\"')
          lines.push(`  ${key}: "${safe}"`)
        }
      })

      const blob = new Blob([lines.join('\n')], {
        type: 'text/yaml;charset=utf-8'
      })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `${this.namespace}_${cm.name}.yaml`
      a.click()
      URL.revokeObjectURL(url)
    },
    fmtBytes(n) {
      const v = Number(n) || 0
      const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB']
      let i = 0
      let x = v
      while (i < units.length - 1 && x >= 1024) {
        x /= 1024
        i++
      }
      const num = x < 10 ? x.toFixed(2) : x.toFixed(1)
      return `${num.replace(/\.0+$/, '').replace(/(\.\d)0$/, '$1')} ${
        units[i]
      }`
    }
  }
}
</script>

<style scoped>
.cm-drawer {
  overflow: hidden;
}

/* 顶部摘要栏 */
.summary-bar {
  position: sticky;
  top: 0;
  z-index: 2;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 16px;
  background: #fff;
  border-bottom: 1px solid #eee;
}
.summary-bar .left {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
}
.ns-name {
  font-weight: 600;
  font-size: 16px;
}
.muted {
  color: #666;
}

/* 主体布局 */
.main {
  display: flex;
  height: calc(100vh - 60px);
}

/* 左侧列表 */
.sidenav {
  width: 280px;
  border-right: 1px solid #f0f0f0;
  padding: 8px 0;
  background: #fafafa;
  display: flex;
  flex-direction: column;
}
.list-toolbar {
  padding: 8px;
}
.menu {
  border-right: none;
  flex: 1 1 auto;
  overflow: auto;
}
.menu-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
}
.menu-row .title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 180px;
}
.menu-row .meta {
  color: #909399;
  font-size: 12px;
}
.empty {
  color: #909399;
  text-align: center;
  padding: 12px;
}

/* 右侧详情 */
.content {
  flex: 1;
  overflow: auto;
  padding: 12px 16px;
}
.section {
  margin-bottom: 20px;
}
.section-title {
  font-weight: 600;
  margin: 4px 0 10px;
}
.kv > div {
  display: flex;
  justify-content: space-between;
  padding: 6px 0;
  border-bottom: 1px dashed #f0f0f0;
}
.kv > div:last-child {
  border-bottom: none;
}
.kv span {
  color: #666;
  margin-right: 12px;
}

.preview pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas,
    "Liberation Mono", monospace;
  font-size: 12px;
}
.preview .tag {
  color: #909399;
  margin-left: 8px;
}
</style>
