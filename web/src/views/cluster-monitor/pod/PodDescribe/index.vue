<template>
  <div class="pod-describe-page">
    <div v-if="loading" class="text-center text-muted mt-4">
      ⏳ 正在加载 Pod 信息...
    </div>

    <div
      v-else-if="error"
      class="text-center text-danger font-weight-bold mt-4"
    >
      {{ error }}
    </div>

    <div v-else-if="pod">
      <div class="container">
        <!-- ✅ 横排 InfoCard 卡片 -->
        <div class="row mb-6">
          <!-- ✅ 用 flex 包裹所有卡片 -->
          <div class="card-flex-container">
            <InfoCard
              v-for="(card, index) in infoCards"
              :key="index"
              :title="card.title"
              :items="card.items"
            />
          </div>
        </div>

        <!-- ✅ 状态条件 与 相关事件 并排展示 -->
        <div class="row mt-4 condition-event-row">
          <div class="half-panel">
            <PodConditionTable
              :conditions="(pod && pod.status && pod.status.conditions) || []"
            />
          </div>
          <div class="half-panel">
            <PodEventTable :events="eventList" />
          </div>
        </div>
        <PodLogCard :log-text="logs" />
      </div>
    </div>
  </div>
</template>

<script>
// import InfoCard from "./components/InfoCard.vue";
import InfoCard from '@/components/Atlhyper/InfoCard.vue'

import PodConditionTable from './components/PodConditionTable.vue' // ✅ 新增引入
import PodLogCard from './components/PodLogCard.vue'
import PodEventTable from './components/EventCard.vue'

import { getPodDescribe } from '@/api/pod'

export default {
  name: 'PodDescribe',
  components: {
    InfoCard,
    PodConditionTable, // ✅ 注册组件
    PodEventTable,
    PodLogCard
  },
  data() {
    return {
      pod: null,
      events: [],
      logs: '',
      error: null,
      loading: true
    }
  },
  computed: {
    infoCards() {
      return [
        { title: '状态概览', items: this.statusInfoItems },
        { title: '基本信息', items: this.basicInfoItems },
        { title: '容器信息', items: this.containerInfoItems },
        { title: 'Service 基本信息', items: this.serviceInfoItems }
      ]
    },

    basicInfoItems() {
      if (!this.pod) return []
      return [
        { label: '名称', value: this.pod.metadata?.name },
        { label: '命名空间', value: this.pod.metadata?.namespace },
        { label: 'Pod IP', value: this.pod.status?.podIP },
        { label: '所属节点', value: this.pod.spec?.nodeName }
      ]
    },

    statusInfoItems() {
      if (!this.pod) return []
      return [
        { label: '状态', value: this.pod.status?.phase },
        {
          label: '启动时间',
          value: this.pod.status?.startTime
            ? new Date(this.pod.status.startTime).toLocaleString()
            : '-'
        },
        {
          label: '重启次数',
          value:
            this.pod.status?.containerStatuses?.[0]?.restartCount?.toString()
        },
        { label: 'QoS 类别', value: this.pod.status?.qosClass },
        { label: '当前 CPU 使用', value: this.pod.usage?.cpu || 'N/A' },
        { label: '当前内存使用', value: this.pod.usage?.memory || 'N/A' }
      ]
    },

    containerInfoItems() {
      if (!this.pod) return []
      const container = this.pod.spec?.containers?.[0] || {}
      const ports = (container.ports || [])
        .map((p) => `${p.containerPort} / ${p.protocol}`)
        .join(', ')

      return [
        { label: '容器名称', value: container.name },
        { label: '镜像', value: container.image },
        { label: '端口', value: ports || '-' },
        { label: 'CPU 限制', value: container.resources?.limits?.cpu || '-' },
        {
          label: '内存限制',
          value: container.resources?.limits?.memory || '-'
        }
      ]
    },

    serviceInfoItems() {
      const service = {
        spec: {
          type: 'ClusterIP',
          clusterIP: '10.43.0.1',
          ports: [
            {
              port: 443,
              targetPort: 6443,
              name: 'https'
            }
          ]
        }
      }

      const port = service.spec.ports?.[0] || {}

      return [
        { label: '类型', value: service.spec.type },
        { label: 'Cluster IP', value: service.spec.clusterIP },
        { label: '服务端口', value: port.port },
        { label: '容器端口', value: port.targetPort },
        { label: '端口名称', value: port.name }
      ]
    },
    eventList() {
      return this.events || []
    }
  },
  created() {
    const { namespace, name } = this.$route.query
    if (!namespace || !name) {
      this.error = '❌ 缺少必要参数 namespace 或 name'
      this.loading = false
      return
    }

    getPodDescribe(namespace, name)
      .then((res) => {
        const data = res.data
        this.pod = data.pod
        this.pod.usage = data.usage || {}
        this.events = data.events || []
        this.logs = data.logs || '（无日志内容）'
      })
      .catch((err) => {
        this.error =
          err.response?.data?.message || '获取 Pod 信息失败，请稍后重试'
      })
      .finally(() => {
        this.loading = false
      })
  }
}
</script>

<style scoped>
.pod-describe-page {
  padding: 20px;
}

.card-flex-container {
  display: flex;
  flex-wrap: wrap;
  gap: 20px;
  justify-content: flex-start;
}

.condition-event-row {
  display: flex;
  flex-wrap: wrap;
  gap: 20px;
  margin-top: 20px;
}

.half-panel {
  flex: 1 1 0;
  min-width: 420px; /* 你 InfoCard 最大宽度是 420，这样风格一致 */
}
</style>
