<template>
  <div class="page-container">
    <!-- ✅ 顶部卡片区域 -->
    <div class="card-row">
      <CardStat
        v-for="(card, index) in cards"
        :key="index"
        :icon-bg="`bg${index + 1}`"
        :number="card.value"
        :number-color="'color1'"
        :title="card.title"
      >
        <template #icon>
          <i :class="card.icon" />
        </template>
      </CardStat>
    </div>

    <!-- ✅ Ingress 表格区域 -->
    <IngressTable :ingresses="ingressList" />
  </div>
</template>

<script>
import CardStat from "@/components/Atlhyper/CardStat.vue";
import IngressTable from "@/components/Atlhyper/IngressTable.vue";
import { getAllIngresses } from "@/api/ingress";

export default {
  name: "IngressView",
  components: {
    CardStat,
    IngressTable,
  },
  data() {
    return {
      ingressList: [],
      stats: {
        totalIngresses: "--",
        uniqueHosts: "--",
        uniqueTLS: "--",
        totalPaths: "--",
      },
    };
  },
  computed: {
    cards() {
      return [
        {
          title: "Ingress 总数",
          value: this.stats.totalIngresses,
          icon: "fas fa-sign-in-alt",
          class: "card-primary card-round",
        },
        {
          title: "使用域名数",
          value: this.stats.uniqueHosts,
          icon: "fas fa-globe",
          class: "card-info card-round",
        },
        {
          title: "TLS 证书数",
          value: this.stats.uniqueTLS,
          icon: "fas fa-shield-alt",
          class: "card-success card-round",
        },
        {
          title: "路由路径总数",
          value: this.stats.totalPaths,
          icon: "fas fa-route",
          class: "card-warning card-round",
        },
      ];
    },
  },
  created() {
    this.fetchIngresses();
  },
  methods: {
    fetchIngresses() {
      getAllIngresses()
        .then((res) => {
          const rawList = res.data || [];

          const parsed = [];
          const hostSet = new Set();
          const tlsSet = new Set();
          let totalPaths = 0;

          rawList.forEach((item) => {
            const name = item.metadata?.name || "—";
            const namespace = item.metadata?.namespace || "—";
            const creationTime = new Date(
              item.metadata?.creationTimestamp
            ).toLocaleString();

            const tls =
              (item.spec?.tls || [])
                .map((t) => {
                  t.hosts?.forEach((h) => tlsSet.add(h));
                  return t.hosts?.join(", ");
                })
                .join("; ") || "—";

            item.spec?.rules?.forEach((rule) => {
              const host = rule.host || "—";
              hostSet.add(host);

              rule.http?.paths?.forEach((p) => {
                totalPaths++;
                parsed.push({
                  name,
                  namespace,
                  host,
                  path: p.path || "/",
                  serviceName: p.backend?.service?.name || "—",
                  servicePort:
                    p.backend?.service?.port?.number ??
                    p.backend?.service?.port?.name ??
                    "—",
                  tls,
                  creationTime,
                });
              });
            });
          });

          this.ingressList = parsed;
          this.stats = {
            totalIngresses: rawList.length,
            uniqueHosts: hostSet.size,
            uniqueTLS: tlsSet.size,
            totalPaths,
          };
        })
        .catch((err) => {
          console.error("获取 Ingress 数据失败:", err);
          this.$message.error(
            "加载 Ingress 数据失败：" +
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
