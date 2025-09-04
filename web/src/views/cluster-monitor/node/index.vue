<template>
  <div class="page-container">
    <!-- 🔁 自动轮询（可见时刷新；集群切换重置定时器） -->
    <AutoPoll
      :key="currentId"
      :interval="10000"
      :visible-only="true"
      :immediate="false"
      :task="refresh"
    />

    <!-- 顶部卡片 -->
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

    <!-- 节点表格 -->
    <NodeTable
      :nodes="nodeList"
      @view="handleViewNode"
      @toggle="handleToggleSchedulable"
    />

    <!-- ▶️ 右侧抽屉：节点详情 -->
    <NodeDetailDrawer
      v-if="drawerVisible"
      v-loading="drawerLoading"
      :visible.sync="drawerVisible"
      :node="nodeDetail"
      width="45%"
      @close="drawerVisible = false"
    />
  </div>
</template>

<script>
import AutoPoll from '@/components/Atlhyper/AutoPoll.vue'
import CardStat from '@/components/Atlhyper/CardStat.vue'
import NodeTable from '@/components/Atlhyper/NodeTable.vue'
import NodeDetailDrawer from './NodeDescribe/NodeDetailDrawer.vue'
import {
  getNodeOverview,
  getNodeDetail,
  getNodecordon,
  getNodeuncordon
} from '@/api/node'
import { mapState } from 'vuex'

export default {
  name: 'NodeView',
  components: { AutoPoll, CardStat, NodeTable, NodeDetailDrawer },
  data() {
    return {
      stats: { totalNodes: 0, readyNodes: 0, totalCPU: 0, totalMemoryGB: 0 },
      nodeList: [],
      loading: false,
      drawerVisible: false,
      drawerLoading: false,
      nodeDetail: {},
      togglingNode: '' // 可选：记录进行中的节点名
    }
  },
  computed: {
    ...mapState('cluster', ['currentId'])
  },
  watch: {
    currentId: {
      immediate: true,
      handler(newId) {
        if (newId) this.refresh()
      }
    }
  },
  methods: {
    async refresh() {
      if (!this.currentId || this.loading) return
      await this.loadNodeData(this.currentId)
    },

    async loadNodeData(clusterId) {
      this.loading = true
      try {
        const res = await getNodeOverview(clusterId)
        if (res.code !== 20000) {
          this.$message.error('获取节点总览失败: ' + (res.message || ''))
          return
        }
        const payload = res.data || {}
        const cards = payload.cards || {}
        const rows = Array.isArray(payload.rows) ? payload.rows : []

        this.stats = {
          totalNodes: Number(cards.totalNodes ?? 0),
          readyNodes: Number(cards.readyNodes ?? 0),
          totalCPU: Number(cards.totalCPU ?? 0),
          totalMemoryGB: Number(cards.totalMemoryGiB ?? 0)
        }

        this.nodeList = rows.map((r) => ({
          name: r.name || '',
          ready: !!r.ready,
          internalIP: r.internalIP || '',
          osImage: r.osImage || '',
          architecture: r.architecture || '',
          cpu: Number(r.cpuCores ?? 0),
          memory: Number(r.memoryGiB ?? 0),
          schedulable: !!r.schedulable,
          unschedulable: !r.schedulable
        }))
      } catch (err) {
        this.$message.error('请求失败: ' + (err.message || err))
      } finally {
        this.loading = false
      }
    },

    // 查看详情
    async handleViewNode(row) {
      if (!this.currentId) {
        this.$message.error('未选择集群')
        return
      }
      const name = row.name
      if (!name) return

      this.drawerLoading = true
      try {
        const res = await getNodeDetail(this.currentId, name)
        if (res.code !== 20000) {
          this.$message.error(res.message || '获取节点详情失败')
          return
        }
        this.nodeDetail = res.data || {}
        this.drawerVisible = true
      } catch (e) {
        this.$message.error('获取节点详情失败：' + (e?.message || e))
      } finally {
        this.drawerLoading = false
      }
    },

    // ✅ 封锁/解封：根据当前可调度状态决定调哪个接口
    async handleToggleSchedulable(row) {
      if (!this.currentId) {
        this.$message.error('未选择集群')
        return
      }
      const node = row.name
      if (!node) return

      // 当前为不可调度 -> 解封；当前为可调度 -> 封锁
      const doUncordon = row.unschedulable === true
      const title = doUncordon ? '解封节点 (uncordon)' : '封锁节点 (cordon)'
      const msg = `确认对节点执行 ${title} 吗？\nNode: ${node}`

      try {
        await this.$confirm(msg, 'Confirm', {
          type: 'warning',
          confirmButtonText: doUncordon ? '解封' : '封锁',
          cancelButtonText: '取消',
          distinguishCancelAndClose: true
        })

        this.togglingNode = node
        const api = doUncordon ? getNodeuncordon : getNodecordon
        const res = await api(this.currentId, node)
        if (res.code !== 20000) {
          this.$message.error(res.message || `${title} 失败`)
          return
        }

        const cmd =
          res.data && res.data.commandID ? `（${res.data.commandID}）` : ''
        this.$message.success(`已下发 ${title} 命令 ${cmd}`)

        // 刷新列表，反映最新调度状态
        await this.refresh()
      } catch (e) {
        if (e !== 'cancel' && e !== 'close') {
          this.$message.error(`${title} 失败：` + (e?.message || e))
        }
      } finally {
        this.togglingNode = ''
      }
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
