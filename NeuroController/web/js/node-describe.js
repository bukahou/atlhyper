new Vue({
  el: "#app",
  data: {
    node: {},
  },
  created() {
    const urlParams = new URLSearchParams(window.location.search);
    const nodeName = urlParams.get("name");
    if (!nodeName) {
      alert("未提供节点名称参数 ?name=xxx");
      return;
    }

    axios
      .get(API_ENDPOINTS.node.getByName(nodeName))
      .then((res) => {
        this.node = res.data;
      })
      .catch((err) => {
        console.error("获取 Node 详情失败:", err);
        alert("无法加载节点信息，请稍后再试。");
      });
  },
  methods: {
    getInternalIP(addresses) {
      const addr = addresses.find((a) => a.type === "InternalIP");
      return addr ? addr.address : "N/A";
    },
    getHostname(addresses) {
      const addr = addresses.find((a) => a.type === "Hostname");
      return addr ? addr.address : "N/A";
    },
    getCondition(type) {
      const cond = this.node.status?.conditions?.find((c) => c.type === type);
      return cond ? cond.status : "N/A";
    },
    formatMemory(k8sMemoryStr) {
      if (!k8sMemoryStr) return "N/A";
      const match = k8sMemoryStr.match(/(\d+)([A-Za-z]+)/);
      if (!match) return k8sMemoryStr;
      const value = parseInt(match[1], 10);
      const unit = match[2];
      const unitMap = {
        Ki: 1024,
        Mi: 1024 * 1024,
        Gi: 1024 * 1024 * 1024,
      };
      const bytes = value * (unitMap[unit] || 1);
      return (bytes / 1024 / 1024 / 1024).toFixed(1);
    },
    formatImageSize(sizeBytes) {
      if (!sizeBytes) return "N/A";
      return (sizeBytes / 1024 / 1024).toFixed(1) + " MB";
    },
  },
});
