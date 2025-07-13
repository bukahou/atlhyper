document.addEventListener("DOMContentLoaded", function () {
  fetch(API_ENDPOINTS.pod.summary)
    .then(res => res.json())
    .then(data => {
      document.getElementById("pod-running").innerText = data.running;
      document.getElementById("pod-pending").innerText = data.pending;
      document.getElementById("pod-failed").innerText = data.failed;
      document.getElementById("pod-unknown").innerText = data.unknown;
    })
    .catch(err => {
      console.error("❌ 无法获取 Pod 概要数据:", err);
    });
});
