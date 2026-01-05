<template>
  <div class="cluster-alert-page">
    <!-- 顶部告警卡片 -->
    <div class="card-row">
      <CardStat
        v-for="card in cards"
        :key="card.title"
        :icon-bg="card.iconBg"
        :number="card.number"
        :number-color="card.numberColor"
        :title="card.title"
      >
        <template #icon>
          <i :class="card.icon" />
        </template>
      </CardStat>
    </div>

    <!-- 告警日志表格 -->
    <AlertLogTable
      ref="alertTable"
      :logs="alertLogs"
      @update-date-range="handleDateRangeChange"
    />
  </div>
</template>

<script>
import CardStat from '@/components/Atlhyper/CardStat.vue'
import AlertLogTable from '@/components/Atlhyper/AlertLogTable.vue'
import { getRecentEventLogs } from '@/api/eventlog'
import { mapState } from 'vuex'

export default {
  name: 'ClusterAlert',
  components: { CardStat, AlertLogTable },
  data() {
    return {
      // 按你要求的展示顺序：totalAlerts / totalEvents / warning / kindsCount
      cards: [
        {
          title: '总告警数量',
          number: 0,
          icon: 'el-icon-warning-outline',
          iconBg: 'bg1',
          numberColor: 'color1'
        },
        {
          title: 'Event 类数量',
          number: 0,
          icon: 'el-icon-bell',
          iconBg: 'bg2',
          numberColor: 'color2'
        },
        {
          title: 'Warning 数量',
          number: 0,
          icon: 'el-icon-warning',
          iconBg: 'bg3',
          numberColor: 'color3'
        },
        {
          title: '资源种类数量',
          number: 0,
          icon: 'el-icon-menu',
          iconBg: 'bg2',
          numberColor: 'color2'
        }
      ],
      alertLogs: [],
      currentDays: 1, // ✅ 父组件记录“最近 N 天”，默认 1 天
      loading: false
    }
  },
  computed: {
    ...mapState('cluster', ['currentId'])
  },
  watch: {
    // 集群切换时，按当前选择的天数刷新
    currentId: {
      immediate: true,
      async handler(id) {
        if (id) {
          await this.fetchAlertLogs(this.currentDays)
          // 同步子组件的下拉显示值（不改子组件源码的前提下）
          this.$nextTick(() => {
            if (this.$refs.alertTable) {
              this.$refs.alertTable.dateRange = String(this.currentDays)
            }
          })
        }
      }
    }
  },
  methods: {
    async fetchAlertLogs(days = 1) {
      if (!this.currentId) return
      this.loading = true
      try {
        const res = await getRecentEventLogs(this.currentId, days) // ✅ 传 clusterId + withinDays(int)
        if (res.code !== 20000) {
          this.$message.error('获取异常日志失败：' + (res.message || ''))
          return
        }

        // cards
        const cards = res.data?.cards || {}
        this.cards[0].number = Number(cards.totalAlerts ?? 0)
        this.cards[1].number = Number(cards.totalEvents ?? 0)
        this.cards[2].number = Number(cards.warning ?? 0)
        this.cards[3].number = Number(cards.kindsCount ?? 0)

        // rows：映射字段名
        const rows = res.data?.rows || []
        this.alertLogs = rows.map((log) => ({
          category: log.Category || '—',
          reason: log.Reason || '—',
          kind: log.Kind || '—',
          name: log.Name || '—',
          namespace: log.Namespace || '—',
          node: log.Node || '—',
          message: log.Message || '—',
          severity: (log.Severity || '').toLowerCase(),
          timestamp: log.Time ? new Date(log.Time).toLocaleString() : '—',
          // 保留原字段（可选）
          clusterID: log.ClusterID || '',
          eventTime: log.EventTime || ''
        }))
      } catch (err) {
        this.$message.error(
          '加载异常日志数据失败：' +
            (err.response?.data?.message || err.message)
        )
      } finally {
        this.loading = false
      }
    },

    // 子组件“最近 N 天”变化 → 更新父组件的 currentDays，并重新拉取
    async handleDateRangeChange(days) {
      this.currentDays = Number(days) || 1 // ✅ 统一为数字
      await this.fetchAlertLogs(this.currentDays)
      // 同步回子组件下拉（防止外部修改造成不一致）
      this.$nextTick(() => {
        if (this.$refs.alertTable) {
          this.$refs.alertTable.dateRange = String(this.currentDays)
        }
      })
    }
  }
}
</script>

<style scoped>
.cluster-alert-page {
  padding: 35px 32px;
}
.card-row {
  display: flex;
  flex-wrap: wrap;
  gap: 80px;
  margin-bottom: 24px;
}
</style>
