new Vue({
  el: "#app",
  data: {
    stats: {
      totalIngresses: "--",
      uniqueHosts: "--",
      uniqueTLS: "--",
      totalPaths: "--",
    },
    ingresses: [],
  },
  computed: {
    cards() {
      return [
        {
          title: "Ingress 总数",
          value: this.stats.totalIngresses,
          icon: "flaticon-network",
          class: "card-primary card-round",
        },
        {
          title: "使用域名数",
          value: this.stats.uniqueHosts,
          icon: "flaticon-globe",
          class: "card-info card-round",
        },
        {
          title: "TLS 证书数",
          value: this.stats.uniqueTLS,
          icon: "flaticon-lock",
          class: "card-success card-round",
        },
        {
          title: "路由路径总数",
          value: this.stats.totalPaths,
          icon: "flaticon-list",
          class: "card-warning card-round",
        },
      ];
    },
  },
  created() {
    axios
      .get(API_ENDPOINTS.ingress.listAll)
      .then((res) => {
        const data = res.data;
        this.ingresses = data.map((item) => {
          const hosts = [];
          const paths = [];
          const services = [];
          const ports = [];

          if (item.spec && item.spec.rules) {
            item.spec.rules.forEach((rule) => {
              const host = rule.host || "-";
              hosts.push(host);
              if (rule.http && rule.http.paths) {
                rule.http.paths.forEach((p) => {
                  paths.push(p.path || "/");
                  services.push(p.backend?.service?.name || "—");
                  ports.push(
                    p.backend?.service?.port?.number ||
                      p.backend?.service?.port?.name ||
                      "—"
                  );
                });
              }
            });
          }

          const tlsUsed =
            item.spec?.tls?.length > 0
              ? item.spec.tls.map((t) => t.hosts.join(",")).join("; ")
              : "—";

          return {
            name: item.metadata.name,
            namespace: item.metadata.namespace,
            hosts: hosts.join(", "),
            paths: paths.join(", "),
            serviceNames: services.join(", "),
            ports: ports.join(", "),
            tls: tlsUsed,
            creationTime: new Date(
              item.metadata.creationTimestamp
            ).toLocaleString(),
          };
        });

        const total = this.ingresses.length;
        const allHosts = data.flatMap(
          (ing) => ing.spec?.rules?.map((r) => r.host) || []
        );
        const allTLS = data.flatMap((ing) =>
          (ing.spec?.tls || []).map((t) => t.secretName)
        );
        const totalPaths = data.reduce((sum, ing) => {
          return (
            sum +
            (ing.spec?.rules?.flatMap((r) => r.http?.paths || []).length || 0)
          );
        }, 0);

        this.stats = {
          totalIngresses: total,
          uniqueHosts: [...new Set(allHosts)].length,
          uniqueTLS: [...new Set(allTLS)].length,
          totalPaths: totalPaths,
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
                const clean = $("<div>").html(d).text();
                select.append(
                  '<option value="' + clean + '">' + clean + "</option>"
                );
              });
          });
        });
      })
      .catch((err) => {
        console.error("获取 Ingress 数据失败:", err);
        alert("后端服务异常，无法加载 Ingress 数据。");
      });
  },
});
