<template>
  <div class="page-container">
    <!-- ✅ 顶部卡片区域 -->
    <div class="card-row">
      <CardStat
        v-for="(card, index) in cards"
        :key="index"
        :icon-bg="'bg' + (index + 1)"
        :number="card.value"
        :number-color="'color' + (index + 1)"
        :title="card.title"
      >
        <template #icon>
          <i :class="card.icon" />
        </template>
      </CardStat>
    </div>

    <!-- ✅ 表格区域 -->
    <NamespaceTable
      :namespaces="namespaceList"
      @view-configmap="handleViewConfigMap"
    />

    <!-- ✅ ConfigMap 弹窗 -->
    <el-dialog
      title="ConfigMap 一览"
      :visible.sync="dialogVisible"
      width="600px"
      center
    >
      <p>所属 Namespace：{{ selectedNamespace }}</p>
      <el-table
        :data="fakeConfigMaps"
        border
        size="small"
        style="margin-top: 10px"
      >
        <el-table-column prop="name" label="名称" width="200" />
        <el-table-column prop="dataCount" label="数据项数量" width="140" />
        <el-table-column prop="creationTime" label="创建时间" />
      </el-table>

      <span slot="footer" class="dialog-footer">
        <el-button @click="dialogVisible = false">关闭</el-button>
      </span>
    </el-dialog>
  </div>
</template>

<script>
import CardStat from "@/components/Atlhyper/CardStat.vue";
import NamespaceTable from "@/components/Atlhyper/NamespaceTable.vue";
import { getAllNamespaces } from "@/api/namespace";

export default {
  name: "NamespaceView",
  components: {
    CardStat,
    NamespaceTable,
  },
  data() {
    return {
      dialogVisible: false,
      selectedNamespace: "",
      namespaceList: [],
      stats: {
        totalNamespaces: "--",
        activeNamespaces: "--",
        terminatingNamespaces: "--",
        totalPods: "--",
      },
      fakeConfigMaps: [
        {
          name: "app-config",
          dataCount: 3,
          creationTime: "2024-07-01 11:00:00",
        },
        {
          name: "logging-config",
          dataCount: 2,
          creationTime: "2024-07-02 09:00:00",
        },
      ],
    };
  },
  computed: {
    cards() {
      return [
        {
          title: "Namespace 总数",
          value: this.stats.totalNamespaces,
          icon: "fas fa-layer-group",
          class: "card-primary card-round",
        },
        {
          title: "Active 数",
          value: this.stats.activeNamespaces,
          icon: "fas fa-check",
          class: "card-success card-round",
        },
        {
          title: "Terminating 数",
          value: this.stats.terminatingNamespaces,
          icon: "fas fa-times",
          class: "card-danger card-round",
        },
        {
          title: "总 Pod 数",
          value: this.stats.totalPods,
          icon: "fas fa-cube",
          class: "card-info card-round",
        },
      ];
    },
  },
  created() {
    this.fetchNamespaces();
  },
  methods: {
    fetchNamespaces() {
      getAllNamespaces()
        .then((res) => {
          const rawList = res.data || [];

          this.namespaceList = rawList.map((item) => {
            const nsMeta = item.Namespace.metadata || {};
            const status = item.Namespace.status?.phase || "Unknown";

            return {
              name: nsMeta.name || "—",
              status: status,
              podCount: item.PodCount || 0,
              labelCount: Object.keys(nsMeta.labels || {}).length,
              annotationCount: Object.keys(nsMeta.annotations || {}).length,
              creationTime: new Date(nsMeta.creationTimestamp).toLocaleString(),
            };
          });

          // 渲染统计卡片数据
          const total = rawList.length;
          let active = 0;
          let terminating = 0;
          let totalPods = 0;

          rawList.forEach((item) => {
            const phase = item.Namespace.status?.phase;
            if (phase === "Active") active++;
            else terminating++;
            totalPods += item.PodCount || 0;
          });

          this.stats = {
            totalNamespaces: total,
            activeNamespaces: active,
            terminatingNamespaces: terminating,
            totalPods: totalPods,
          };
        })
        .catch((err) => {
          console.error("获取 Namespace 数据失败:", err);
          this.$message.error(
            "加载命名空间数据失败：" +
              (err.response?.data?.message || err.message)
          );
        });
    },
  },
};
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
