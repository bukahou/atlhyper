<template>
  <div class="deploy-describe-page">
    <div v-if="loading" class="text-center text-muted mt-4">
      ⏳ 正在加载 Deployment 信息...
    </div>

    <div
      v-else-if="error"
      class="text-center text-danger font-weight-bold mt-4"
    >
      {{ error }}
    </div>

    <div v-else-if="deployment">
      <div class="container">
        <!-- ✅ 单个大卡片展示所有信息 -->
        <div class="row mb-6">
          <div class="card-flex-container">
            <InfoCard title="Deployment 详情信息" :items="allItems" />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import InfoCard from "@/components/Atlhyper/InfoCard.vue";
import { getDeploymentByName } from "@/api/deployment";

export default {
  name: "DeploymentDescribe",
  components: { InfoCard },
  data() {
    return {
      deployment: null,
      loading: true,
      error: null,
    };
  },
  computed: {
    allItems() {
      const d = this.deployment || {};
      const meta = d.metadata || {};
      const spec = d.spec || {};
      const status = d.status || {};
      const tplSpec = spec.template?.spec || {};
      const container = tplSpec.containers?.[0] || {};
      const strategy = spec.strategy?.rollingUpdate || {};

      const format = (v) => v || "—";
      const formatObj = (obj) =>
        obj && Object.keys(obj).length
          ? Object.entries(obj)
              .map(([k, v]) => `${k}=${v}`)
              .join(", ")
          : "—";

      const ports =
        Array.isArray(container.ports) && container.ports.length > 0
          ? container.ports
              .map((p) => `${p.containerPort}/${p.protocol || "TCP"}`)
              .join(", ")
          : "-";

      const volumeMounts =
        Array.isArray(container.volumeMounts) &&
        container.volumeMounts.length > 0
          ? container.volumeMounts.map((v) => v.mountPath).join(", ")
          : "-";

      return [
        { label: "名称", value: format(meta.name) },
        { label: "命名空间", value: format(meta.namespace) },
        {
          label: "副本数",
          value: `${status.readyReplicas ?? 0} / ${spec.replicas ?? 0}`,
        },
        {
          label: "创建时间",
          value: meta.creationTimestamp
            ? new Date(meta.creationTimestamp).toLocaleString()
            : "-",
        },
        { label: "镜像", value: format(container.image) },
        { label: "端口", value: ports },
        {
          label: "请求资源",
          value: `CPU: ${format(
            container.resources?.requests?.cpu
          )}, Mem: ${format(container.resources?.requests?.memory)}`,
        },
        {
          label: "限制资源",
          value: `CPU: ${format(
            container.resources?.limits?.cpu
          )}, Mem: ${format(container.resources?.limits?.memory)}`,
        },
        { label: "Volume Mounts", value: volumeMounts },
        { label: "Node Selector", value: formatObj(tplSpec.nodeSelector) },
        {
          label: "Tolerations",
          value: Array.isArray(tplSpec.tolerations)
            ? `${tplSpec.tolerations.length} 条`
            : "0 条",
        },
        {
          label: "滚动更新策略",
          value: `maxSurge: ${format(
            strategy.maxSurge
          )}, maxUnavailable: ${format(strategy.maxUnavailable)}`,
        },
        {
          label: "Restart Policy",
          value: format(tplSpec.restartPolicy || "Always"),
        },
        { label: "调度器", value: format(tplSpec.schedulerName) },
        { label: "服务名", value: format(meta.labels?.app || "-") },
        { label: "UID", value: format(meta.uid) },
        { label: "容器名称", value: format(container.name) },
        { label: "镜像拉取策略", value: format(container.imagePullPolicy) },
        {
          label: "环境变量",
          value: Array.isArray(container.env)
            ? container.env.map((e) => `${e.name}=${e.value}`).join(", ")
            : "-",
        },
        {
          label: "环境变量引用",
          value: Array.isArray(container.envFrom)
            ? container.envFrom.map((e) => e.configMapRef?.name).join(", ")
            : "-",
        },
        {
          label: "卷定义",
          value: Array.isArray(tplSpec.volumes)
            ? tplSpec.volumes
                .map((v) => `${v.name}(${v.hostPath?.path || "-"})`)
                .join(", ")
            : "-",
        },
        {
          label: "镜像拉取密钥",
          value: Array.isArray(tplSpec.imagePullSecrets)
            ? tplSpec.imagePullSecrets.map((s) => s.name).join(", ")
            : "-",
        },
        {
          label: "亲和性调度",
          value: tplSpec.affinity?.podAntiAffinity
            ?.preferredDuringSchedulingIgnoredDuringExecution
            ? "已设置"
            : "—",
        },
        {
          label: "状态条件",
          value: Array.isArray(status.conditions)
            ? status.conditions.map((c) => `${c.type}: ${c.status}`).join(", ")
            : "-",
        },
        {
          label: "当前版本",
          value: meta.annotations?.["neurocontroller.version.latest"] || "—",
        },
      ];
    },
  },
  created() {
    const ns = this.$route.query.ns;
    const name = this.$route.query.name;

    if (!ns || !name) {
      this.error = "❌ 缺少参数 ns 或 name";
      this.loading = false;
      return;
    }

    getDeploymentByName(ns, name)
      .then((res) => {
        if (res && res.code === 20000 && res.data) {
          this.deployment = res.data;
        } else {
          this.error = "加载 Deployment 失败：响应格式异常";
        }
      })
      .catch((err) => {
        this.error = "加载 Deployment 失败：" + (err.message || "未知错误");
      })
      .finally(() => {
        this.loading = false;
      });
  },
};
</script>

<style scoped>
.deploy-describe-page {
  padding: 20px;
}

.card-flex-container {
  display: block;
}

.card-flex-container > * {
  width: 100%;
  max-width: 1600px;
  margin: 0 auto 24px;
}
</style>
