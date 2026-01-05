<template>
  <div class="cluster-summary">
    <!-- 标题左上角展示 -->
    <div class="cluster-summary-header">
      <h3 class="title">展示集群概要</h3>
      <p class="subtitle">Daily information about statistics in system</p>
    </div>

    <!-- 核心统计区域 -->
    <div class="summary-content">
      <div class="stat-box">
        <div class="circle green">
          <span class="value">{{ readyNodes }}</span>
        </div>
        <div class="label">Ready Nodes</div>
      </div>

      <div class="stat-box">
        <div class="circle red">
          <span class="value">{{ readyPods }}</span>
        </div>
        <div class="label">Ready Pods</div>
      </div>

      <div class="stat-box right">
        <div class="version">{{ k8sVersion }}</div>
        <div class="label">Kubernetes Version</div>
        <div class="metrics">
          <el-icon style="color: green"><Check /></el-icon> Metrics OK
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { Check } from '@element-plus/icons-vue'

// 使用 JS 写法定义 props（不需要类型）
defineProps({
  readyNodes: String,
  readyPods: String,
  k8sVersion: String
})
</script>

<style scoped>
.cluster-summary {
  width: 40%; /* 控制卡片占据父容器宽度的 40%，便于并排布局 */
  min-width: 650px; /* 设置卡片最小宽度，防止在较窄屏幕下压缩变形 */
  height: 200px; /* 固定卡片高度，保持各卡片高度一致，视觉统一 */
  padding: 20px; /* 内边距，保持内容与边缘间距，增加可读性 */
  border-radius: 8px; /* 设置圆角，使卡片边缘更柔和，美观 */
  background-color: white; /* 背景颜色为白色，符合仪表盘清晰风格 */
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05); /* 添加轻微阴影，突出卡片层次感 */
  border-top: 5px solid #409eff; /* 顶部边框突出标识，使用主题蓝色强调 */

  display: flex; /* 使用 flexbox 布局，方便垂直内容对齐 */
  flex-direction: column; /* 设置为垂直方向布局：标题在上，内容在下 */
  justify-content: space-between; /* 上下内容分布占满整个卡片高度 */
}

/* 标题区域左对齐 */
.cluster-summary-header {
  text-align: left;
  margin-bottom: 20px;
}

.title {
  font-size: 18px;
  font-weight: 600;
  margin-bottom: 4px;
}

.subtitle {
  font-size: 13px;
  color: #888;
}

/* 核心内容居中展示 */
.summary-content {
  display: flex;
  justify-content: space-around;
  align-items: center;
  flex-wrap: wrap;
}

.stat-box {
  text-align: center;
}

.stat-box.right {
  text-align: right;
}

.circle {
  width: 72px;
  height: 72px;
  border-radius: 50%;
  line-height: 72px;
  font-size: 20px;
  font-weight: 600;
  margin: 0 auto 8px auto;
  border: 4px solid;
}

.green {
  border-color: #67c23a;
  color: #67c23a;
}

.red {
  border-color: #f56c6c;
  color: #f56c6c;
}

.label {
  font-size: 14px;
  color: #333;
}

.version {
  font-size: 16px;
  font-weight: 600;
  color: #333;
}

.metrics {
  font-size: 14px;
  margin-top: 4px;
  color: #2ecc71;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 4px;
}
</style>
