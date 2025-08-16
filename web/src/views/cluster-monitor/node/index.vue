<template>
  <div class="page-container">
    <!-- ✅ 顶部卡片区域 -->
    <div class="card-row">
      <CardStat
        icon-bg="bg1"
        :number="stats.totalNodes"
        number-color="color1"
        title="节点总数"
      >
        <template #icon><i class="fas fa-server" /></template>
      </CardStat>

      <CardStat
        icon-bg="bg2"
        :number="stats.readyNodes"
        number-color="color1"
        title="就绪节点"
      >
        <template #icon><i class="fas fa-check-circle" /></template>
      </CardStat>

      <CardStat
        icon-bg="bg3"
        :number="stats.totalCPU"
        number-color="color1"
        title="总 CPU（核）"
      >
        <template #icon><i class="fas fa-microchip" /></template>
      </CardStat>

      <CardStat
        icon-bg="bg4"
        :number="stats.totalMemoryGB"
        number-color="color1"
        title="总内存（GiB）"
      >
        <template #icon><i class="fas fa-memory" /></template>
      </CardStat>
    </div>

    <!-- ✅ 节点表格 -->
    <NodeTable
      :nodes="nodeList"
      @view="handleViewNode"
      @toggle="handleToggleSchedulable"
    />
  </div>
</template>

<script>
import CardStat from '@/components/Atlhyper/CardStat.vue'
import NodeTable from '@/components/Atlhyper/NodeTable.vue'
import { getNodeOverview, setNodeSchedulable } from '@/api/node' // ✅ 导入 API

export default {
  name: 'NodeView',
  components: {
    CardStat,
    NodeTable
  },
  data() {
    return {
      stats: {
        totalNodes: 0,
        readyNodes: 0,
        totalCPU: 0,
        totalMemoryGB: 0
      },
      nodeList: []
    }
  },
  mounted() {
    this.loadNodeData()
  },
  methods: {
    loadNodeData() {
      getNodeOverview()
        .then((res) => {
          if (res.code === 20000) {
            this.stats = res.data.stats
            this.nodeList = res.data.nodes
          } else {
            this.$message.error('获取节点总览失败: ' + res.message)
          }
        })
        .catch((err) => {
          this.$message.error('请求失败: ' + err.message)
        })
    },
    handleViewNode(row) {
      this.$message.info(`查看节点：${row.name}`)
    },
    handlePageChange(page) {
      this.currentPage = page
    },
    handlePageSizeChange(size) {
      this.pageSize = size
      this.currentPage = 1
    },
    toggleSchedulable(row) {
      // 发出自定义事件给父组件，让父组件决定是否调用 API
      this.$emit('toggle', row)
    },
    handleToggleSchedulable(row) {
      const isCurrentlyUnschedulable = row.unschedulable
      const next = !isCurrentlyUnschedulable // 发送反向值
      const action = isCurrentlyUnschedulable ? '解封' : '封锁'

      this.$confirm(`确认要${action}节点 ${row.name} 吗？`, '节点调度控制', {
        confirmButtonText: '确认',
        cancelButtonText: '取消',
        type: 'warning'
      })
        .then(() => {
          return setNodeSchedulable(row.name, next)
        })
        .then((res) => {
          if (res.code === 20000) {
            this.$message.success(res.message || `${action}成功`)
            this.loadNodeData()
          } else {
            this.$message.error(`${action}失败：${res.message}`)
          }
        })
        .catch((err) => {
          if (err !== 'cancel') {
            this.$message.error(`${action}失败：${err.message || err}`)
          }
        })
    },
    handleViewNode(row) {
      this.$router.push({
        name: 'NodeDescribe',
        params: { name: row.name }
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
