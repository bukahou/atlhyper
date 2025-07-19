// const app = new Vue({
//   el: "#app",
//   data: {
//     deployment: {},
//     container: {},
//     editing: false,
//     editingImage: false,
//     editableReplicas: 0,
//     editableImage: "",
//   },
//   created() {
//     this.fetchDeployment();
//   },
//   methods: {
//     fetchDeployment() {
//       const token = localStorage.getItem("jwt");
//       if (!token) {
//         alert("❌ 未登录，未找到 Token，请重新登录！");
//         console.error("Token 不存在，终止请求。");
//         return;
//       }

//       const urlParams = new URLSearchParams(window.location.search);
//       const ns = urlParams.get("ns");
//       const name = urlParams.get("name");

//       if (!ns || !name) {
//         alert("❌ 缺少参数，请在 URL 中提供 ?ns=命名空间&name=名称");
//         return;
//       }

//       $.ajax({
//         url: API_ENDPOINTS.deployment.get(ns, name),
//         method: "GET",
//         headers: {
//           Authorization: "Bearer " + token,
//         },
//         success: (data) => {
//           this.deployment = data;
//           const containers = data.spec?.template?.spec?.containers || [];
//           this.container = containers.length > 0 ? containers[0] : {};
//           this.editableReplicas = data.spec?.replicas || 1;
//           this.editableImage = this.container.image || "";
//         },
//         error: (xhr) => {
//           console.error("获取 Deployment 详情失败:", xhr);
//           alert(
//             "❌ 无法加载 Deployment 信息：" +
//               (xhr.responseText || xhr.statusText)
//           );
//         },
//       });
//     },

//     enterEdit() {
//       this.editing = true;
//       this.editableReplicas = this.deployment.spec?.replicas || 1;
//     },

//     updateReplicas() {
//       const token = localStorage.getItem("jwt");
//       if (!token) {
//         alert("❌ 未登录，未找到 Token，请重新登录！");
//         return;
//       }

//       const current = this.deployment.spec?.replicas || 1;
//       const updated = this.editableReplicas;

//       if (current === updated) {
//         alert("ℹ️ 副本数未变更，无需更新。");
//         this.editing = false;
//         return;
//       }

//       if (!confirm(`请确认是否将副本数从 ${current} 更新为 ${updated}？`))
//         return;

//       $.ajax({
//         url: API_ENDPOINTS.deployment.scale,
//         method: "POST",
//         contentType: "application/json",
//         headers: {
//           Authorization: "Bearer " + token,
//         },
//         data: JSON.stringify({
//           namespace: this.deployment.metadata.namespace,
//           name: this.deployment.metadata.name,
//           replicas: updated,
//         }),
//         success: (resp) => {
//           if (resp.error) {
//             alert("❌ 更新副本数失败: " + resp.error);
//             return;
//           }
//           alert("✅ 副本数更新成功！");
//           this.deployment.spec.replicas = updated;
//           this.editing = false;
//         },
//         error: (xhr) => {
//           alert("❌ 请求失败: " + (xhr.responseText || xhr.statusText));
//         },
//       });
//     },

//     enterEditImage() {
//       this.editingImage = true;
//       this.editableImage = this.container.image || "";
//     },

//     updateImage() {
//       const token = localStorage.getItem("jwt");
//       if (!token) {
//         alert("❌ 未登录，未找到 Token，请重新登录！");
//         return;
//       }

//       const current = this.container.image || "";
//       const updated = this.editableImage || "";

//       if (current === updated) {
//         alert("ℹ️ 镜像未变更，无需更新。");
//         this.editingImage = false;
//         return;
//       }

//       if (!confirm(`请确认是否将镜像从\n${current}\n更新为\n${updated}？`))
//         return;

//       $.ajax({
//         url: API_ENDPOINTS.deployment.scale,
//         method: "POST",
//         contentType: "application/json",
//         headers: {
//           Authorization: "Bearer " + token,
//         },
//         data: JSON.stringify({
//           namespace: this.deployment.metadata.namespace,
//           name: this.deployment.metadata.name,
//           image: updated,
//         }),
//         success: (resp) => {
//           if (resp.error) {
//             alert("❌ 更新镜像失败: " + resp.error);
//             return;
//           }
//           alert("✅ 镜像更新成功！");
//           this.container.image = updated;
//           this.editingImage = false;
//         },
//         error: (xhr) => {
//           alert("❌ 请求失败: " + (xhr.responseText || xhr.statusText));
//         },
//       });
//     },
//   },
// });

