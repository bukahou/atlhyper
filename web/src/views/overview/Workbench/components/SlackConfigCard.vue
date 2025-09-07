<template>
  <!-- 关键：增加 one-third 类 -->
  <el-card class="slack-config-card compact one-third" shadow="never">
    <div class="card-header">
      <div class="title">
        <span class="dot" :class="form.Enable ? 'dot--on' : 'dot--off'" />
        Slack 通知配置
      </div>
      <el-tag size="mini" type="info">{{ form.Name || "slack" }}</el-tag>
    </div>

    <el-form
      :model="form"
      label-width="88px"
      size="small"
      class="dense-form"
      @submit.native.prevent
    >
      <!-- 启用（单字段提交） -->
      <el-form-item label="启用">
        <div class="inline">
          <el-switch
            v-model="form.Enable"
            :active-value="1"
            :inactive-value="0"
            :disabled="savingEnable || loading"
            class="switch-compact"
            @change="onToggleEnable"
          />
          <span class="hint">未配置 Webhook 时即使启用也不会发送</span>
        </div>
      </el-form-item>

      <!-- Webhook：默认仅展示；点“修改”后可编辑，确认前弹窗 -->
      <el-form-item label="Webhook">
        <div class="row">
          <template v-if="!editingWebhook">
            <span class="mono ellipsis webhook-text">
              {{ form.Webhook || "（未配置）" }}
            </span>
            <el-button
              class="ml8"
              size="mini"
              :loading="savingWebhook"
              @click="startEditWebhook"
            >修改</el-button>
          </template>
          <template v-else>
            <el-input
              v-model.trim="webhookDraft"
              placeholder="https://hooks.slack.com/services/..."
              clearable
              class="w-360"
              :disabled="savingWebhook"
            />
            <el-button
              class="ml8"
              type="primary"
              size="mini"
              :loading="savingWebhook"
              @click="confirmEditWebhook"
            >确认</el-button>
            <el-button
              size="mini"
              :disabled="savingWebhook"
              @click="cancelEditWebhook"
            >取消</el-button>
          </template>
        </div>
      </el-form-item>

      <!-- 发送间隔（只展示） -->
      <el-form-item label="发送间隔">
        <span class="kv">{{ (form.IntervalSec || 5) + " 秒" }}</span>
      </el-form-item>
    </el-form>

    <div class="footer">
      <span>最后更新：{{ formatTime(form.UpdatedAt) }}</span>
      <el-tag
        v-if="!form.Webhook"
        size="mini"
        type="warning"
        class="ml8"
      >未配置 Webhook</el-tag>
    </div>
  </el-card>
</template>

<script>
import { getSlackConfig, updateSlackConfig } from '@/api/workbench'

