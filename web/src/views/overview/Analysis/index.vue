<template>
  <div class="page-container">
    <!-- ✅ 页面标题（可选） -->

    <!-- ✅ 集群概要卡片区域 -->
    <el-row :gutter="24" class="card-row">
      <el-col :xs="24" :sm="24" :md="24">
        <ClusterSummary
          :ready-nodes="clusterSummary.readyNodes"
          :ready-pods="clusterSummary.readyPods"
          :k8s-version="clusterSummary.k8sVersion"
        />
      </el-col>
    </el-row>

    <!-- 🔧 后续你可以继续添加更多卡片 -->
  </div>
</template>

<script>
import ClusterSummary from '@/components/Workspace/ClusterSummary.vue'

export default {
  name: 'AnalysisDashboard',
  components: {
    ClusterSummary
  },
  data() {
    return {
      clusterSummary: {
        readyNodes: '6/6',
        readyPods: '39/39',
        k8sVersion: '1.32.5+k3s1'
      }
    }
  }
}
</script>

<style scoped>
/* ✅ 页面整体边距控制 */
.page-container {
  padding: 30px 32px; /* 上下 30px，左右 32px */
}

/* ✅ 卡片区域样式 */
.card-row {
  margin-bottom: 24px;
}
</style>