const app = new Vue({
  el: "#app",
  data: {
    deployment: {},
    container: {},
    editing: false,
    editingImage: false,
    editableReplicas: 0,
    editableImage: "",
  },
  created() {
    this.fetchDeployment();
  },
  methods: {
    // ✅ 加载 Deployment 详情
    fetchDeployment() {
      const token = localStorage.getItem("jwt");
      if (!token) {
        alert("❌ 未登录，未找到 Token，请重新登录！");
        return;
      }

      const urlParams = new URLSearchParams(window.location.search);
      const ns = urlParams.get("ns");
      const name = urlParams.get("name");

      if (!ns || !name) {
        alert("❌ 缺少参数，请在 URL 中提供 ?ns=命名空间&name=名称");
        return;
      }

      $.ajax({
        url: API_ENDPOINTS.deployment.get(ns, name),
        method: "GET",
        headers: {
          Authorization: "Bearer " + token,
        },
        success: (data) => {
          this.deployment = data;
          const containers = data.spec?.template?.spec?.containers || [];
          this.container = containers.length > 0 ? containers[0] : {};
          this.editableReplicas = data.spec?.replicas || 1;
          this.editableImage = this.container.image || "";
        },
        error: (xhr) => {
          console.error("获取 Deployment 详情失败:", xhr);
          alert(
            "❌ 无法加载 Deployment 信息：" +
              (xhr.responseText || xhr.statusText)
          );
        },
      });
    },

    // ✏️ 进入副本数编辑模式
    enterEdit() {
      this.editing = true;
      this.editableReplicas = this.deployment.spec?.replicas || 1;
    },

    // ✅ 提交副本数更新
    updateReplicas() {
      const token = localStorage.getItem("jwt");
      if (!token) {
        alert("❌ 未登录，未找到 Token，请重新登录！");
        return;
      }

      const current = this.deployment.spec?.replicas || 1;
      const updated = this.editableReplicas;

      if (current === updated) {
        alert("ℹ️ 副本数未变更，无需更新。");
        this.editing = false;
        return;
      }

      if (!confirm(`请确认是否将副本数从 ${current} 更新为 ${updated}？`))
        return;

      $.ajax({
        url: API_ENDPOINTS.deployment.scale,
        method: "POST",
        contentType: "application/json",
        headers: {
          Authorization: "Bearer " + token,
        },
        data: JSON.stringify({
          namespace: this.deployment.metadata.namespace,
          name: this.deployment.metadata.name,
          replicas: updated,
        }),
        success: (resp) => {
          if (resp.error) {
            alert("❌ 更新副本数失败: " + resp.error);
            return;
          }
          alert("✅ 副本数更新成功！");
          this.deployment.spec.replicas = updated;
          this.editing = false;
        },
        error: (xhr) => {
          alert("❌ 请求失败: " + (xhr.responseText || xhr.statusText));
        },
      });
    },

    // ✏️ 镜像编辑模式
    enterEditImage() {
      this.editingImage = true;
      this.editableImage = this.container.image || "";
    },

    // ✅ 提交镜像更新
    updateImage() {
      const token = localStorage.getItem("jwt");
      if (!token) {
        alert("❌ 未登录，未找到 Token，请重新登录！");
        return;
      }

      const current = this.container.image || "";
      const updated = this.editableImage || "";

      if (current === updated) {
        alert("ℹ️ 镜像未变更，无需更新。");
        this.editingImage = false;
        return;
      }

      if (!confirm(`请确认是否将镜像从\n${current}\n更新为\n${updated}？`))
        return;

      $.ajax({
        url: API_ENDPOINTS.deployment.scale,
        method: "POST",
        contentType: "application/json",
        headers: {
          Authorization: "Bearer " + token,
        },
        data: JSON.stringify({
          namespace: this.deployment.metadata.namespace,
          name: this.deployment.metadata.name,
          image: updated,
        }),
        success: (resp) => {
          if (resp.error) {
            alert("❌ 更新镜像失败: " + resp.error);
            return;
          }
          alert("✅ 镜像更新成功！");
          this.container.image = updated;
          this.editingImage = false;
        },
        error: (xhr) => {
          alert("❌ 请求失败: " + (xhr.responseText || xhr.statusText));
        },
      });
    },
  },
});
