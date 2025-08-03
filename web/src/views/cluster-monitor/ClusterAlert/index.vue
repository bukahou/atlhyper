<template>
  <div class="cluster-alert-page">
    <!-- ✅ 顶部告警卡片 -->
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

    <!-- ✅ 告警日志表格 -->
    <AlertLogTable
      :logs="alertLogs"
      @update-date-range="handleDateRangeChange"
    />
  </div>
</template>

<script>
import CardStat from "@/components/Atlhyper/CardStat.vue";
import AlertLogTable from "@/components/Atlhyper/AlertLogTable.vue";
import { getRecentEventLogs } from "@/api/eventlog";
// ✅ 正确导入函数名

export default {
  name: "ClusterAlert",
  components: {
    CardStat,
    AlertLogTable,
  },
  data() {
    return {
      cards: [
        {
          title: "总告警数量",
          number: 0,
          icon: "el-icon-warning-outline",
          iconBg: "bg1",
          numberColor: "color1",
        },
        {
          title: "Critical 数量",
          number: 0,
          icon: "el-icon-close",
          iconBg: "bg4",
          numberColor: "color4",
        },
        {
          title: "Warning 数量",
          number: 0,
          icon: "el-icon-warning",
          iconBg: "bg3",
          numberColor: "color3",
        },
        {
          title: "资源种类数量",
          number: 0,
          icon: "el-icon-menu",
          iconBg: "bg2",
          numberColor: "color2",
        },
      ],
      alertLogs: [],
    };
  },
  created() {
    this.fetchAlertLogs();
  },
  methods: {
    fetchAlertLogs(days = 1) {
      getRecentEventLogs(days)
        .then((res) => {
          const logs = (res.data?.logs || []).map((log) => ({
            category: log.Category || "—",
            reason: log.Reason || "—",
            kind: log.Kind || "—",
            name: log.Name || "—",
            namespace: log.Namespace || "—",
            node: log.Node || "—",
            message: log.Message || "—",
            severity: log.Severity?.toLowerCase() || "",
            timestamp: log.Time ? new Date(log.Time).toLocaleString() : "—",
          }));

          this.alertLogs = logs;

          const total = logs.length;
          const critical = logs.filter((l) => l.severity === "critical").length;
          const warning = logs.filter((l) => l.severity === "warning").length;
          const uniqueKinds = [...new Set(logs.map((l) => l.kind))].length;

          this.cards[0].number = total;
          this.cards[1].number = critical;
          this.cards[2].number = warning;
          this.cards[3].number = uniqueKinds;
        })
        .catch((err) => {
          console.error("获取异常日志失败:", err);
          this.$message.error(
            "加载异常日志数据失败：" +
              (err.response?.data?.message || err.message)
          );
        });
    },

    handleDateRangeChange(days) {
      this.fetchAlertLogs(days); // ✅ 使用选择的天数重新请求
    },
  },
};
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