export default {
  name: 'SlackConfigCard',
  data() {
    return {
      loading: false,
      savingEnable: false,
      savingWebhook: false,
      editingWebhook: false,
      webhookDraft: '',
      form: {
        ID: null,
        Name: 'slack',
        Enable: 0,
        Webhook: '',
        IntervalSec: 5,
        UpdatedAt: ''
      }
    }
  },
  created() {
    this.fetchConfig()
  },
  methods: {
    async fetchConfig() {
      this.loading = true
      try {
        const res = await getSlackConfig()
        const { code, data, message } = res || {}
        if (code === 20000 && data) { this.form = Object.assign({}, this.form, data) } else this.$message && this.$message.error(message || '读取失败')
      } catch (e) {
        this.$message && this.$message.error(e.message || '读取异常')
      } finally {
        this.loading = false
      }
    },
    async onToggleEnable(val) {
      const actionText = val ? '启用' : '关闭'
      try {
        await this.$confirm(`确定${actionText} Slack 告警吗？`, '确认操作', {
          type: 'warning',
          confirmButtonText: '确定',
          cancelButtonText: '取消'
        })
      } catch (_) {
        this.form.Enable = val ? 0 : 1
        return
      }
      try {
        this.savingEnable = true
        const res = await updateSlackConfig({ enable: val })
        this.afterSave(res)
      } catch (e) {
        this.$message && this.$message.error(e.message || '更新失败')
        this.form.Enable = val ? 0 : 1
      } finally {
        this.savingEnable = false
      }
    },
    startEditWebhook() {
      this.webhookDraft = this.form.Webhook || ''
      this.editingWebhook = true
    },
    cancelEditWebhook() {
      this.editingWebhook = false
      this.webhookDraft = ''
    },
    async confirmEditWebhook() {
      const newVal = (this.webhookDraft || '').trim()
      if (newVal === (this.form.Webhook || '')) {
        this.$message && this.$message.info('Webhook 未变化')
        this.editingWebhook = false
        return
      }
      const preview = newVal
        ? newVal.length > 38
          ? newVal.slice(0, 18) + '...' + newVal.slice(-15)
          : newVal
        : '(清空)'
      try {
        await this.$confirm(
          `确定将 Webhook 更新为：\n${preview} ？\n（提示：清空将暂停发送）`,
          '确认修改 Webhook',
          { type: 'warning' }
        )
      } catch (_) {
        return
      }
      try {
        this.savingWebhook = true
        const res = await updateSlackConfig({ webhook: newVal })
        this.afterSave(res)
        this.editingWebhook = false
      } catch (e) {
        this.$message && this.$message.error(e.message || '保存失败')
      } finally {
        this.savingWebhook = false
      }
    },
    afterSave(res) {
      const { code, data, message } = res || {}
      if (code === 20000) {
        this.$message && this.$message.success(message || '已保存')
        if (data) this.form = Object.assign({}, this.form, data)
      } else {
        this.$message && this.$message.error(message || '操作失败')
      }
    },
    formatTime(s) {
      if (!s) return '-'
      try {
        return s.replace('T', ' ').replace('+', ' GMT+')
      } catch (_) {
        return s
      }
    }
  }
}
</script>

<style scoped>
/* ——长度控制：让卡片约占父容器 1/3 宽—— */
.slack-config-card.one-third {
  width: 33.333%;
  min-width: 360px; /* 防止过窄，按需调整 */
  display: inline-block; /* 允许多卡片并排 */
  vertical-align: top;
  box-sizing: border-box;
}
/* 中屏回退到 1/2 宽 */
@media (max-width: 1200px) {
  .slack-config-card.one-third {
    width: 50%;
    min-width: 320px;
  }
}
/* 小屏回退到 100% 宽 */
@media (max-width: 768px) {
  .slack-config-card.one-third {
    width: 100%;
    min-width: 0;
  }
}

/* 卡片更紧凑 */
.slack-config-card.compact,
.slack-config-card {
  --pad-x: 14px;
  --pad-y: 12px;
  --radius: 10px;
  --font: 13px;
}
.slack-config-card :deep(.el-card__body) {
  padding: var(--pad-y) var(--pad-x);
}
.slack-config-card {
  margin-top: 12px;
  border-radius: var(--radius);
  font-size: var(--font);
}

/* 头部 */
.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 6px;
}
.title {
  font-weight: 600;
  display: flex;
  align-items: center;
}
.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  margin-right: 8px;
  background: #dcdfe6;
}
.dot--on {
  background: #67c23a;
}
.dot--off {
  background: #f56c6c;
}

/* 表单更紧凑 */
.dense-form :deep(.el-form-item) {
  margin-bottom: 8px;
}
.dense-form :deep(.el-form-item__label) {
  color: #606266;
  padding-right: 8px;
}
.inline {
  display: flex;
  align-items: center;
}
.switch-compact {
  transform: scale(0.92);
  transform-origin: left center;
}

/* 文本样式 */
.row {
  display: flex;
  align-items: center;
}
.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas,
    "Liberation Mono", monospace;
}
.ellipsis {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.webhook-text {
  color: #303133;
  background: #f6f8fa;
  border: 1px solid #ebeef5;
  padding: 3px 6px;
  border-radius: 6px;
}

/* 底部 */
.footer {
  margin-top: 12px;
  padding-top: 10px;
  border-top: 1px dashed #ebeef5;
  font-size: 12px;
  color: #909399;
  display: flex;
  align-items: center;
}

/* 细节 */
.ml8 {
  margin-left: 8px;
}
.w-360 {
  width: 360px;
  max-width: 100%;
}
.hint {
  margin-left: 8px;
  color: #a8abb2;
  font-size: 12px;
}
.kv {
  color: #303133;
}
</style>
