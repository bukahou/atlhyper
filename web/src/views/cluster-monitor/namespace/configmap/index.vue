<template>
  <div class="configmap-detail-container">
    <div class="card-column">
      <InfoCard :title="firstCard.title" :items="firstCard.items" />
      <InfoCard :title="secondCard.title" :items="secondCard.items" />
    </div>
  </div>
</template>

<script>
import InfoCard from "@/components/Atlhyper/InfoCard.vue";
import { getConfigMapsByNamespace } from "@/api/namespace";

export default {
  name: "ConfigMapDetail",
  components: {
    InfoCard,
  },
  data() {
    return {
      firstCard: {
        title: "基本信息",
        items: [],
      },
      secondCard: {
        title: "配置项内容",
        items: [],
      },
    };
  },
  created() {
    const namespace = this.$route.query.ns;
    if (!namespace) {
      this.$message.error("未提供命名空间参数");
      return;
    }
    this.loadConfigMap(namespace);
  },
  methods: {
    async loadConfigMap(ns) {
      try {
        const res = await getConfigMapsByNamespace(ns);
        const cm = res.data?.[0];
        if (!cm) {
          this.$message.warning("该命名空间下无 ConfigMap");
          return;
        }

        const metadata = cm.metadata || {};
        const data = cm.data || {};

        this.firstCard.items = [
          { label: "名称", value: metadata.name || "-" },
          { label: "命名空间", value: metadata.namespace || "-" },
          {
            label: "注解条数",
            value: Object.keys(metadata.annotations || {}).length,
          },
          {
            label: "创建时间",
            value: new Date(metadata.creationTimestamp).toLocaleString(),
          },
          {
            label: "数据条数",
            value: Object.keys(data).length,
          },
        ];

        this.secondCard.items = Object.entries(data).map(([key, value]) => ({
          label: key,
          value,
        }));
      } catch (err) {
        console.error("加载 ConfigMap 失败:", err);
        this.$message.error("加载 ConfigMap 失败");
      }
    },
  },
};
</script>

<style scoped>
.configmap-detail-container {
  padding: 32px;
}

/* ✅ 替代 :deep(.full-card)，使用 >>> 选择器兼容 Vue2 */
.card-column >>> .full-card {
  max-width: 1600px !important;
  width: 100%;
}

/* ✅ 上下排列卡片 */
.card-column {
  display: flex;
  flex-direction: column;
  gap: 32px;
  align-items: center;
}
</style>
