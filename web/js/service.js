new Vue({
  el: "#app",
  data: {
    stats: {
      totalServices: "--",
      externalServices: "--",
      internalServices: "--",
      headlessServices: "--",
    },
    services: [],
  },
  computed: {
    cards() {
      return [
        {
          title: "服务总数",
          value: this.stats.totalServices,
          icon: "flaticon-network",
          class: "card-primary card-round",
        },
        {
          title: "外部服务",
          value: this.stats.externalServices,
          icon: "flaticon-globe",
          class: "card-info card-round",
        },
        {
          title: "内部服务",
          value: this.stats.internalServices,
          icon: "flaticon-interface-6",
          class: "card-success card-round",
        },
        {
          title: "Headless 服务",
          value: this.stats.headlessServices,
          icon: "flaticon-technology",
          class: "card-warning card-round",
        },
      ];
    },
  },
  created() {
    axios
      .get(API_ENDPOINTS.service.listAll)
      .then((res) => {
        this.services = res.data.map((item) => {
          return {
            name: item.metadata.name,
            namespace: item.metadata.namespace,
            type: item.spec.type || "ClusterIP",
            clusterIP: item.spec.clusterIP || "None",
            ports: item.spec.ports
              .map((p) => `${p.port}:${p.targetPort}`)
              .join(", "),
            protocols: item.spec.ports.map((p) => p.protocol).join(", "),
            selectors: item.spec.selector
              ? Object.entries(item.spec.selector)
                  .map(([k, v]) => `${k}=${v}`)
                  .join(", ")
              : "—",
            creationTime: new Date(
              item.metadata.creationTimestamp
            ).toLocaleString(),
          };
        });

        const total = this.services.length;
        const external = this.services.filter((svc) =>
          ["LoadBalancer", "NodePort"].includes(svc.type)
        ).length;
        const headless = this.services.filter(
          (svc) => svc.clusterIP === "None"
        ).length;
        const internal = total - external;

        this.stats = {
          totalServices: total,
          externalServices: external,
          internalServices: internal,
          headlessServices: headless,
        };

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
        console.error("获取 Service 数据失败:", err);
        alert(
          "后端服务异常，无法加载 Service 数据：" +
            (err.response?.data?.message || err.message)
        );
      });
  },
});
