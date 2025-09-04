<template>
  <div class="page-container">
    <!-- 🔁 自动轮询（页面可见；集群切换重建定时器） -->
    <AutoPoll
      v-if="currentId"
      :key="currentId"
      :interval="10000"
      :visible-only="true"
      :immediate="false"
      :task="refresh"
    />

    <!-- 顶部卡片 -->
    <div class="card-row">
      <CardStat
        icon-bg="bg1"
        :number="stats.totalIngresses"
        number-color="color1"
        title="Ingress 总数"
      >
        <template #icon><i class="fas fa-sign-in-alt" /></template>
      </CardStat>
      <CardStat
        icon-bg="bg2"
        :number="stats.usedHosts"
        number-color="color1"
        title="使用域名数"
      >
        <template #icon><i class="fas fa-globe" /></template>
      </CardStat>
      <CardStat
        icon-bg="bg3"
        :number="stats.tlsCerts"
        number-color="color1"
        title="TLS 证书数"
      >
        <template #icon><i class="fas fa-shield-alt" /></template>
      </CardStat>
      <CardStat
        icon-bg="bg4"
        :number="stats.totalPaths"
        number-color="color1"
        title="路由路径总数"
      >
        <template #icon><i class="fas fa-route" /></template>
      </CardStat>
    </div>

    <!-- 表格：请在 IngressTable 的 Actions 里触发 $emit('view', row) -->
    <IngressTable :ingresses="ingressList" @view="handleViewIngress" />

    <!-- ▶️ 右侧抽屉：Ingress 详情 -->
    <IngressDetailDrawer
      v-if="drawerVisible"
      :visible.sync="drawerVisible"
      :ing="ingDetail"
      width="50%"
      v-loading="drawerLoading"
      @close="drawerVisible = false"
    />
  </div>
</template>

<script>
import AutoPoll from "@/components/Atlhyper/AutoPoll.vue";
import CardStat from "@/components/Atlhyper/CardStat.vue";
import IngressTable from "@/components/Atlhyper/IngressTable.vue";
import IngressDetailDrawer from "./ingressDescribe/IngressDetailDrawer.vue";
import { getAllIngresses, getIngressesDetail } from "@/api/ingress";
import { mapState } from "vuex";

export default {
  name: "IngressView",
  components: { AutoPoll, CardStat, IngressTable, IngressDetailDrawer },
  data() {
    return {
      stats: { totalIngresses: 0, usedHosts: 0, tlsCerts: 0, totalPaths: 0 },
      ingressList: [],
      loading: false,

      // 抽屉
      drawerVisible: false,
      drawerLoading: false,
      ingDetail: {},
    };
  },
  computed: {
    ...mapState("cluster", ["currentId"]),
  },
  watch: {
    currentId: {
      immediate: true,
      handler(id) {
        if (id) this.refresh();
      },
    },
  },
  methods: {
    // 🔁 轮询/首帧统一入口
    async refresh() {
      if (!this.currentId || this.loading) return;
      await this.loadIngressData(this.currentId);
    },

    async loadIngressData(clusterId) {
      if (!clusterId || this.loading) return;
      this.loading = true;
      try {
        const res = await getAllIngresses(clusterId);
        if (res.code !== 20000) {
          this.$message.error(res.message || "获取 Ingress 概览失败");
          return;
        }
        const { cards = {}, rows } = res.data || {};

        // 顶部 4 卡
        this.stats = {
          totalIngresses: Number(cards.totalIngresses || 0),
          usedHosts: Number(cards.usedHosts || 0),
          tlsCerts: Number(cards.tlsCerts || 0),
          totalPaths: Number(cards.totalPaths || 0),
        };

        // 表格数据
        const list = Array.isArray(rows) ? rows : [];
        this.ingressList = list.map((r) => ({
          name: r.name || "-",
          namespace: r.namespace || "-",
          host: r.host || "-",
          path: r.path || "/",
          serviceName: r.serviceName || "-",
          servicePort: r.servicePort != null ? r.servicePort : "-",
          tls: Array.isArray(r.tls) ? r.tls.join(", ") : r.tls || "",
          createdAt: r.createdAt || "",
          creationTime: this.formatTime(r.createdAt),
        }));
      } catch (err) {
        this.$message.error("请求失败：" + (err.message || err));
      } finally {
        this.loading = false;
      }
    },

    // ▶️ 查看详情：clusterId + namespace + name
    async handleViewIngress(row) {
      if (!this.currentId) {
        this.$message.error("未选择集群");
        return;
      }
      const ns = row.namespace;
      const name = row.name;
      if (!ns || !name) return;

      this.drawerLoading = true;
      try {
        const res = await getIngressesDetail(this.currentId, ns, name);
        if (res.code !== 20000) {
          this.$message.error(res.message || "获取 Ingress 详情失败");
          return;
        }
        this.ingDetail = res.data || {};
        this.drawerVisible = true;
      } catch (e) {
        this.$message.error("获取 Ingress 详情失败：" + (e?.message || e));
      } finally {
        this.drawerLoading = false;
      }
    },

    formatTime(iso) {
      const t = Date.parse(iso);
      if (!Number.isFinite(t)) return iso || "-";
      const d = new Date(t);
      const pad = (n) => String(n).padStart(2, "0");
      return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(
        d.getDate()
      )} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
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
