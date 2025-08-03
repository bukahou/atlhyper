<template>
  <div class="card full-card">
    <div class="card-header text-center">
      <h4 class="card-title">状态概览</h4>
    </div>
    <div class="card-body">
      <div class="info-row">
        <span class="label">状态</span>
        <span class="value">{{ status.phase || "-" }}</span>
      </div>
      <div class="info-row">
        <span class="label">启动时间</span>
        <span class="value">{{ formatTime(status.startTime) }}</span>
      </div>
      <div class="info-row">
        <span class="label">重启次数</span>
        <span class="value">
          {{
            (status.containerStatuses &&
              status.containerStatuses[0] &&
              status.containerStatuses[0].restartCount) ||
            0
          }}
        </span>
      </div>
      <div class="info-row">
        <span class="label">QoS 类别</span>
        <span class="value">{{ status.qosClass || "-" }}</span>
      </div>
      <div class="info-row">
        <span class="label">当前 CPU 使用</span>
        <span class="value">{{ usage.cpu || "N/A" }}</span>
      </div>
      <div class="info-row">
        <span class="label">当前内存使用</span>
        <span class="value">{{ usage.memory || "N/A" }}</span>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: "PodStatusCard",
  props: {
    status: {
      type: Object,
      required: true,
    },
    usage: {
      type: Object,
      default: () => ({}),
    },
  },
  methods: {
    formatTime(time) {
      if (!time) return "-";
      return new Date(time).toLocaleString();
    },
  },
};
</script>

<style scoped>
.full-card {
  width: 100%;
  max-width: 420px; /* 控制最大宽度 */
  min-height: 280px; /* 控制最小高度 */
  margin: 0 auto; /* 居中 */
  border: 1px solid #5682e9;
  border-radius: 8px;
  box-shadow: 0 2px 6px rgba(7, 7, 7, 0.678);
  transition: all 0.3s ease;
}

.card-header {
  background-color: #b4eee4;
  border-bottom: 1px solid #0ea8c4;
  padding: 1px 20px;
}
.card-title {
  font-size: 18px; /* 字号（默认通常是 16px） */
  font-weight: bold; /* 加粗 */
  color: #708d97; /* 深色字体 */
  margin-bottom: 12px; /* 可选：下方留白 */
}

.card-body {
  padding: 16px 20px;
}

.info-row {
  display: flex;
  justify-content: space-between;
  padding: 8px 0;
  border-bottom: 1px solid #999d9e;
}

.info-row:last-child {
  border-bottom: none;
}

.label {
  font-weight: 500;
  color: #464444;
}

.value {
  font-weight: 600;
  color: #b3aaaa;
}
</style>
