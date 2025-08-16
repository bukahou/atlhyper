<template>
  <div class="card stat-card" :style="{ '--accent': accentColor }">
    <div class="top">
      <div class="title-wrap">
        <div v-if="icon" class="icon-badge">
          <i :class="['el-icon', icon]" aria-hidden="true" />
        </div>
        <div class="title">{{ title }}</div>
      </div>
      <div v-if="subText" class="sub chip" :title="subText">{{ subText }}</div>
    </div>

    <div class="main">
      <div class="value" :title="valueText">{{ valueText }}</div>
    </div>

    <div v-if="progressPercent !== null" class="progress">
      <el-progress
        :percentage="progressPercent"
        :stroke-width="8"
        :show-text="false"
        :color="accentColor"
      />
    </div>
  </div>
</template>

<script>
export default {
  name: 'StatCard',
  props: {
    title: { type: String, required: true },
    value: { type: [String, Number], required: true },
    unit: { type: String, default: '' },
    subText: { type: String, default: '' }, // 如 '24h' 或 '6 / 6'
    icon: { type: String, default: '' }, // 如 'el-icon-bell'
    percent: { type: [Number, null], default: null },
    accent: { type: String, default: '#3B82F6' } // 主题色，可传 '#10B981' 等
  },
  computed: {
    valueText() {
      if (this.value === null || this.value === undefined || this.value === '') { return '--' }
      if (typeof this.value === 'number' && this.unit === '%') {
        return `${this.value.toFixed(2)}%`
      }
      return `${this.value}${this.unit}`
    },
    progressPercent() {
      if (this.percent === null || this.percent === undefined) return null
      const x = Number(this.percent)
      if (isNaN(x)) return 0
      return Math.max(0, Math.min(100, x))
    },
    accentColor() {
      // 简单兜底：传错就用默认蓝
      const s = String(this.accent || '').trim()
      return /^#|rgb|hsl/i.test(s) ? s : '#3B82F6'
    }
  }
}
</script>

<style scoped>
.card {
  background: #fff;
  border-radius: 14px;
  padding: 14px 16px;
  box-shadow: 0 1px 0 rgba(16, 24, 40, 0.02), 0 1px 2px rgba(16, 24, 40, 0.04);
  border: 1px solid #eef2f6;
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-height: 160px; /* 与 HealthCard 对齐 */
  height: 100%;
  transition: box-shadow 0.2s ease, transform 0.2s ease, border-color 0.2s ease;
}
.card:hover {
  box-shadow: 0 4px 14px rgba(16, 24, 40, 0.08);
  transform: translateY(-1px);
  border-color: #e6ebf0;
}

.top {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.title-wrap {
  display: flex;
  align-items: center;
  gap: 10px;
}
.icon-badge {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: #f1f5f9; /* 柔和底色 */
  display: inline-flex;
  align-items: center;
  justify-content: center;
  box-shadow: inset 0 0 0 1px #e5eaf0;
}
.icon-badge .el-icon {
  font-size: 16px;
  color: var(--accent, #3b82f6); /* 图标用主题色 */
}
.title {
  font-size: 15px;
  font-weight: 600;
  color: #0f172a;
  letter-spacing: 0.2px;
}

.chip {
  padding: 2px 8px;
  border-radius: 9999px;
  font-size: 12px;
  color: #475569;
  background: #f8fafc;
  border: 1px solid #eef2f7;
  line-height: 20px;
}

.main {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.value {
  font-size: 26px; /* 比 28 更收敛，排版更稳 */
  font-weight: 700;
  color: #0f172a;
  line-height: 1.1;
  letter-spacing: 0.2px;
  font-variant-numeric: tabular-nums;
  word-break: break-word;
}

/* 进度条：更细 & 圆角，颜色跟主题色联动 */
.progress {
  margin-top: 4px;
}
:deep(.el-progress-bar__outer) {
  border-radius: 999px;
}
:deep(.el-progress-bar__inner) {
  border-radius: 999px;
  background-color: var(--accent, #3b82f6) !important;
}
</style>
