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
        :number="stats.totalDeployments"
        number-color="color1"
        title="Deployment 总数"
      >
        <template #icon><i class="fas fa-th-large" /></template>
      </CardStat>
      <CardStat
        icon-bg="bg2"
        :number="stats.namespaces"
        number-color="color1"
        title="命名空间数"
      >
        <template #icon><i class="fas fa-project-diagram" /></template>
      </CardStat>
      <CardStat
        icon-bg="bg3"
        :number="stats.totalReplicas"
        number-color="color1"
        title="总副本数"
      >
        <template #icon><i class="fas fa-clone" /></template>
      </CardStat>
      <CardStat
        icon-bg="bg4"
        :number="stats.readyReplicas"
        number-color="color1"
        title="Ready 副本数"
      >
        <template #icon><i class="fas fa-check-double" /></template>
      </CardStat>
    </div>

    <!-- 表格（要求 DeploymentTable 里通过 $emit('view', row) 触发查看） -->
    <DeploymentTable
      :deployments="deploymentList"
      @view="handleViewDeployment"
      @update="handleUpdateDeployment"
    />

    <!-- ▶️ 右侧抽屉：Deployment 详情 -->
    <DeploymentDetailDrawer
      v-if="drawerVisible"
      :visible.sync="drawerVisible"
      :dep="depDetail"
      width="55%"
      v-loading="drawerLoading"
      @close="drawerVisible = false"
    />
  </div>
</template>

<script>
import AutoPoll from "@/components/Atlhyper/AutoPoll.vue";
import CardStat from "@/components/Atlhyper/CardStat.vue";
import DeploymentTable from "@/components/Atlhyper/DeploymentTable.vue";
import DeploymentDetailDrawer from "./deployDescribe/DeploymentDetailDrawer.vue";
import {
  getDeploymentOverview,
  getDeploymentDetail,
  getDeploymentupdateImage,
  getDeploymentScale,
} from "@/api/deployment";
import { mapState } from "vuex";

