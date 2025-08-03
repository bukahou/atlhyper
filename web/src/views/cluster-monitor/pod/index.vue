<template>
  <div class="page-container">
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
import CardStat from "@/components/Atlhyper/CardStat.vue";
import PodTable from "@/components/Atlhyper/PodTable.vue";
import { getPodSummary, getBriefPods, restartPod } from "@/api/pod";

export default {
  name: "PodPage",
  components: {
    CardStat,
    PodTable,
  },
  data() {
    return {
      podStats: [],
      podList: [],
    };
  },
  mounted() {
    this.loadPodSummary();
    this.loadPodList();
  },
  methods: {
    loadPodSummary() {
      getPodSummary()
        .then((res) => {
          const data = res.data;
          this.podStats = [
            {
              title: "Running",
              count: data.running,
              iconClass: "fas fa-play-circle",
              iconBg: "bg2",
              numberColor: "color1",
            },
            {
              title: "Pending",
              count: data.pending,
              iconClass: "fas fa-hourglass-half",
              iconBg: "bg3",
              numberColor: "color1",
            },
            {
              title: "Failed",
              count: data.failed,
              iconClass: "fas fa-times-circle",
              iconBg: "bg4",
              numberColor: "color1",
            },
            {
              title: "Unknown",
              count: data.unknown,
              iconClass: "fas fa-question-circle",
              iconBg: "bg1",
              numberColor: "color1",
            },
          ];
        })
        .catch(() => {
          this.$message.error("获取 Pod 状态失败");
        });
    },
    loadPodList() {
      getBriefPods()
        .then((res) => {
          this.podList = res.data;
        })
        .catch(() => {
          this.$message.error("获取 Pod 列表失败");
        });
    },
    handleRestartPod(pod) {
      this.$confirm(`确认要重启 Pod「${pod.name}」吗？`, "重启确认", {
        type: "warning",
      })
        .then(() => {
          return restartPod(pod.namespace, pod.name);
        })
        .then((res) => {
          this.$message.success(res.message || "重启成功");
          this.loadPodList(); // 重载列表
        })
        .catch(() => {});
    },
    handleViewPod(pod) {
      this.$router.push({
        name: "PodDescribe",
        query: {
          namespace: pod.namespace,
          name: pod.name,
        },
      });
    },
  },
};
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
