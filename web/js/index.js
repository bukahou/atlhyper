const stats = {
  totalLogs: 0,
  criticalCount: 0,
  warningCount: 0,
  uniqueKinds: 0,
};

const safe = (val) => (val === undefined || val === null ? "—" : val);

function updateCards() {
  document.getElementById("stat-total").textContent = stats.totalLogs;
  document.getElementById("stat-critical").textContent = stats.criticalCount;
  document.getElementById("stat-warning").textContent = stats.warningCount;
  document.getElementById("stat-kinds").textContent = stats.uniqueKinds;
}

function updateLogTitle(days) {
  const title = document.getElementById("log-range-title");
  if (title) {
    title.textContent = `最近 ${days} 天事件数`;
  }
}

function rebuildFootFilters(tableApi) {
  tableApi.columns().every(function () {
    const column = this;
    const select = $(
      '<select class="form-control"><option value=""></option></select>'
    )
      .appendTo($(column.footer()).empty())
      .on("change", function () {
        const val = $.fn.dataTable.util.escapeRegex($(this).val());
        column.search(val ? "^" + val + "$" : "", true, false).draw();
      });

    column
      .data()
      .unique()
      .sort()
      .each(function (d) {
        const clean = $("<div>").html(d).text();
        select.append('<option value="' + clean + '">' + clean + "</option>");
      });
  });
}

document.addEventListener("DOMContentLoaded", () => {
  const defaultDays = 1;
  let currentDays = defaultDays;

  // ✅ 异常事件日志
  function renderLogs(logs) {
    stats.totalLogs = logs.length;
    stats.criticalCount = logs.filter((l) => l.severity === "critical").length;
    stats.warningCount = logs.filter((l) => l.severity === "warning").length;
    stats.uniqueKinds = [...new Set(logs.map((l) => l.kind))].length;
    updateCards();

    const tableId = "#multi-filter-select";
    const tableData = logs.map((log) => [
      log.category,
      log.reason,
      log.message,
      log.kind,
      log.name,
      log.namespace,
      log.node,
      log.timestamp ? new Date(log.timestamp).toLocaleString() : "—",
    ]);

    if ($.fn.dataTable.isDataTable(tableId)) {
      const table = $(tableId).DataTable();
      table.clear();
      table.rows.add(tableData).draw();
      rebuildFootFilters(table);
    } else {
      $(tableId).DataTable({
        pageLength: 10,
        data: tableData,

        columns: [
          { title: "category" },
          { title: "reason" },
          { title: "message" },
          { title: "kind" },
          { title: "name" },
          { title: "namespace" },
          { title: "node" },
          { title: "timestamp" },
        ],

        initComplete: function () {
          rebuildFootFilters(this.api());
          setupDaysSelector(currentDays);
        },
      });
    }
  }

  function setupDaysSelector(days) {
    const filterDiv = $("#multi-filter-select_filter");
    filterDiv.html(`
        <label>最近 
          <select id="log-days-selector" class="form-control input-sm" style="width:auto;display:inline-block;margin:0 5px;">
            <option value="1">1 天</option>
            <option value="2">2 天</option>
            <option value="3">3 天</option>
            <option value="4">4 天</option>
            <option value="5">5 天</option>
            <option value="6">6 天</option>
            <option value="7">7 天</option>
          </select> 内日志</label>
      `);

    document.getElementById("log-days-selector").value = days;

    document
      .getElementById("log-days-selector")
      .addEventListener("change", (e) => {
        const selectedDays = parseInt(e.target.value);
        currentDays = selectedDays;
        updateLogTitle(currentDays);
        fetchOnceAndRender(currentDays);
      });
  }

  function fetchOnceAndRender(days) {
    updateLogTitle(days);
    const apiUrl = API_ENDPOINTS.event.listRecent(days);
    console.log("📡 正在请求异常事件日志接口：", apiUrl);

    axios
      .get(apiUrl)
      .then((res) => {
        const data = res.data;
        const logs = (data.logs || []).map((log) => ({
          category: safe(log.Category),
          reason: safe(log.Reason),
          message: safe(log.Message),
          kind: safe(log.Kind),
          name: safe(log.Name),
          namespace: safe(log.Namespace),
          node: safe(log.Node),
          timestamp: log.EventTime || log.Timestamp || log.time || "",
        }));

        renderLogs(logs);
      })
      .catch((err) => {
        console.error("获取异常告警日志失败:", err);
        alert("后端接口异常，无法加载异常日志数据。");
      });
  }

  fetchOnceAndRender(currentDays);

  const refresher = new RealtimeRefresher({
    fetchFunc: () => {
      return axios
        .get(API_ENDPOINTS.event.listRecent(currentDays))
        .then((res) => {
          const data = res.data;
          return (data.logs || []).map((log) => ({
            category: safe(log.Category),
            reason: safe(log.Reason),
            message: safe(log.Message),
            kind: safe(log.Kind),
            name: safe(log.Name),
            namespace: safe(log.Namespace),
            node: safe(log.Node),
            timestamp: log.EventTime || log.Timestamp || log.time || "",
          }));
        });
    },
    interval: 10000,
    hashFunc: (logs) =>
      JSON.stringify(logs.map((l) => l.timestamp + l.name)).slice(0, 500),
    onChange: renderLogs,
  });

  refresher.start();

  // ✅ 集群总览模块（原 fetch 改 axios）
  axios
    .get(API_ENDPOINTS.cluster.overview)
    .then((res) => {
      const data = res.data;

      Circles.create({
        id: "circles-1",
        radius: 45,
        value: data.ready_nodes,
        maxValue: data.total_nodes,
        width: 7,
        text: `${data.ready_nodes}/${data.total_nodes}`,
        colors: ["#f1f1f1", "#2BB930"],
        duration: 400,
      });

      const availablePods = data.total_pods - data.abnormal_pods;
      Circles.create({
        id: "circles-2",
        radius: 45,
        value: availablePods,
        maxValue: data.total_pods,
        width: 7,
        text: `${availablePods}/${data.total_pods}`,
        colors: ["#f1f1f1", "#F25961"],
        duration: 400,
      });

      const versionDiv = document.getElementById("k8s-version");
      if (versionDiv) {
        versionDiv.innerText = data.k8s_version;
      }

      const metricsText = data.has_metrics_server
        ? "✅ Metrics OK"
        : "⚠️ No Metrics";
      const metricsDiv = document.getElementById("metrics-info");
      if (metricsDiv) {
        metricsDiv.innerText = metricsText;
        metricsDiv.style.fontWeight = "bold";
        metricsDiv.style.marginTop = "10px";
      }
    })
    .catch((error) => {
      console.error("❌ 获取集群概要失败:", error);
      alert("🚫 无法加载集群信息，请确认已登录且权限足够");
    });
});