export default {
  name: "DeploymentView",
  components: { AutoPoll, CardStat, DeploymentTable, DeploymentDetailDrawer },
  data() {
    return {
      stats: {
        totalDeployments: 0,
        namespaces: 0,
        totalReplicas: 0,
        readyReplicas: 0,
      },
      deploymentList: [],
      loading: false,

      // 抽屉
      drawerVisible: false,
      drawerLoading: false,
      depDetail: {},

      // 提交中状态（防止重复点击）
      updating: false,
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
    async refresh() {
      if (!this.currentId || this.loading) return;
      await this.fetchDeployments(this.currentId);
    },

    async fetchDeployments(clusterId) {
      if (!clusterId || this.loading) return;
      this.loading = true;
      try {
        const res = await getDeploymentOverview(clusterId);
        if (res.code !== 20000) {
          this.$message.error(res.message || "获取 Deployment 概览失败");
          return;
        }
        const { cards = {}, rows } = res.data || {};

        this.stats = {
          totalDeployments: Number(cards.totalDeployments ?? 0),
          namespaces: Number(cards.namespaces ?? 0),
          totalReplicas: Number(cards.totalReplicas ?? 0),
          readyReplicas: Number(cards.readyReplicas ?? 0),
        };

        const list = Array.isArray(rows) ? rows : [];
        this.deploymentList = list.map((r) => ({
          namespace: r.namespace || "-",
          name: r.name || "-",
          image: Array.isArray(r.image) ? r.image.join(", ") : r.image || "-",
          replicas: r.replicas || "0/0", // "ready/total"
          labelCount: Number(r.labelCount ?? 0),
          annotationCount: Number(r.annoCount ?? r.annotationCount ?? 0),
          createdAt: r.createdAt || "",
          creationTime: this.formatTime(r.createdAt),
        }));
      } catch (err) {
        this.$message.error("请求失败：" + (err.message || err));
      } finally {
        this.loading = false;
      }
    },

    formatTime(iso) {
      const t = Date.parse(iso);
      if (!Number.isFinite(t)) return iso || "-";
      const d = new Date(t);
      const p = (n) => String(n).padStart(2, "0");
      return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(
        d.getHours()
      )}:${p(d.getMinutes())}:${p(d.getSeconds())}`;
    },

    async handleViewDeployment(row) {
      if (!this.currentId) return this.$message.error("未选择集群");
      const ns = row.namespace;
      const name = row.name;
      if (!ns || !name) return;
      this.drawerLoading = true;
      try {
        const res = await getDeploymentDetail(this.currentId, ns, name);
        if (res.code !== 20000) {
          this.$message.error(res.message || "获取 Deployment 详情失败");
          return;
        }
        this.depDetail = res.data || {};
        this.drawerVisible = true;
      } catch (e) {
        this.$message.error("获取 Deployment 详情失败：" + (e?.message || e));
      } finally {
        this.drawerLoading = false;
      }
    },

    // ✅ 被 DeploymentTable 的 "Apply" 触发
    // payload: { namespace, name, image, replicas }
    async handleUpdateDeployment(payload) {
      if (!this.currentId) return this.$message.error("未选择集群");
      if (this.updating) return;

      const {
        namespace,
        name,
        image: newImageRaw,
        replicas: newReplicas,
      } = payload;
      const kind = "Deployment";

      // 从当前表格列表拿“旧镜像/旧副本数”
      const row = this.deploymentList.find(
        (d) => d.namespace === namespace && d.name === name
      );
      if (!row) {
        this.$message.error("未找到要更新的 Deployment 行数据");
        return;
      }

      const oldImageStr = (row.image || "").trim();
      const newImage = (newImageRaw || "").trim();

      // 旧副本总数：row.replicas 形如 "ready/total"
      const oldTotalReplicas =
        parseInt(String(row.replicas || "").split("/")[1], 10) || 0;

      // 需要请求哪些动作
      const needUpdateImage = newImage && newImage !== oldImageStr;
      const needScale =
        Number.isFinite(newReplicas) && newReplicas !== oldTotalReplicas;

      // 多容器镜像时的简单保护（行内编辑仅支持单镜像场景）
      if (needUpdateImage && oldImageStr.includes(",")) {
        this.$message.warning(
          "该 Deployment 含多个容器镜像：行内编辑仅支持单容器场景，请到详情页逐个更新镜像。"
        );
      }

      if (!needUpdateImage && !needScale) {
        this.$message.info("本次无修改");
        return;
      }

      this.updating = true;
      try {
        // 1) 先改镜像（如果有）
        if (needUpdateImage && !oldImageStr.includes(",")) {
          const resImg = await getDeploymentupdateImage(
            this.currentId,
            namespace,
            kind,
            name,
            newImage,
            oldImageStr
          );
          if (resImg.code !== 20000) {
            this.$message.error(resImg.message || "更新镜像失败");
            // 镜像失败直接中断；你也可以选择继续缩放
            throw new Error(resImg.message || "update image failed");
          }
          const cid =
            resImg.data && resImg.data.commandID
              ? `（${resImg.data.commandID}）`
              : "";
          this.$message.success("已下发镜像更新命令" + cid);
        }

        // 2) 再扩缩容（如果有）
        if (needScale) {
          const resScale = await getDeploymentScale(
            this.currentId,
            namespace,
            kind,
            name,
            newReplicas
          );
          if (resScale.code !== 20000) {
            this.$message.error(resScale.message || "扩缩容失败");
            throw new Error(resScale.message || "scale failed");
          }
          const cid =
            resScale.data && resScale.data.commandID
              ? `（${resScale.data.commandID}）`
              : "";
          this.$message.success("已下发扩缩容命令" + cid);
        }

        // 刷新数据
        await this.refresh();
      } catch (e) {
        // 已在上面分别弹了错误，这里兜底一下
        // console.error(e)
      } finally {
        this.updating = false;
      }
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
