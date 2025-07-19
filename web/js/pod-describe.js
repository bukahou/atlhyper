const app = new Vue({
  el: "#app",
  data: {
    pod: null,
    events: [],
    error: null,
    loading: true,
  },
  created() {
    const urlParams = new URLSearchParams(window.location.search);
    const namespace = urlParams.get("namespace");
    const podName = urlParams.get("name");

    if (!namespace || !podName) {
      this.error = "❌ 缺少必要参数 namespace / name";
      this.loading = false;
      return;
    }

    const describeUrl = API_ENDPOINTS.pod.describe(namespace, podName);

    axios
      .get(describeUrl)
      .then((res) => {
        const data = res.data;
        this.pod = data.pod;
        this.pod.usage = data.usage || {};
        this.pod.service = data.service || null;
        this.pod.logs = data.logs || "（无日志内容）";
        this.events = data.events || [];
      })
      .catch((err) => {
        console.error("❌ 获取 Pod 详情失败", err);
        this.error =
          "❌ 获取数据失败：" + (err.response?.data?.message || err.message);
      })
      .finally(() => {
        this.loading = false;
      });
  },
});

// ✅ 复制按钮逻辑（从 Vue 中提取日志）
function copyLogs() {
  const logs = app?.$data?.pod?.logs;
  if (!logs) {
    alert("❌ 日志为空，无法复制");
    return;
  }

  const textarea = document.createElement("textarea");
  textarea.value = logs;
  document.body.appendChild(textarea);
  textarea.select();

  try {
    document.execCommand("copy");
    alert("📋 日志已复制到剪贴板");
  } catch (err) {
    alert("❌ 复制失败，请手动复制");
  }

  document.body.removeChild(textarea);
}
