new Vue({
  el: "#app",
  data: {
    stats: {
      totalNamespaces: "--",
      activeNamespaces: "--",
      terminatingNamespaces: "--",
      totalPods: "--",
    },
    namespaces: [],
  },
  computed: {
    cards() {
      return [
        {
          title: "Namespace 总数",
          value: this.stats.totalNamespaces,
          icon: "flaticon-network",
          class: "card-primary card-round",
        },
        {
          title: "Active 数",
          value: this.stats.activeNamespaces,
          icon: "flaticon-success",
          class: "card-success card-round",
        },
        {
          title: "Terminating 数",
          value: this.stats.terminatingNamespaces,
          icon: "flaticon-error",
          class: "card-danger card-round",
        },
        {
          title: "总 Pod 数",
          value: this.stats.totalPods,
          icon: "flaticon-box-1",
          class: "card-info card-round",
        },
      ];
    },
  },
  created() {
    axios
      .get(API_ENDPOINTS.namespace.list)
      .then((res) => {
        const data = res.data;
        this.namespaces = data;

        let active = 0,
          terminating = 0,
          totalPods = 0;
        data.forEach((item) => {
          const phase = item.Namespace?.status?.phase || "Unknown";
          if (phase === "Active") active++;
          else terminating++;

          totalPods += item.PodCount || 0;
        });

        this.stats = {
          totalNamespaces: data.length,
          activeNamespaces: active,
          terminatingNamespaces: terminating,
          totalPods: totalPods,
        };

        this.$nextTick(() => {
          const table = $("#multi-filter-select").DataTable({ pageLength: 10 });
          table.columns().every(function () {
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
                const clean = $("<div>").html(d).text(); // 防止 HTML 注入
                select.append(
                  '<option value="' + clean + '">' + clean + "</option>"
                );
              });
          });
        });
      })
      .catch((err) => {
        console.error("获取 Namespace 数据失败:", err);
        alert("后端服务异常，无法加载 Namespace 数据。");
      });
  },
});
