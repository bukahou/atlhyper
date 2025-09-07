<template>
  <div class="page-container">
    <!-- 🔁 自动轮询（页面可见；集群切换重建定时器） -->
    <AutoPoll
      v-if="currentId"
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
        :number="stats.totalNamespaces"
        number-color="color1"
        title="Namespace 总数"
      >
        <template #icon><i class="fas fa-layer-group" /></template>
      </CardStat>
      <CardStat
        icon-bg="bg2"
        :number="stats.activeCount"
        number-color="color1"
        title="Active 数"
      >
        <template #icon><i class="fas fa-check" /></template>
      </CardStat>
      <CardStat
        icon-bg="bg3"
        :number="stats.terminating"
        number-color="color1"
        title="Terminating 数"
      >
        <template #icon><i class="fas fa-times" /></template>
      </CardStat>
      <CardStat
        icon-bg="bg4"
        :number="stats.totalPods"
        number-color="color1"
        title="总 Pod 数"
      >
        <template #icon><i class="fas fa-cube" /></template>
      </CardStat>
    </div>

    <!-- 表格 -->
    <NamespaceTable
      :namespaces="namespaceList"
      @view="handleViewNamespace"
      @configmap="handleViewConfigMap"
    />

    <!-- ▶️ 右侧抽屉：Namespace 详情 -->
    <NamespaceDetailDrawer
      v-if="drawerVisible"
      v-loading="drawerLoading"
      :visible.sync="drawerVisible"
      :ns="nsDetail"
      width="45%"
      @close="drawerVisible = false"
    />

    <!-- ▶️ ConfigMap 抽屉 -->
    <ConfigMapDrawer
      v-if="cmDrawerVisible"
      :visible.sync="cmDrawerVisible"
      :namespace="cmNsName"
      :items="cmList"
      :loading="cmLoading"
      width="60%"
    />
  </div>
</template>

<script>
import AutoPoll from '@/components/Atlhyper/AutoPoll.vue'
import CardStat from '@/components/Atlhyper/CardStat.vue'
import NamespaceTable from '@/components/Atlhyper/NamespaceTable.vue'
import NamespaceDetailDrawer from './NsDescribe/NamespaceDetailDrawer.vue'
import ConfigMapDrawer from './NsDescribe/ConfigMapDrawer.vue'
import {
  getAllNamespaces,
  getNamespacesDetail,
  getNamespacesConfigmap
} from '@/api/namespace'
import { mapState } from 'vuex'

export default {
  name: 'NamespaceView',
  components: {
    AutoPoll,
    CardStat,
    NamespaceTable,
    NamespaceDetailDrawer,
    ConfigMapDrawer
  },
  data() {
    return {
      stats: {
        totalNamespaces: 0,
        activeCount: 0,
        terminating: 0,
        totalPods: 0
      },
      namespaceList: [],
      loading: false,

      // NS 详情抽屉
      drawerVisible: false,
      drawerLoading: false,
      nsDetail: {},

      // ConfigMap 抽屉
      cmDrawerVisible: false,
      cmLoading: false,
      cmNsName: '',
      cmList: []
    }
  },
  computed: {
    ...mapState('cluster', ['currentId'])
  },
  watch: {
    currentId: {
      immediate: true,
      handler(id) {
        if (id) this.refresh()
      }
    }
  },
  methods: {
    // 轮询与首页加载
    async refresh() {
      if (!this.currentId || this.loading) return
      await this.loadNamespaces(this.currentId)
    },

    async loadNamespaces(clusterId) {
      if (!clusterId || this.loading) return
      this.loading = true
      try {
        const res = await getAllNamespaces(clusterId)
        if (res.code !== 20000) {
          this.$message.error(res.message || '命名空间概览获取失败')
          return
        }
        const { cards = {}, rows } = res.data || {}

        this.stats = {
          totalNamespaces: Number(cards.totalNamespaces ?? 0),
          activeCount: Number(cards.activeCount ?? 0),
          terminating: Number(cards.terminating ?? 0),
          totalPods: Number(cards.totalPods ?? 0)
        }

        const list = Array.isArray(rows) ? rows : []
        this.namespaceList = list.map((r) => ({
          name: r.name || '',
          status: r.status || 'Unknown',
          podCount: Number(r.podCount ?? 0),
          labelCount: Number(r.labelCount ?? 0),
          annotationCount: Number(r.annotationCount ?? 0),
          createdAt: r.createdAt || '',
          creationTime: this.formatTime(r.createdAt)
        }))
      } catch (err) {
        this.$message.error('请求失败：' + (err.message || err))
      } finally {
        this.loading = false
      }
    },

    // 查看 Namespace 详情
    async handleViewNamespace(row) {
      if (!this.currentId) {
        this.$message.error('未选择集群')
        return
      }
      const name = row.name
      if (!name) return

      this.drawerLoading = true
      try {
        const res = await getNamespacesDetail(this.currentId, name)
        if (res.code !== 20000) {
          this.$message.error(res.message || '获取命名空间详情失败')
          return
        }
        this.nsDetail = res.data || {}
        this.drawerVisible = true
      } catch (e) {
        this.$message.error('获取命名空间详情失败：' + (e?.message || e))
      } finally {
        this.drawerLoading = false
      }
    },

    // ▶️ 查看 Namespace 下的 ConfigMap 抽屉
    async handleViewConfigMap(row) {
      if (!this.currentId) return this.$message.error('未选择集群')
      const ns = row.name
      if (!ns) return

      this.cmNsName = ns
      this.cmLoading = true
      this.cmDrawerVisible = true // 先打开抽屉，loading 态
      try {
        const res = await getNamespacesConfigmap(this.currentId, ns)
        if (res.code !== 20000) {
          this.$message.error(res.message || '获取 ConfigMap 失败')
          this.cmList = []
          return
        }
        this.cmList = Array.isArray(res.data) ? res.data : []
      } catch (e) {
        this.$message.error('请求失败：' + (e.message || e))
        this.cmList = []
      } finally {
        this.cmLoading = false
      }
    },

    formatTime(iso) {
      const t = Date.parse(iso)
      if (!Number.isFinite(t)) return iso || '-'
      const d = new Date(t)
      const pad = (n) => String(n).padStart(2, '0')
      return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(
        d.getDate()
      )} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
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
