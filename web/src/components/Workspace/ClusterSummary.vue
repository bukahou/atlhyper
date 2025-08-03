<template>
  <div class="cluster-summary-card">
    <div class="header">
      <h3>展示集群概要</h3>
      <p class="subtext">Daily information about statistics in system</p>
    </div>
    <div class="summary-content">
      <!-- Ready Nodes -->
      <div :class="['circle-box', 'node-circle', nodesReady ? 'green' : 'red']">
        <div class="circle-text">{{ readyNodes }}</div>
        <div class="circle-label">Ready Nodes</div>
      </div>

      <!-- Ready Pods -->
      <div :class="['circle-box', 'pod-circle', podsReady ? 'green' : 'red']">
        <div class="circle-text">{{ readyPods }}</div>
        <div class="circle-label">Ready Pods</div>
      </div>

      <!-- Kubernetes Info -->
      <div class="info-box">
        <div class="version">v{{ k8sVersion }}</div>
        <div class="label">Kubernetes Version</div>
        <div class="metrics-status">
          <i class="el-icon-circle-check" style="color: green" />
          <span>Metrics OK</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'ClusterSummary',
  props: {
    readyNodes: {
      type: String,
      required: true
    },
    readyPods: {
      type: String,
      required: true
    },
    k8sVersion: {
      type: String,
      required: true
    }
  },
  computed: {
    nodesReady() {
      const [ready, total] = this.readyNodes.split('/').map(Number)
      return ready === total
    },
    podsReady() {
      const [ready, total] = this.readyPods.split('/').map(Number)
      return ready === total
    }
  }
}
</script>

<style scoped>
.cluster-summary-card {
  width: 720px;
  height: 250px;
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 8px;
  padding: 20px;
  box-sizing: border-box;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.05);
}

.header {
  margin-bottom: 16px;
}
.header h3 {
  margin: 0;
  font-size: 18px;
  color: #5f6166; /* ✅ 主文本色 */
}
.subtext {
  color: #909399;
  font-size: 13px;
}

.summary-content {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 130px;
  gap: 8px;
}

/* ✅ 三块区域统一占 1/3 宽度并内容居中 */
.circle-box,
.info-box {
  flex: 1;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}

/* ✅ 默认圆圈样式 */
.circle-text {
  font-size: 24px;
  font-weight: bold;
  border: 6px solid;
  border-radius: 50%;
  width: 90px;
  height: 90px;
  line-height: 82px;
  text-align: center;
  margin-bottom: 8px;
}

/* ✅ 节点圆圈颜色 */
.node-circle.green .circle-text {
  border-color: #8fe266;
  color: #67c23a;
}
.node-circle.red .circle-text {
  border-color: #5ae9ca;
  color: #f56c6c;
}

/* ✅ Pod 圆圈颜色 */
.pod-circle.green .circle-text {
  border-color: #47e6f1;
  color: #409eff;
}
.pod-circle.red .circle-text {
  border-color: #e6a23c;
  color: #e6a23c;
}
.version {
  font-size: 18px;
  font-weight: 600;
  color: #4d4f53; /* ✅ 主文本色 */
}

.circle-label {
  font-size: 14px;
  color: #606266; /* ✅ 次文本色 */
}

.label {
  font-size: 13px;
  color: #606266; /* ✅ 次文本色 */
}

.metrics-status {
  margin-top: 8px;
  font-size: 14px;
  color: green;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
}
</style>
