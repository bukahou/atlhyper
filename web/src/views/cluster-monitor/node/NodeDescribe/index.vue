<template>
  <div class="node-describe-page">
    <div class="container">
      <!-- ✅ 横排 InfoCard 卡片区域 -->
      <div class="card-flex-container">
        <InfoCard title="基本信息" :items="basicInfoItems" />
        <InfoCard title="系统信息" :items="systemInfoItems" />
        <InfoCard title="网络信息" :items="networkInfoItems" />
        <InfoCard title="节点状态" :items="statusInfoItems" />
      </div>

      <!-- ✅ 下方事件表格与 Pod 表格 -->
      <!-- ✅ 修改后的 下方事件表格与 Pod 表格 区域：上下排列 -->
      <div class="mt-4">
        <EventTable :title="'节点事件'" :events="nodeEvents" />
      </div>
      <div class="mt-4">
        <NodePodTable :pods="runningPods" />
      </div>
    </div>
  </div>
</template>

<script>
import InfoCard from "@/components/Atlhyper/InfoCard.vue";
// import InfoCard from "../../pod/PodDescribe/components/InfoCard.vue";
import EventTable from "@/components/Atlhyper/EventDescribe.vue";
import NodePodTable from "./components/NodePodTable.vue";
import { getNodeDetail } from "@/api/node";

export default {
  name: "NodeDescribe",
  components: {
    InfoCard,
    EventTable,
    NodePodTable,
  },
  data() {
    return {
      node: null,
      basicInfoItems: [],
      systemInfoItems: [],
      networkInfoItems: [],
      statusInfoItems: [],
      nodeEvents: [],
      runningPods: [],
    };
  },
  mounted() {
    const nodeName = this.$route.params.name;
    getNodeDetail(nodeName).then((res) => {
      if (res.code === 20000) {
        this.node = res.data;
        this.prepareBasicInfo(res.data);
        this.prepareSystemInfo(res.data);
        this.prepareNetworkInfo(res.data);
        this.prepareStatusInfo(res.data);
        this.nodeEvents = res.data.events || [];
        this.runningPods = (res.data.runningPods || []).map((pod) => ({
          name: pod.metadata.name,
          namespace: pod.metadata.namespace,
          containerCount: pod.spec.containers?.length || 0,
          status: pod.status.phase || "-",
          restartCount:
            pod.status.containerStatuses?.reduce(
              (sum, c) => sum + (c.restartCount || 0),
              0
            ) || 0,
          startTime: pod.status.startTime,
        }));
      } else {
        this.$message.error(res.message || "获取节点信息失败");
      }
    });
  },
  methods: {
    prepareBasicInfo(data) {
      this.basicInfoItems = [
        { label: "节点名称", value: data.node.metadata.name },
        {
          label: "调度状态",
          value: data.unschedulable ? "不可调度" : "可调度",
        },
        {
          label: "是否为污点",
          value: data.taints && data.taints.length > 0 ? "是" : "否",
        },
        {
          label: "CPU 使用率",
          value:
            data.usage && data.usage.cpuUsagePercent != null
              ? data.usage.cpuUsagePercent.toFixed(2) + "%"
              : "-",
        },
        {
          label: "内存使用率",
          value:
            data.usage && data.usage.memoryUsagePercent != null
              ? data.usage.memoryUsagePercent.toFixed(2) + "%"
              : "-",
        },
      ];
    },
    prepareSystemInfo(data) {
      const si = data.node.status.nodeInfo;
      this.systemInfoItems = [
        { label: "内核版本", value: si.kernelVersion || "-" },
        { label: "OS 镜像", value: si.osImage || "-" },
        { label: "容器运行时", value: si.containerRuntimeVersion || "-" },
        { label: "Kubelet 版本", value: si.kubeletVersion || "-" },
        { label: "架构", value: si.architecture || "-" },
      ];
    },
    prepareNetworkInfo(data) {
      const status = data.node.status;
      const addresses = status.addresses || [];
      const internalIP = addresses.find((a) => a.type === "InternalIP");
      const hostname = addresses.find((a) => a.type === "Hostname");

      const internal = internalIP?.address || "-";

      this.networkInfoItems = [
        { label: "Internal IP", value: internal },
        { label: "Hostname", value: hostname?.address || "-" },
        { label: "Flannel 公网 IP", value: internal }, // ✅ 改为同 internal
        { label: "Pod CIDR", value: data.node.spec.podCIDR || "-" },
        { label: "网络插件", value: data.cniPlugin || "Flannel" },
      ];
    },
    prepareStatusInfo(data) {
      const conditions = data.node.status.conditions || [];

      const getCondStatus = (type) => {
        const cond = conditions.find((c) => c.type === type);
        return cond ? cond.status : "-";
      };

      this.statusInfoItems = [
        { label: "就绪", value: getCondStatus("Ready") },
        { label: "内存压力", value: getCondStatus("MemoryPressure") },
        { label: "磁盘压力", value: getCondStatus("DiskPressure") },
        { label: "PID 压力", value: getCondStatus("PIDPressure") },
        {
          label: "运行 Pod 数",
          value:
            Array.isArray(data.runningPods) && data.runningPods.length >= 0
              ? data.runningPods.length
              : "-",
        },
      ];
    },
  },
};
</script>

<style scoped>
.node-describe-page {
  padding: 20px;
}

/* ✅ 统一卡片区域样式 */
.card-flex-container {
  display: flex;
  flex-wrap: wrap;
  gap: 20px;
  justify-content: flex-start;
}

/* ✅ 两侧平分区域 */
.condition-event-row {
  display: flex;
  flex-wrap: wrap;
  gap: 20px;
  margin-top: 20px;
}

/* ✅ 卡片最小宽度限制，与 Pod 页一致 */
.half-panel {
  flex: 1 1 0;
  min-width: 420px;
}
</style>
