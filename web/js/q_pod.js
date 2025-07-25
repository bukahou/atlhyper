// document.addEventListener("DOMContentLoaded", function () {
//   fetch(API_ENDPOINTS.pod.summary)
//     .then((res) => res.json())
//     .then((data) => {
//       document.getElementById("pod-running").innerText = data.running;
//       document.getElementById("pod-pending").innerText = data.pending;
//       document.getElementById("pod-failed").innerText = data.failed;
//       document.getElementById("pod-unknown").innerText = data.unknown;
//     })
//     .catch((err) => {
//       console.error("❌ 无法获取 Pod 概要数据:", err);
//     });
// });

document.addEventListener("DOMContentLoaded", function () {
  const token = localStorage.getItem("jwt");

  fetch(API_ENDPOINTS.pod.summary, {
    headers: {
      Authorization: "Bearer " + token,
    },
  })
    .then((res) => {
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      return res.json();
    })
    .then((data) => {
      document.getElementById("pod-running").innerText = data.running;
      document.getElementById("pod-pending").innerText = data.pending;
      document.getElementById("pod-failed").innerText = data.failed;
      document.getElementById("pod-unknown").innerText = data.unknown;
    })
    .catch((err) => {
      console.error("❌ 无法获取 Pod 概要数据:", err);
    });
});

// ✅ 重启 Pod 请求逻辑
function restartPod(namespace, name) {
  if (!confirm(`🔁 确认要重启 Pod「${name}」吗？`)) return;

  axios
    .post(API_ENDPOINTS.pod.restart(namespace, name))
    .then((res) => {
      alert(res.data.message || `✅ Pod ${name} 重启成功`);
      location.reload();
    })
    .catch((err) => {
      console.error("❌ 重启失败:", err);
      alert(err.response?.data?.message || "重启失败，权限不足");
    });
}

// ✅ 初始化筛选器
function initColumnFilters(tableInstance) {
  tableInstance.columns().every(function () {
    const column = this;
    const footer = $(column.footer());
    const currentVal = footer.find("select").val();

    if (column.index() === tableInstance.columns().nodes().length - 1) {
      footer.empty();
      return;
    }

    const select = $(
      '<select class="form-control"><option value=""></option></select>'
    ).on("change", function () {
      const val = $.fn.dataTable.util.escapeRegex($(this).val());
      column.search(val ? "^" + val + "$" : "", true, false).draw();
    });

    footer.empty().append(select);

    column
      .data()
      .unique()
      .sort()
      .each(function (d) {
        if (d) select.append('<option value="' + d + '">' + d + "</option>");
      });

    if (currentVal) select.val(currentVal);
  });
}

// ✅ 页面初始化
$(document).ready(function () {
  const table = $("#pod-table").DataTable({
    pageLength: 5,
    columns: [
      { title: "Namespace" },
      { title: "Deployment" },
      { title: "Pod Name" },
      { title: "Ready" },
      { title: "Phase" },
      { title: "Restart Count" },
      { title: "Start Time" },
      { title: "Pod IP" },
      { title: "Node" },
      { title: "操作" },
    ],
    initComplete: function () {
      initColumnFilters(this.api());
    },
  });

  // ✅ 加载 Pod 简略数据
  axios
    .get(API_ENDPOINTS.pod.listBrief)
    .then((res) => {
      res.data.forEach((pod) => {
        const actionBtns = `
          <button class="btn btn-sm btn-primary mr-1" onclick="window.location.href='pod-describe.html?namespace=${encodeURIComponent(
            pod.namespace
          )}&name=${encodeURIComponent(pod.name)}'">查看</button>
          <button class="btn btn-sm btn-warning" onclick="restartPod('${
            pod.namespace
          }', '${pod.name}')">重启</button>
        `;

        table.row.add([
          pod.namespace,
          pod.deployment,
          pod.name,
          pod.ready
            ? '<span class="badge badge-success">Yes</span>'
            : '<span class="badge badge-danger">No</span>',
          pod.phase,
          pod.restartCount,
          new Date(pod.startTime).toLocaleString(),
          pod.podIP,
          pod.nodeName,
          actionBtns,
        ]);
      });
      table.draw();
    })
    .catch((err) => {
      console.error("❌ 获取 Pod 简略数据失败:", err);
      alert("获取 Pod 数据失败，请稍后再试。");
    });

  // ✅ 表格刷新后同步更新筛选器
  table.on("draw", function () {
    initColumnFilters(table);
  });
});
