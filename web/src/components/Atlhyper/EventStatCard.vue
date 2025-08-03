<template>
  <div class="event-summary">
    <h3 class="title">最近 1 天事件数</h3>
    <div class="event-grid">
      <div v-for="item in stats" :key="item.title" class="event-card">
        <div :class="['label', item.colorClass]">{{ item.title }}</div>
        <div class="count">{{ item.count }}</div>
      </div>
    </div>
  </div>
</template>

<script setup>
defineProps({
  stats: {
    type: Array,
    required: true
  }
})
</script>

<style scoped>
/* ✅ 容器整体尺寸设置 */
/* ✅ 容器整体样式（参考 .cluster-summary） */
.event-summary {
  width: 40%; /* 控制卡片占据父容器宽度的 40%，用于并排布局 */
  min-width: 650px; /* 设置最小宽度，防止在小屏下压缩变形 */
  height: 200px; /* 固定卡片高度，与 ClusterSummary 一致 */
  padding: 20px; /* 内边距，保持内容与边缘间距 */
  border-radius: 8px; /* 圆角边框，美观统一 */
  background-color: white; /* 白色背景，清晰简洁 */
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05); /* 添加阴影层次 */
  border-top: 5px solid #67c23a; /* 顶部绿色标识（可按主题颜色定制） */

  display: flex; /* Flex 布局，垂直排列标题和内容 */
  flex-direction: column; /* 垂直方向排列元素 */
  justify-content: space-between; /* 上下内容均匀分布占满高度 */
}

/* ✅ 标题样式 */
.title {
  font-size: 18px;
  font-weight: 600;
  margin-bottom: 10px;
}

/* ✅ 卡片排列网格 */
.event-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr); /* 一行两个 */
  gap: 10px;
}

/* ✅ 单个小卡片样式 */
.event-card {
  width: 60%; /* ✅ 设置卡片宽度为父容器的 60%，使其自适应调整大小 */
  height: 60px; /* ✅ 固定卡片高度，保持统一视觉尺寸 */
  min-width: 140px; /* ✅ 设置最小宽度，避免在过窄容器中被压缩过小 */
  text-align: center; /* ✅ 内容水平居中 */
  padding: 10px 57px; /* ✅ 上下内边距为 10px，左右为 57px，增加垂直空间 */
  display: flex; /* ✅ 使用 Flex 布局，使内部内容更易控制 */
  flex-direction: column; /* ✅ 主轴方向为垂直，元素上下排列 */
  justify-content: center; /* ✅ 垂直居中内容（label 和 count） */
  border-radius: 8px; /* ✅ 圆角样式，使卡片边缘更柔和 */
  background-color: #f9f9f9; /* ✅ 卡片背景色为浅灰白色，视觉柔和 */
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.05); /* ✅ 添加轻微阴影，提升卡片立体感 */
}

/* ✅ 文本样式 */
.label {
  font-size: 13px;
  font-weight: 500;
  margin-bottom: 6px;
  line-height: 1.2;
}

.count {
  font-size: 22px;
  font-weight: 600;
  line-height: 1.2;
  color: #333;
}

/* ✅ 颜色样式 */
.green-text {
  color: #67c23a;
}
.red-text {
  color: #f56c6c;
}
.orange-text {
  color: #e6a23c;
}
.blue-text {
  color: #409eff;
}
</style>
