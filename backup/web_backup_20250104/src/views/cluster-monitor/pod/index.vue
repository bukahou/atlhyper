<template>
  <div class="page-container">
    <!-- 轮询 -->
    <AutoPoll
      :key="currentId"
      :interval="5000"
      :visible-only="true"
      :immediate="false"
      :task="refreshAll"
    />

    <!-- 顶部状态卡片 -->
    <div class="card-row">
      <CardStat
        v-for="(item, index) in podStats"
        :key="index"
        :icon-bg="item.iconBg"
        :number="item.count"
        :number-color="item.numberColor"
        :title="item.title"
      >
        <template #icon><i :class="item.iconClass" /></template>
      </CardStat>
    </div>

    <!-- 表格 -->
    <PodTable
      :pods="podList"
      @restart="handleRestartPod"
      @view="handleViewPod"
      @logs="handleViewLogs"
    />

    <!-- 详情抽屉 -->
    <PodDetailDrawer
      v-if="drawerVisible"
      v-loading="drawerLoading"
      :visible.sync="drawerVisible"
      :pod="podDetail"
      width="45%"
      @close="drawerVisible = false"
    />

    <!-- 日志抽屉 -->
    <PodLogsDrawer
      v-if="logsDrawerVisible"
      :visible.sync="logsDrawerVisible"
      :cluster-id="currentId"
      :namespace="logsNs"
      :pod-name="logsPod"
      :default-tail-lines="50"
      width="60%"
    />
  </div>
</template>

<script>
import AutoPoll from '@/components/Atlhyper/AutoPoll.vue'
import CardStat from '@/components/Atlhyper/CardStat.vue'
import PodTable from '@/components/Atlhyper/PodTable.vue'
import PodDetailDrawer from './PodDescribe/PodDetailDrawer.vue'
import PodLogsDrawer from './PodDescribe/PodLogsDrawer.vue'
import { getPodOverview, getPodDetail, getPodRestart } from '@/api/pod' // ✅ 引入重启API
import { mapState } from 'vuex'

