<template>
  <div class="card full-card log-card">
    <div class="card-header text-center">
      <h4 class="card-title">Pod 内日志</h4>
    </div>
    <div class="card-body log-content" ref="logContent">
      <pre class="log-text">{{ logText || "暂无日志内容。" }}</pre>
    </div>
  </div>
</template>

<script>
export default {
  name: "PodLogCard",
  props: {
    logText: {
      type: String,
      default: "",
    },
  },
};
</script>

<style scoped>
.card.full-card.log-card {
  width: 100%;
  max-width: 1500px;
  min-height: 240px;
  border: 1px solid #d0d7de;
  border-radius: 10px;
  box-shadow: 0 6px 12px rgba(0, 0, 0, 0.05);
  background-color: #ffffff;
  margin: 20px auto;
  transition: all 0.3s ease;
  overflow: hidden;
}

.card-header {
  background-color: #f0f9ff;
  border-bottom: 1px solid #d0ebff;
  padding: 14px 24px;
  border-top-left-radius: 10px;
  border-top-right-radius: 10px;
}

.card-title {
  font-size: 18px;
  font-weight: bold;
  color: #2c3e50;
  margin: 0;
}

.log-content {
  max-height: 400px;
  overflow-y: auto;
  padding: 16px 20px;
  background-color: #f8f8f8;
}

.log-text {
  white-space: pre-wrap;
  word-break: break-word;
  font-family: Consolas, "Courier New", monospace;
  font-size: 13px;
  line-height: 1.6;
  color: #333;
}
</style>
