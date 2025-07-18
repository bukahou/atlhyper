const stats = {
  totalLogs: 0,
  criticalCount: 0,
  warningCount: 0,
  uniqueKinds: 0,
};

const safe = (val) => (val === undefined || val === null ? "—" : val);

// ✅ 自动附带 JWT Token 的 fetch 封装
function authFetch(url, options = {}) {
  const token = localStorage.getItem("jwt");
  if (!token) {
    alert("❌ 未登录，未找到 Token，请重新登录！");
    throw new Error("Token 不存在");
  }

  options.headers = {
    ...(options.headers || {}),
    Authorization: "Bearer " + token,
  };

  return fetch(url, options);
}

function updateCards() {
  document.getElementById("stat-total").textContent = stats.totalLogs;
  document.getElementById("stat-critical").textContent = stats.criticalCount;
  document.getElementById("stat-warning").textContent = stats.warningCount;
  document.getElementById("stat-kinds").textContent = stats.uniqueKinds;
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
      fetchLogs(parseInt(e.target.value));
    });
}

function fetchLogs(days = 3) {
  authFetch(API_ENDPOINTS.event.listRecent(days))
    .then((res) => {
      if (!res.ok) throw new Error("请求失败：" + res.status);
      return res.json();
    })
    .then((data) => {
      const logs = (data.logs || []).map((log) => ({
        name: safe(log.Name),
        namespace: safe(log.Namespace),
        kind: safe(log.Kind),
        node: safe(log.Node),
        severity: safe((log.Severity || "").toLowerCase()),
        ReasonCode: safe(log.ReasonCode),
        message: safe(log.Message),
        timestamp: log.Timestamp || log.eventTime || log.time || "",
      }));

      stats.totalLogs = logs.length;
      stats.criticalCount = logs.filter(
        (l) => l.severity === "critical"
      ).length;
      stats.warningCount = logs.filter((l) => l.severity === "warning").length;
      stats.uniqueKinds = [...new Set(logs.map((l) => l.kind))].length;
      updateCards();

      const tableId = "#multi-filter-select";
      const tableData = logs.map((log) => [
        log.name,
        log.namespace,
        log.kind,
        log.node,
        log.severity,
        log.ReasonCode,
        log.message,
        log.timestamp ? new Date(log.timestamp).toLocaleString() : "—",
      ]);

      if ($.fn.dataTable.isDataTable(tableId)) {
        const table = $(tableId).DataTable();
        table.clear();
        table.rows.add(tableData).draw();
        rebuildFootFilters(table); // 👈 更新筛选器
      } else {
        $(tableId).DataTable({
          pageLength: 10,
          data: tableData,
          columns: [
            { title: "名称" },
            { title: "命名空间" },
            { title: "资源类型" },
            { title: "节点" },
            { title: "严重等级" },
            { title: "原因" },
            { title: "信息" },
            { title: "事件时间" },
          ],
          initComplete: function () {
            rebuildFootFilters(this.api());
            setupDaysSelector(days);
          },
        });
      }
    })
    .catch((err) => {
      console.error("获取异常告警日志失败:", err);
      alert("❌ 无法加载异常日志数据：" + err.message);
    });
}

document.addEventListener("DOMContentLoaded", () => {
  fetchLogs();
});
