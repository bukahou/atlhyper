// new Vue({
//   el: "#app",
//   data: {
//     nodeInfo: {},
//   },
//   created() {
//     const urlParams = new URLSearchParams(window.location.search);
//     const nodeName = urlParams.get("name");
//     if (!nodeName) {
//       alert("未提供节点名称参数 ?name=xxx");
//       return;
//     }

//     axios
//       .get(API_ENDPOINTS.node.getByName(nodeName)) // ✅ 会自动携带 JWT 拦截器
//       .then((res) => {
//         this.nodeInfo = res.data;
//       })
//       .catch((err) => {
//         console.error("获取节点信息失败：", err);
//         alert(err.response?.data?.error || "无法获取节点详情数据");
//       });
//   },

//   methods: {
//     getInternalIP(addresses) {
//       const addr = addresses?.find((a) => a.type === "InternalIP");
//       return addr ? addr.address : "N/A";
//     },
//     getHostname(addresses) {
//       const addr = addresses?.find((a) => a.type === "Hostname");
//       return addr ? addr.address : "N/A";
//     },
//     getCondition(conditions, type) {
//       const cond = conditions?.find((c) => c.type === type);
//       return cond ? cond.status : "未知";
//     },
//     hasInternalIP(addresses) {
//       return (
//         Array.isArray(addresses) &&
//         addresses.some((a) => a.type === "InternalIP")
//       );
//     },
//     getCNIType(annotations) {
//       if (!annotations) return "未知";
//       if ("flannel.alpha.coreos.com/public-ip" in annotations) return "Flannel";
//       if ("projectcalico.org/IPv4Address" in annotations) return "Calico";
//       return "未知";
//     },
//   },
// });

new Vue({
  el: "#app",
  data: {
    nodeInfo: {},
  },
  created() {
    const urlParams = new URLSearchParams(window.location.search);
    const nodeName = urlParams.get("name");
    if (!nodeName) {
      alert("未提供节点名称参数 ?name=xxx");
      return;
    }

    axios
      .get(API_ENDPOINTS.node.getByName(nodeName)) // ✅ 会自动携带 JWT 拦截器
      .then((res) => {
        this.nodeInfo = res.data;
      })
      .catch((err) => {
        console.error("获取节点信息失败：", err);
        alert(err.response?.data?.error || "无法获取节点详情数据");
      });
  },

  methods: {
    getInternalIP(addresses) {
      const addr = addresses?.find((a) => a.type === "InternalIP");
      return addr ? addr.address : "N/A";
    },
    getHostname(addresses) {
      const addr = addresses?.find((a) => a.type === "Hostname");
      return addr ? addr.address : "N/A";
    },
    getCondition(conditions, type) {
      const cond = conditions?.find((c) => c.type === type);
      return cond ? cond.status : "未知";
    },
    hasInternalIP(addresses) {
      return (
        Array.isArray(addresses) &&
        addresses.some((a) => a.type === "InternalIP")
      );
    },
    getCNIType(annotations) {
      if (!annotations) return "未知";
      if ("flannel.alpha.coreos.com/public-ip" in annotations) return "Flannel";
      if ("projectcalico.org/IPv4Address" in annotations) return "Calico";
      return "未知";
    },

    // 封锁节点
    // 封锁节点
    lockNode() {
      const nodeName = this.nodeInfo.node.metadata.name; // 获取当前节点名称
      const requestData = {
        name: nodeName,
        unschedulable: true, // 设置为 true 表示封锁
      };

      axios
        .post(API_ENDPOINTS.node.schedule, requestData)
        .then((response) => {
          console.log("节点已封锁:", response.data);
          // 更新节点信息
          this.loadNodeInfo(nodeName);
        })
        .catch((error) => {
          console.error("封锁节点失败:", error.response?.data?.error || error);
        });
    },

    // 解封节点
    unlockNode() {
      const nodeName = this.nodeInfo.node.metadata.name; // 获取当前节点名称
      const requestData = {
        name: nodeName,
        unschedulable: false, // 设置为 false 表示解封
      };

      axios
        .post(API_ENDPOINTS.node.schedule, requestData)
        .then((response) => {
          console.log("节点已解封:", response.data);
          // 更新节点信息
          this.loadNodeInfo(nodeName);
        })
        .catch((error) => {
          console.error("解封节点失败:", error.response?.data?.error || error);
        });
    },

    // 用于重新加载节点信息
    loadNodeInfo(nodeName) {
      axios
        .get(API_ENDPOINTS.node.getByName(nodeName))
        .then((response) => {
          this.nodeInfo = response.data;
        })
        .catch((error) => {
          console.error("获取节点信息失败:", error);
        });
    },
  },
});
