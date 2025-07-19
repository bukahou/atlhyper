new Vue({
  el: "#app",
  data: {
    configmap: {},
    loading: true,
    error: null,
  },
  created() {
    const urlParams = new URLSearchParams(window.location.search);
    const ns = urlParams.get("ns");

    if (!ns) {
      this.error = "❌ 缺少命名空间参数，请在 URL 中提供 ?ns=命名空间";
      this.loading = false;
      return;
    }

    console.log("📡 正在请求 ConfigMap，命名空间:", ns);

    axios
      .get(API_ENDPOINTS.configmap.listByNamespace(ns))
      .then((res) => {
        console.log("✅ 获取成功:", res.data);
        if (res.data && res.data.length > 0) {
          this.configmap = res.data[0];
          // alert("✅ 成功加载 ConfigMap 数据！");
        } else {
          alert("⚠️ 未找到该命名空间下的 ConfigMap 数据");
        }
      })
      .catch((err) => {
        console.error("❌ 请求失败:", err);
        alert(
          "❌ 加载 ConfigMap 失败：" +
            (err.response?.data?.error || err.message)
        );
        this.error = "加载失败：" + (err.response?.data?.error || err.message);
      })
      .finally(() => {
        this.loading = false;
      });
  },
});
