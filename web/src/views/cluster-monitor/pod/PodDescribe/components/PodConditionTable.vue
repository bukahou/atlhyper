<template>
  <div class="card full-card">
    <div class="card-header text-center">
      <h4 class="card-title">Pod 状态条件</h4>
    </div>
    <div class="card-body">
      <table class="table">
        <thead>
          <tr>
            <th>类型</th>
            <th>状态</th>
            <th>变更时间</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(cond, index) in conditions" :key="index">
            <td>{{ cond.type }}</td>
            <td>
              <span
                class="status-pill"
                :class="{
                  True: cond.status === 'True',
                  False: cond.status === 'False',
                  Unknown: cond.status === 'Unknown',
                }"
              >
                {{ cond.status }}
              </span>
            </td>
            <td>{{ formatTime(cond.lastTransitionTime) }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script>
export default {
  name: "PodConditionTable",
  props: {
    conditions: {
      type: Array,
      required: true,
    },
  },
  methods: {
    formatTime(time) {
      return time ? new Date(time).toLocaleString() : "-";
    },
  },
};
</script>

<style scoped>
.full-card {
  width: 100%;
  max-width: 800px;
  min-height: 240px;
  border: 1px solid #d0d7de;
  border-radius: 10px;
  box-shadow: 0 4px 10px rgba(0, 0, 0, 0.05);
  margin: 16px auto;
  background-color: #ffffff;
  transition: all 0.3s ease;
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

table {
  width: 100%;
  border-collapse: collapse;
}

th,
td {
  padding: 10px 12px;
  border-bottom: 1px solid #ebeef5;
  font-size: 14px;
  color: #444;
  text-align: left;
}

th {
  background-color: #f9fafc;
  font-weight: 600;
  color: #666;
}

.status-pill {
  display: inline-block;
  padding: 2px 10px;
  border-radius: 20px;
  font-size: 13px;
  font-weight: 500;
  text-align: center;
  min-width: 60px;
}

.status-pill.True {
  background-color: #e7f9ed;
  color: #22a35f;
  border: 1px solid #a7e5b6;
}

.status-pill.False {
  background-color: #fff1f1;
  color: #e55353;
  border: 1px solid #f5b5b5;
}

.status-pill.Unknown {
  background-color: #fdf8e3;
  color: #c99813;
  border: 1px solid #f6dc92;
}
</style>