export default {
  name: 'PodPage',
  components: { AutoPoll, CardStat, PodTable, PodDetailDrawer, PodLogsDrawer },
  data() {
    return {
      podStats: [],
      podList: [],
      loading: false,
      // 详情
      drawerVisible: false,
      drawerLoading: false,
      podDetail: {},
      // 日志
      logsDrawerVisible: false,
      logsNs: '',
      logsPod: '',
      logsTailLines: 200,
      // 重启中的 Pod（可用于后续加禁用/loading状态）
      restartingKey: ''
    }
  },
  computed: {
    ...mapState('cluster', ['currentId'])
  },
  watch: {
    currentId: {
      immediate: true,
      handler(id) {
        if (id) this.refreshAll()
      }
    }
  },
  methods: {
    async refreshAll() {
      if (!this.currentId || this.loading) return
      await this.loadOverview()
    },
    async loadOverview() {
      this.loading = true
      try {
        const res = await getPodOverview(this.currentId)
        if (res.code !== 20000) {
          this.$message.error(res.message || '获取 Pod 概览失败')
          return
        }
        const { cards = {}, pods = [] } = res.data || {}
        this.podStats = this.adaptPodCards(cards)
        this.podList = pods.map(this.adaptPodRow)
      } catch (e) {
        this.$message.error('获取 Pod 概览失败：' + (e?.message || e))
      } finally {
        this.loading = false
      }
    },
    adaptPodCards(cards) {
      const n = (v) => (Number.isFinite(Number(v)) ? Number(v) : 0)
      return [
        {
          title: 'Running',
          count: n(cards.running),
          iconClass: 'fas fa-play-circle',
          iconBg: 'bg2',
          numberColor: 'color1'
        },
        {
          title: 'Pending',
          count: n(cards.pending),
          iconClass: 'fas fa-hourglass-half',
          iconBg: 'bg3',
          numberColor: 'color1'
        },
        {
          title: 'Failed',
          count: n(cards.failed),
          iconClass: 'fas fa-times-circle',
          iconBg: 'bg4',
          numberColor: 'color1'
        },
        {
          title: 'Unknown',
          count: n(cards.unknown),
          iconClass: 'fas fa-question-circle',
          iconBg: 'bg1',
          numberColor: 'color1'
        }
      ]
    },
    adaptPodRow(p) {
      let ready = false
      const m = String(p.ready || '').match(/^(\d+)\s*\/\s*(\d+)$/)
      if (m) {
        const a = +m[1]
        const b = +m[2]
        ready = Number.isFinite(a) && Number.isFinite(b) && b > 0 && a === b
      }
      const pct = (v) =>
        Number.isFinite(Number(v)) ? `${Number(v).toFixed(1)}%` : '-'
      return {
        namespace: p.namespace || '',
        deployment: p.deployment || '',
        name: p.name || '',
        ready,
        phase: p.phase || '-',
        restartCount: Number(p.restarts ?? p.restartCount ?? 0),
        cpuUsage: p.cpuText ?? '-',
        cpuUsagePercent: pct(p.cpuPercent),
        memoryUsage: p.memoryText ?? '-',
        memoryPercent: pct(p.memPercent),
        startTime: p.startTime || '',
        nodeName: p.node || p.nodeName || ''
      }
    },

    // ▶️ 重启 Pod（删除当前 Pod，让上层控制器重建）
    async handleRestartPod(row) {
      if (!this.currentId) {
        this.$message.error('未选择集群')
        return
      }
      try {
        await this.$confirm(
          `确认重启该 Pod 吗？\nNamespace: ${row.namespace}\nPod: ${row.name}\n\n该操作会删除当前 Pod，由 Deployment/ReplicaSet 重新创建。`,
          'Confirm',
          {
            type: 'warning',
            confirmButtonText: '重启',
            cancelButtonText: '取消',
            distinguishCancelAndClose: true
          }
        )

        this.restartingKey = `${row.namespace}/${row.name}`
        const res = await getPodRestart(
          this.currentId,
          row.namespace,
          row.name
        )
        if (res.code !== 20000) {
          this.$message.error(res.message || '重启失败')
          return
        }

        const cmd =
          res.data && res.data.commandID ? `（${res.data.commandID}）` : ''
        this.$message.success(`已下发重启命令${cmd}`)
        // 刷新一次列表以反映重启前后的状态/计数
        await this.loadOverview()
      } catch (e) {
        // 用户取消不提示为错误
        if (e !== 'cancel' && e !== 'close') {
          this.$message.error('重启失败：' + (e?.message || e))
        }
      } finally {
        this.restartingKey = ''
      }
    },

    async handleViewPod(row) {
      if (!this.currentId) {
        this.$message.error('未选择集群')
        return
      }
      this.drawerLoading = true
      try {
        const res = await getPodDetail(this.currentId, row.namespace, row.name)
        if (res.code !== 20000) {
          this.$message.error(res.message || '获取 Pod 详情失败')
          return
        }
        this.podDetail = res.data || {}
        this.drawerVisible = true
      } catch (e) {
        this.$message.error('获取 Pod 详情失败：' + (e?.message || e))
      } finally {
        this.drawerLoading = false
      }
    },

    handleViewLogs(row) {
      this.logsNs = row.namespace
      this.logsPod = row.name
      this.logsTailLines = 200
      this.logsDrawerVisible = true
    }
  }
}
</script>

<style scoped>
.page-container {
  padding: 35px 32px;
}
.card-row {
  display: flex;
  flex-wrap: wrap;
  gap: 80px;
  margin-bottom: 24px;
}
</style>

<style scoped>
.page-container {
  padding: 35px 32px;
}

.card-row {
  display: flex;
  flex-wrap: wrap;
  gap: 80px;
  margin-bottom: 24px;
}
</style>
