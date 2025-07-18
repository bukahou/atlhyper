// new Vue({
//   el: "#app",
//   data: {
//     stats: {
//       totalDeployments: "--",
//       uniqueNamespaces: "--",
//       totalReplicas: "--",
//       readyReplicas: "--",
//     },
//     deployments: [],
//   },
//   computed: {
//     cards() {
//       return [
//         {
//           title: "Deployment 总数",
//           value: this.stats.totalDeployments,
//           icon: "flaticon-network",
//           class: "card-primary card-round",
//         },
//         {
//           title: "命名空间数",
//           value: this.stats.uniqueNamespaces,
//           icon: "flaticon-map",
//           class: "card-info card-round",
//         },
//         {
//           title: "总副本数",
//           value: this.stats.totalReplicas,
//           icon: "flaticon-layers",
//           class: "card-success card-round",
//         },
//         {
//           title: "Ready 副本数",
//           value: this.stats.readyReplicas,
//           icon: "flaticon-check",
//           class: "card-warning card-round",
//         },
//       ];
//     },
//   },
//   created() {
//     const token = localStorage.getItem("jwt");
//     if (!token) {
//       alert("❌ 未登录，未找到 Token，请重新登录！");
//       console.error("Token 不存在，终止请求。");
//       return;
//     }

//     axios
//       .get(API_ENDPOINTS.deployment.listAll)
//       .then((res) => {
//         const data = res.data;
//         this.deployments = data;

//         const nsSet = new Set();
//         let totalReplicas = 0;
//         let readyReplicas = 0;

//         data.forEach((item) => {
//           nsSet.add(item.metadata.namespace);
//           totalReplicas += item.spec.replicas || 0;
//           readyReplicas += item.status.readyReplicas || 0;
//         });

//         this.stats = {
//           totalDeployments: data.length,
//           uniqueNamespaces: nsSet.size,
//           totalReplicas,
//           readyReplicas,
//         };

//         this.$nextTick(() => {
//           const table = $("#multi-filter-select").DataTable({ pageLength: 10 });
//           table.columns().every(function () {
//             const column = this;
//             const select = $(
//               '<select class="form-control"><option value=""></option></select>'
//             )
//               .appendTo($(column.footer()).empty())
//               .on("change", function () {
//                 const val = $.fn.dataTable.util.escapeRegex($(this).val());
//                 column.search(val ? "^" + val + "$" : "", true, false).draw();
//               });

//             column
//               .data()
//               .unique()
//               .sort()
//               .each(function (d) {
//                 const clean = $("<div>").html(d).text();
//                 select.append(
//                   '<option value="' + clean + '">' + clean + "</option>"
//                 );
//               });
//           });
//         });
//       })
//       .catch((err) => {
//         console.error("❌ 获取 Deployment 数据失败:", err);
//         alert(
//           "❌ 无法加载 Deployment 数据：" +
//             (err.response?.data?.error || err.message)
//         );
//       });
//   },
// });

new Vue({
  el: "#app",
  data: {
    stats: {
      totalDeployments: "--",
      uniqueNamespaces: "--",
      totalReplicas: "--",
      readyReplicas: "--",
    },
    deployments: [],
  },
  computed: {
    cards() {
      return [
        {
          title: "Deployment 总数",
          value: this.stats.totalDeployments,
          icon: "flaticon-network",
          class: "card-primary card-round",
        },
        {
          title: "命名空间数",
          value: this.stats.uniqueNamespaces,
          icon: "flaticon-map",
          class: "card-info card-round",
        },
        {
          title: "总副本数",
          value: this.stats.totalReplicas,
          icon: "flaticon-layers",
          class: "card-success card-round",
        },
        {
          title: "Ready 副本数",
          value: this.stats.readyReplicas,
          icon: "flaticon-check",
          class: "card-warning card-round",
        },
      ];
    },
  },
  created() {
    const token = localStorage.getItem("jwt");
    if (!token) {
      alert("❌ 未登录，未找到 Token，请重新登录！");
      console.error("Token 不存在，终止请求。");
      return;
    }

    console.log("📡 发起请求：获取全部 Deployment");

    axios
      .get(API_ENDPOINTS.deployment.listAll)
      .then((res) => {
        const data = res.data;
        this.deployments = data;

        const nsSet = new Set();
        let totalReplicas = 0;
        let readyReplicas = 0;

        data.forEach((item) => {
          nsSet.add(item.metadata.namespace);
          totalReplicas += item.spec.replicas || 0;
          readyReplicas += item.status.readyReplicas || 0;
        });

        this.stats = {
          totalDeployments: data.length,
          uniqueNamespaces: nsSet.size,
          totalReplicas,
          readyReplicas,
        };

        this.$nextTick(() => {
          const table = $("#multi-filter-select").DataTable({ pageLength: 10 });

          // ✅ 点击跳转到详情页
          $("#multi-filter-select tbody").on("click", "tr", function () {
            const rowData = table.row(this).data();
            const ns = $(this).find("td").eq(0).text().trim();
            const name = $(this).find("td").eq(1).text().trim();
            if (ns && name) {
              const url = `/pages/deployment-describe.html?ns=${encodeURIComponent(
                ns
              )}&name=${encodeURIComponent(name)}`;
              window.location.href = url;
            }
          });

          // ✅ 添加每列下拉筛选
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
        console.error("❌ 获取 Deployment 数据失败:", err);
        alert(
          "❌ 无法加载 Deployment 数据：" +
            (err.response?.data?.error || err.message)
        );
      });
  },
});
