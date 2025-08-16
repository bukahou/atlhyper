<template>
  <div class="page-container">
    <!-- 🔁 5s 自动轮询：仅页面可见 & 命中 keep-alive 时启停 -->
    <AutoPoll
      :interval="5000"
      :visible-only="true"
      :immediate="true"
      :task="refreshAll"
    />

    <!-- ✅ 顶部状态卡片区域 -->
    <div class="card-row">
      <CardStat
        v-for="(item, index) in podStats"
        :key="index"
        :icon-bg="item.iconBg"
        :number="item.count"
        :number-color="item.numberColor"
        :title="item.title"
      >
        <template #icon>
          <i :class="item.iconClass" />
        </template>
      </CardStat>
    </div>

    <!-- ✅ Pod 表格组件 -->
    <PodTable
      :pods="podList"
      @restart="handleRestartPod"
      @view="handleViewPod"
    />
  </div>
</template>

<script>
import AutoPoll from '@/components/Atlhyper/AutoPoll.vue'
import CardStat from '@/components/Atlhyper/CardStat.vue'
import PodTable from '@/components/Atlhyper/PodTable.vue'
import { getPodSummary, getBriefPods, restartPod } from '@/api/pod'

export default {
  name: 'PodPage',
  components: {
    AutoPoll,
    CardStat,
    PodTable
  },
  data() {
    return {
      podStats: [],
      podList: []
    }
  },
  methods: {
    // 一次性刷新两个接口
    async refreshAll() {
      await Promise.all([this.loadPodSummary(), this.loadPodList()])
    },

    async loadPodSummary() {
      try {
        const res = await getPodSummary()
        const data = res.data
        this.podStats = [
          {
            title: 'Running',
            count: data.running,
            iconClass: 'fas fa-play-circle',
            iconBg: 'bg2',
            numberColor: 'color1'
          },
          {
            title: 'Pending',
            count: data.pending,
            iconClass: 'fas fa-hourglass-half',
            iconBg: 'bg3',
            numberColor: 'color1'
          },
          {
            title: 'Failed',
            count: data.failed,
            iconClass: 'fas fa-times-circle',
            iconBg: 'bg4',
            numberColor: 'color1'
          },
          {
            title: 'Unknown',
            count: data.unknown,
            iconClass: 'fas fa-question-circle',
            iconBg: 'bg1',
            numberColor: 'color1'
          }
        ]
      } catch (e) {
        this.$message.error('获取 Pod 状态失败')
      }
    },

    async loadPodList() {
      try {
        const res = await getBriefPods()
        this.podList = res.data
      } catch (e) {
        this.$message.error('获取 Pod 列表失败')
      }
    },

    async handleRestartPod(pod) {
      try {
        await this.$confirm(`确认要重启 Pod「${pod.name}」吗？`, '重启确认', {
          type: 'warning'
        })
        const res = await restartPod(pod.namespace, pod.name)
        this.$message.success(res.message || '重启成功')
        // 重启后立即刷新一次列表
        await this.loadPodList()
      } catch (_) {
        // 用户取消或失败都忽略
      }
    },

    handleViewPod(pod) {
      this.$router.push({
        name: 'PodDescribe',
        query: {
          namespace: pod.namespace,
          name: pod.name
        }
      })
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
