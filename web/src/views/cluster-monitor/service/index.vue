<template>
  <div class="page-container">
    <!-- 🔁 自动轮询（页面可见时生效；集群切换重建定时器） -->
    <AutoPoll
      v-if="currentId"
      :key="currentId"
      :interval="10000"
      :visible-only="true"
      :immediate="false"
      :task="refresh"
    />

    <!-- ✅ 顶部卡片 -->
    <div class="card-row">
      <CardStat
        icon-bg="bg1"
        :number="stats.totalServices"
        number-color="color1"
        title="服务总数"
      >
        <template #icon><i class="fas fa-cubes" /></template>
      </CardStat>
      <CardStat
        icon-bg="bg2"
        :number="stats.externalServices"
        number-color="color1"
        title="外部服务"
      >
        <template #icon><i class="fas fa-globe" /></template>
      </CardStat>
      <CardStat
        icon-bg="bg3"
        :number="stats.internalServices"
        number-color="color1"
        title="内部服务"
      >
        <template #icon><i class="fas fa-network-wired" /></template>
      </CardStat>
      <CardStat
        icon-bg="bg4"
        :number="stats.headlessServices"
        number-color="color1"
        title="Headless 服务"
      >
        <template #icon><i class="fas fa-unlink" /></template>
      </CardStat>
    </div>

    <!-- ✅ 表格 -->
    <ServiceTable :services="serviceList" @view="handleViewService" />

    <!-- ▶️ 右侧抽屉：Service 详情 -->
    <ServiceDetailDrawer
      v-if="drawerVisible"
      v-loading="drawerLoading"
      :visible.sync="drawerVisible"
      :svc="serviceDetail"
      width="45%"
      @close="drawerVisible = false"
    />
  </div>
</template>

<script>
import AutoPoll from '@/components/Atlhyper/AutoPoll.vue'
import CardStat from '@/components/Atlhyper/CardStat.vue'
import ServiceTable from '@/components/Atlhyper/ServiceTable.vue'
import ServiceDetailDrawer from './ServiceDescribe/ServiceDetailDrawer.vue'
import { getAllServices, getServiceDetails } from '@/api/service'
import { mapState } from 'vuex'

export default {
  name: 'ServiceView',
  components: { AutoPoll, CardStat, ServiceTable, ServiceDetailDrawer },
  data() {
    return {
      serviceList: [],
      stats: {
        totalServices: 0,
        externalServices: 0,
        internalServices: 0,
        headlessServices: 0
      },
      loading: false,

      // 抽屉相关
      drawerVisible: false,
      drawerLoading: false,
      serviceDetail: {}
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
    // 🔁 轮询/首帧统一入口
    async refresh() {
      if (!this.currentId || this.loading) return
      await this.fetch(this.currentId)
    },

    async fetch(clusterId) {
      if (!clusterId || this.loading) return
      this.loading = true
      try {
        const res = await getAllServices(clusterId)
        if (res.code !== 20000) {
          this.$message.error(res.message || '获取 Service 概览失败')
          return
        }

        const { cards = {}, rows } = res.data || {}

        // 顶部 4 卡
        this.stats = {
          totalServices: Number(cards.totalServices ?? 0),
          externalServices: Number(cards.externalServices ?? 0),
          internalServices: Number(cards.internalServices ?? 0),
          headlessServices: Number(cards.headlessServices ?? 0)
        }

        // 表格数据
        const list = Array.isArray(rows) ? rows : []
        this.serviceList = list.map((r) => ({
          name: r.name || '',
          namespace: r.namespace || '',
          type: r.type || 'ClusterIP',
          clusterIP: r.clusterIP || 'None',
          ports: r.ports || '', // 若你的表格直接展示字符串
          protocol: r.protocol || '',
          selector: r.selector || '—',
          createdAt: r.createdAt || '',
          createTime: this.fmtTime(r.createdAt)
        }))
      } catch (err) {
        this.$message.error('请求失败：' + (err.message || err))
      } finally {
        this.loading = false
      }
    },

    // 🔎 查看 Service：拉详情并打开抽屉
    async handleViewService(row) {
      if (!this.currentId) {
        this.$message.error('未选择集群')
        return
      }
      const namespace = row.namespace
      const name = row.name
      if (!namespace || !name) {
        this.$message.warning('缺少 namespace/name')
        return
      }

      this.drawerLoading = true
      try {
        const res = await getServiceDetails(this.currentId, namespace, name)
        if (res.code !== 20000) {
          this.$message.error(res.message || '获取 Service 详情失败')
          return
        }
        this.serviceDetail = res.data || {}
        this.drawerVisible = true
      } catch (e) {
        this.$message.error('获取 Service 详情失败：' + (e?.message || e))
      } finally {
        this.drawerLoading = false
      }
    },

    fmtTime(ts) {
      if (!ts) return '-'
      const t = Date.parse(ts)
      if (!Number.isFinite(t)) return ts
      return new Date(t).toLocaleString()
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
