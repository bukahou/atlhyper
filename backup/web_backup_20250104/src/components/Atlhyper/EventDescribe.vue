<template>
  <div class="event-card">
    <div class="event-card-header">
      <h4 class="event-title">{{ title }}</h4>
    </div>
    <div class="event-card-body">
      <table class="event-table">
        <thead>
          <tr>
            <th>类型</th>
            <th>原因</th>
            <th>消息</th>
            <th class="right-align">时间</th>
          </tr>
        </thead>
        <tbody>
          <template v-if="events.length">
            <tr v-for="(event, index) in events" :key="index">
              <td>
                <span class="status-pill" :class="statusClass(event.type)">
                  {{ event.type }}
                </span>
              </td>
              <td>
                <span class="status-pill" :class="reasonClass(event.reason)">
                  {{ event.reason }}
                </span>
              </td>
              <td class="message-cell">{{ event.message }}</td>
              <td class="right-align">{{ formatTime(event.lastTimestamp) }}</td>
            </tr>
          </template>
          <tr v-else>
            <td colspan="4" class="no-event">暂无事件</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script>
export default {
  name: 'EventTable',
  props: {
    events: {
      type: Array,
      required: true
    },
    title: {
      type: String,
      default: '相关事件'
    },
    reasonMap: {
      type: Object,
      default: () => ({
        good: ['Started', 'Pulled', 'NodeSchedulable', 'NodeReady'],
        bad: [
          'Failed',
          'BackOff',
          'Unhealthy',
          'NodeNotSchedulable',
          'NodeNotReady',
          'KubeletNotReady'
        ]
      })
    }
  },
  methods: {
    formatTime(ts) {
      if (!ts) return '-'
      const date = new Date(ts)
      return `${date.getFullYear()}/${
        date.getMonth() + 1
      }/${date.getDate()} ${date.getHours().toString().padStart(2, '0')}:${date
        .getMinutes()
        .toString()
        .padStart(2, '0')}:${date.getSeconds().toString().padStart(2, '0')}`
    },
    statusClass(type) {
      if (type === 'Normal') return 'True'
      if (type === 'Warning') return 'False'
      return 'Unknown'
    },
    reasonClass(reason) {
      if (this.reasonMap.good.includes(reason)) return 'True'
      if (this.reasonMap.bad.includes(reason)) return 'False'
      return 'Unknown'
    }
  }
}
</script>

<style scoped>
/* 样式不变略 */
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

.message-cell {
  white-space: normal;
  word-break: break-word;
  max-width: 320px;
  line-height: 1.4;
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

/* 状态标签 */
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
