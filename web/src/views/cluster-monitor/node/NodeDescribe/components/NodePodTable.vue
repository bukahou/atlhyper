<template>
  <div class="event-card">
    <div class="event-card-header">
      <h4 class="event-title">{{ title }}</h4>
    </div>
    <div class="event-card-body">
      <table class="event-table">
        <thead>
          <tr>
            <th>Pod 名称</th>
            <th>命名空间</th>
            <th>容器数</th>
            <th>状态</th>
            <th>重启次数</th>
            <th class="right-align">启动时间</th>
          </tr>
        </thead>
        <tbody>
          <template v-if="pods.length">
            <tr v-for="(pod, index) in pods" :key="index">
              <td>{{ pod.name }}</td>
              <td>{{ pod.namespace }}</td>
              <td>{{ pod.containerCount }}</td>
              <td>
                <span class="status-pill" :class="statusClass(pod.status)">
                  {{ pod.status }}
                </span>
              </td>
              <td>{{ pod.restartCount }}</td>
              <td class="right-align">{{ formatTime(pod.startTime) }}</td>
            </tr>
          </template>
          <tr v-else>
            <td colspan="6" class="no-event">暂无运行中 Pod</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script>
export default {
  name: "PodTable",
  props: {
    pods: {
      type: Array,
      required: true,
    },
    title: {
      type: String,
      default: "运行中 Pod",
    },
  },
  methods: {
    formatTime(ts) {
      if (!ts) return "-";
      const date = new Date(ts);
      return `${date.getFullYear()}/${
        date.getMonth() + 1
      }/${date.getDate()} ${date.getHours().toString().padStart(2, "0")}:${date
        .getMinutes()
        .toString()
        .padStart(2, "0")}:${date.getSeconds().toString().padStart(2, "0")}`;
    },
    statusClass(status) {
      if (status === "Running") return "True";
      if (status === "Pending" || status === "Failed") return "False";
      return "Unknown";
    },
  },
};
</script>

<style scoped>
/* ✅ 复用 EventTable 样式 */
.event-card {
  width: 100%;
  max-width: 1500px;
  margin: 24px auto;
  background: #ffffff;
  border-radius: 12px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.06);
  transition: box-shadow 0.3s ease;
  overflow: hidden;
}

.event-card:hover {
  box-shadow: 0 6px 28px rgba(0, 0, 0, 0.1);
}

.event-card-header {
  background-color: #e6f4ff;
  padding: 16px 24px;
  border-bottom: 1px solid #d0e6fa;
}

.event-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: #1e3a8a;
  text-align: center;
}

.event-card-body {
  padding: 16px 20px;
  max-height: 360px;
  overflow-y: auto;
}

.event-table {
  width: 100%;
  border-collapse: collapse;
  table-layout: fixed;
}

.event-table th,
.event-table td {
  padding: 12px 14px;
  font-size: 14px;
  color: #333;
  border-bottom: 1px solid #f0f0f0;
  vertical-align: top;
}

.event-table th {
  background-color: #f5f8fa;
  font-weight: 600;
  color: #666;
  text-align: left;
}

.event-table tr:hover {
  background-color: #f3faff;
}

.right-align {
  text-align: right;
  color: #555;
}

.no-event {
  text-align: center;
  color: #aaa;
  font-style: italic;
  padding: 20px 0;
  background-color: #f9fafc;
}

.status-pill {
  display: inline-block;
  padding: 4px 12px;
  border-radius: 20px;
  font-size: 13px;
  font-weight: 500;
  min-width: 64px;
  text-align: center;
  border: 1px solid transparent;
}

.status-pill.True {
  background-color: #ecfdf5;
  color: #059669;
  border-color: #a7f3d0;
}

.status-pill.False {
  background-color: #fef2f2;
  color: #dc2626;
  border-color: #fecaca;
}

.status-pill.Unknown {
  background-color: #fff7ed;
  color: #d97706;
  border-color: #fde68a;
}
</style>
