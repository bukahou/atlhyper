<template>
  <div class="summary-card" :class="type">
    <div class="icon-section">
      <i :class="icon" class="summary-icon" />
    </div>
    <div class="text-section">
      <p class="label">{{ title }}</p>
      <h4 class="count">{{ count }}</h4>
    </div>
  </div>
</template>

<script>
export default {
  name: 'SummaryCard',
  props: {
    icon: {
      type: String,
      required: true
    },
    title: {
      type: String,
      required: true
    },
    count: {
      type: [Number, String],
      required: true
    },
    type: {
      type: String,
      default: 'default'
    }
  }
}
</script>

<style scoped>
.summary-card {
  display: flex;
  align-items: center;
  padding: 16px 20px;
  border-radius: 10px;
  background-color: #fff;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
  margin-bottom: 16px;
  transition: all 0.2s ease;
}
.summary-card:hover {
  transform: translateY(-2px);
}

.icon-section {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 16px;
}
.summary-icon {
  font-size: 24px;
  color: #fff;
}

/* 文字 */
.text-section .label {
  font-size: 14px;
  margin: 0;
  color: #888;
}
.text-section .count {
  font-size: 20px;
  font-weight: bold;
  margin: 4px 0 0;
}

/* 颜色风格 */
.summary-card.success .icon-section {
  background-color: #28a745;
}
.summary-card.info .icon-section {
  background-color: #17a2b8;
}
.summary-card.danger .icon-section {
  background-color: #dc3545;
}
.summary-card.warning .icon-section {
  background-color: #ffc107;
}
.summary-card.default .icon-section {
  background-color: #6c757d;
}
</style>
