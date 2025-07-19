new Vue({
  el: "#app",
  data: {
    stats: {
      totalNodes: "--",
      readyNodes: "--",
      totalCPU: "--",
      totalMemoryGB: "--",
    },
    nodes: [],
  },
  computed: {
    cards() {
      return [
        {
          title: "节点总数",
          value: this.stats.totalNodes,
          icon: "flaticon-users",
          class: "card-primary card-round",
        },
        {
          title: "就绪节点",
          value: this.stats.readyNodes,
          icon: "flaticon-success",
          class: "card-success card-round",
        },
        {
          title: "总 CPU",
          value: this.stats.totalCPU + " 核",
          icon: "flaticon-analytics",
          class: "card-info card-round",
        },
        {
          title: "总内存 (GiB)",
          value: this.stats.totalMemoryGB + " GiB",
          icon: "flaticon-network",
          class: "card-secondary card-round",
        },
      ];
    },
  },
  methods: {
    goToDetail(name) {
      window.location.href = `node-describe.html?name=${encodeURIComponent(
        name
      )}`;
    },
  },
  created() {
    axios
      .get(API_ENDPOINTS.node.overview)
      .then((res) => {
        this.stats = res.data.stats;
        this.nodes = res.data.nodes;

        this.$nextTick(() => {
          const table = $("#multi-filter-select").DataTable({
            pageLength: 10,
          });

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
                select.append('<option value="' + d + '">' + d + "</option>");
              });
          });
        });
      })
      .catch((err) => {
        console.error("获取 Node 总览失败:", err);
        alert("后端服务异常，无法加载节点总览数据。");
      });
  },
});
