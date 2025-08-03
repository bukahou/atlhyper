<template>
  <div class="card full-card">
    <div class="card-header text-center">
      <h4 class="card-title">{{ title }}</h4>
    </div>
    <div class="card-body">
      <div
        v-for="(item, index) in items"
        :key="index"
        class="info-row"
        :class="{
          'row-grey': index % 2 === 0,
          last: index === items.length - 1,
        }"
      >
        <span class="label">{{ item.label }}</span>
        <span class="value">{{ item.value || "-" }}</span>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: "InfoCard",
  props: {
    title: {
      type: String,
      required: true,
    },
    items: {
      type: Array,
      required: true,
    },
  },
};
</script>

<style scoped>
.full-card {
  width: 100%;
  max-width: 1600px;
  min-height: 240px;
  border: 1px solid #d0d7de;
  border-radius: 10px;
  box-shadow: 0 4px 10px rgba(0, 0, 0, 0.05);
  background-color: #ffffff;
  transition: all 0.3s ease;
  margin: 16px auto;
}

.card-header {
  background-color: #f0f9ff;
  border-bottom: 1px solid #d0ebff;
  padding: 12px 20px;
  border-top-left-radius: 10px;
  border-top-right-radius: 10px;
}

.card-title {
  font-size: 18px;
  font-weight: bold;
  color: #333;
  margin: 0;
}

.card-body {
  padding: 16px 20px;
}

/* ✅ 行样式：基础结构 */
.info-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 12px;
  border-bottom: 1px solid #ebeef5;
}

/* ✅ 最后一行不显示底部边框 */
.info-row.last {
  border-bottom: none;
}

/* ✅ 偶数行（0,2,4...）底色浅灰色 */
.row-grey {
  background-color: #f9f9f9;
}

/* ✅ 文本样式 */
.label {
  font-weight: 600;
  font-size: 14px;
  color: #666;
}

.value {
  font-weight: 500;
  font-size: 14px;
  color: #444;
  text-align: right;
  max-width: 60%;
  word-break: break-word;
}
</style>
