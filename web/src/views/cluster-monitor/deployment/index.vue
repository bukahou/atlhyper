<template>
  <div class="page-container">
    <!-- ✅ 顶部卡片区域 -->
    <div class="card-row">
      <CardStat
        icon-bg="bg1"
        :number="stats.totalDeployments"
        number-color="color1"
        title="Deployment 总数"
      >
        <template #icon>
          <i class="fas fa-th-large" />
        </template>
      </CardStat>

      <CardStat
        icon-bg="bg2"
        :number="stats.uniqueNamespaces"
        number-color="color1"
        title="命名空间数"
      >
        <template #icon>
          <i class="fas fa-project-diagram" />
        </template>
      </CardStat>

      <CardStat
        icon-bg="bg3"
        :number="stats.totalReplicas"
        number-color="color1"
        title="总副本数"
      >
        <template #icon>
          <i class="fas fa-clone" />
        </template>
      </CardStat>

      <CardStat
        icon-bg="bg4"
        :number="stats.readyReplicas"
        number-color="color1"
        title="Ready 副本数"
      >
        <template #icon>
          <i class="fas fa-check-double" />
        </template>
      </CardStat>
    </div>

    <!-- ✅ Deployment 表格 -->
    <DeploymentTable
      :deployments="deploymentList"
      @view="handleViewDeployment"
      @update="handleUpdateDeployment"
    />
  </div>
</template>

<script>
import CardStat from '@/components/Atlhyper/CardStat.vue'
import DeploymentTable from '@/components/Atlhyper/DeploymentTable.vue'
import { getAllDeployments, updateDeployment } from '@/api/deployment'

export default {
  name: 'DeploymentView',
  components: {
    CardStat,
    DeploymentTable
  },
  data() {
    return {
      deploymentList: [],
      stats: {
        totalDeployments: 0,
        uniqueNamespaces: 0,
        totalReplicas: 0,
        readyReplicas: 0
      }
    }
  },
  created() {
    this.fetchDeployments()
  },
  methods: {
    fetchDeployments() {
      getAllDeployments()
        .then((res) => {
          const raw = res.data || []
          const nsSet = new Set()
          let totalReplicas = 0
          let readyReplicas = 0
          const list = []

          raw.forEach((d) => {
            const name = d.metadata?.name || '—'
            const namespace = d.metadata?.namespace || '—'
            const image = d.spec?.template?.spec?.containers?.[0]?.image || '—'
            const replicas = d.spec?.replicas ?? 0
            const ready = d.status?.readyReplicas ?? 0
            const labelCount =
              Object.keys(d.metadata?.labels || {}).length || 0
            const annotationCount =
              Object.keys(d.metadata?.annotations || {}).length || 0
            const creationTime = new Date(
              d.metadata?.creationTimestamp
            ).toLocaleString()

            nsSet.add(namespace)
            totalReplicas += replicas
            readyReplicas += ready

            list.push({
              name,
              namespace,
              image,
              replicas: `${ready}/${replicas}`,
              labelCount,
              annotationCount,
              creationTime
            })
          })

          this.deploymentList = list
          this.stats = {
            totalDeployments: raw.length,
            uniqueNamespaces: nsSet.size,
            totalReplicas,
            readyReplicas
          }
        })
        .catch((err) => {
          console.error('获取 Deployment 数据失败:', err)
          this.$message.error(
            '加载 Deployment 数据失败：' +
              (err.response?.data?.message || err.message)
          )
        })
    },
    handleViewDeployment(row) {
      this.$router.push({
        name: 'DeploymentDescribe',
        query: {
          ns: row.namespace,
          name: row.name
        }
      })
    },
    handleUpdateDeployment(payload) {
      updateDeployment(payload)
        .then((res) => {
          this.$message.success(res.message || '更新成功')
          this.fetchDeployments() // 刷新表格
        })
        .catch((err) => {
          this.$message.error('更新失败：' + err.message)
        })
    }
  }
}
</script>

<style scoped>
.page-container {
  padding-top: 35px;
  padding-left: 32px;
  padding-right: 32px;
}

.card-row {
  display: flex;
  flex-wrap: wrap;
  gap: 80px;
  margin-bottom: 24px;
}
</style>
