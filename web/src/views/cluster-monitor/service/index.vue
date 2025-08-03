<template>
  <div class="page-container">
    <!-- ✅ 顶部卡片区域 -->
    <div class="card-row">
      <CardStat
        icon-bg="bg1"
        :number="stats.totalServices"
        number-color="color1"
        title="服务总数"
      >
        <template #icon>
          <i class="fas fa-cubes" />
        </template>
      </CardStat>

      <CardStat
        icon-bg="bg2"
        :number="stats.externalServices"
        number-color="color1"
        title="外部服务"
      >
        <template #icon>
          <i class="fas fa-globe" />
        </template>
      </CardStat>

      <CardStat
        icon-bg="bg3"
        :number="stats.internalServices"
        number-color="color1"
        title="内部服务"
      >
        <template #icon>
          <i class="fas fa-network-wired" />
        </template>
      </CardStat>

      <CardStat
        icon-bg="bg4"
        :number="stats.headlessServices"
        number-color="color1"
        title="Headless 服务"
      >
        <template #icon>
          <i class="fas fa-unlink" />
        </template>
      </CardStat>
    </div>

    <!-- ✅ Service 表格区域 -->
    <ServiceTable :services="serviceList" />
  </div>
</template>

<script>
import CardStat from '@/components/Atlhyper/CardStat.vue'
import ServiceTable from '@/components/Atlhyper/ServiceTable.vue'
import { getAllServices } from '@/api/service' // 引入 API

export default {
  name: 'ServiceView',
  components: {
    CardStat,
    ServiceTable
  },
  data() {
    return {
      serviceList: [],
      stats: {
        totalServices: 0,
        externalServices: 0,
        internalServices: 0,
        headlessServices: 0
      }
    }
  },
  computed: {
    cards() {
      return [
        {
          title: '服务总数',
          value: this.stats.totalServices,
          icon: 'fas fa-cubes',
          class: 'card-primary card-round'
        },
        {
          title: '外部服务',
          value: this.stats.externalServices,
          icon: 'fas fa-globe',
          class: 'card-info card-round'
        },
        {
          title: '内部服务',
          value: this.stats.internalServices,
          icon: 'fas fa-network-wired',
          class: 'card-success card-round'
        },
        {
          title: 'Headless 服务',
          value: this.stats.headlessServices,
          icon: 'fas fa-unlink',
          class: 'card-warning card-round'
        }
      ]
    }
  },
  created() {
    getAllServices()
      .then((res) => {
        const raw = res.data || []
        this.serviceList = raw.map((item) => {
          const ports = (item.spec.ports || [])
            .map((p) => `${p.port}:${p.targetPort}`)
            .join(', ')
          const protocols = (item.spec.ports || [])
            .map((p) => p.protocol)
            .join(', ')
          const selector = item.spec.selector
            ? Object.entries(item.spec.selector)
              .map(([k, v]) => `${k}=${v}`)
              .join(', ')
            : '—'

          return {
            name: item.metadata.name,
            namespace: item.metadata.namespace,
            type: item.spec.type || 'ClusterIP',
            clusterIP: item.spec.clusterIP || 'None',
            ports: ports,
            protocol: protocols,
            selector: selector,
            createTime: new Date(
              item.metadata.creationTimestamp
            ).toLocaleString()
          }
        })

        const total = this.serviceList.length
        const external = this.serviceList.filter((s) =>
          ['LoadBalancer', 'NodePort'].includes(s.type)
        ).length
        const headless = this.serviceList.filter(
          (s) => s.clusterIP === 'None'
        ).length
        const internal = total - external

        this.stats = {
          totalServices: total,
          externalServices: external,
          internalServices: internal,
          headlessServices: headless
        }
      })
      .catch((err) => {
        console.error('获取 Service 列表失败：', err)
        this.$message.error('后端服务异常，无法加载 Service 数据！')
      })
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
