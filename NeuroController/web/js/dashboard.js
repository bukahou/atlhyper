document.addEventListener("DOMContentLoaded", () => {
  fetch(API_ENDPOINTS.cluster.overview)
    .then(response => response.json())
    .then(data => {
      // ✅ 圆环 1：Ready Nodes
      Circles.create({
        id: 'circles-1',
        radius: 45,
        value: data.ready_nodes,
        maxValue: data.total_nodes,
        width: 7,
        text: `${data.ready_nodes}/${data.total_nodes}`,
        colors: ['#f1f1f1', '#2BB930'],
        duration: 400
      });

      // ✅ 圆环 2：Available Pods / Total Pods
      const availablePods = data.total_pods - data.abnormal_pods;
      Circles.create({
        id: 'circles-2',
        radius: 45,
        value: availablePods,
        maxValue: data.total_pods,
        width: 7,
        text: `${availablePods}/${data.total_pods}`,
        colors: ['#f1f1f1', '#F25961'],
        duration: 400
      });

      // ✅ 文本：Kubernetes Version
      const versionDiv = document.getElementById("k8s-version");
      if (versionDiv) {
        versionDiv.innerText = data.k8s_version;
      }

      // ✅ metrics-server 状态
      const metricsText = data.has_metrics_server ? "✅ Metrics OK" : "⚠️ No Metrics";
      const metricsDiv = document.getElementById("metrics-info");
      if (metricsDiv) {
        metricsDiv.innerText = metricsText;
        metricsDiv.style.fontWeight = "bold";
        metricsDiv.style.marginTop = "10px";
      }
    })
    .catch(error => {
      console.error("❌ 获取集群概要失败:", error);
    });
});
