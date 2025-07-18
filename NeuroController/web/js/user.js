// ========================================================================
// ✅ 加载用户列表
// ========================================================================
function loadUserList() {
  $.ajax({
    url: API_ENDPOINTS.auth.listUsers,
    method: "GET",
    headers: {
      Authorization: "Bearer " + localStorage.getItem("jwt"),
    },
    success: function (data) {
      const tbody = $("#user-table-body");
      tbody.empty();

      data.forEach(function (user) {
        const row = `
          <tr>
            <td>${user.Username}</td>
            <td>${user.DisplayName || "—"}</td>
            <td>${user.Email || "—"}</td>
            <td>${user.CreatedAt || "—"}</td>
            <td>${user.Role === 3 ? "管理员" : "普通用户"}</td>
          </tr>
        `;
        tbody.append(row);
      });
    },
    error: function (xhr) {
      alert("❌ 加载用户失败：" + (xhr.responseJSON?.error || xhr.statusText));
    },
  });
}

// ========================================================================
// ✅ 注册新用户
// ========================================================================
function registerUser() {
  const payload = {
    username: $("#reg-username").val().trim(),
    password: $("#reg-password").val(),
    display_name: $("#reg-display-name").val().trim(),
    email: $("#reg-email").val().trim(),
    role: parseInt($("#reg-role").val()),
  };

  if (!payload.username || !payload.password) {
    alert("❗ 请填写用户名和密码");
    return;
  }

  $.ajax({
    url: API_ENDPOINTS.auth.register,
    method: "POST",
    contentType: "application/json",
    headers: {
      Authorization: "Bearer " + localStorage.getItem("jwt"),
    },
    data: JSON.stringify(payload),
    success: function (res) {
      alert("✅ 注册成功！");
      $("#register-form")[0].reset();
      loadUserList();
    },
    error: function (xhr) {
      alert("❌ 注册失败：" + (xhr.responseJSON?.error || xhr.statusText));
    },
  });
}

// 页面加载时立即加载用户列表
$(document).ready(function () {
  loadUserList();
});

function loadUserList() {
  $.ajax({
    url: API_ENDPOINTS.auth.listUsers,
    method: "GET",
    headers: {
      Authorization: "Bearer " + localStorage.getItem("jwt"),
    },
    success: function (data) {
      const tbody = $("#user-table-body");
      tbody.empty();

      data.forEach(function (user) {
        const roleOptions = `
          <select class="form-control form-control-sm role-select" data-id="${
            user.ID
          }">
            <option value="1" ${
              user.Role === 1 ? "selected" : ""
            }>普通用户</option>
            <option value="3" ${
              user.Role === 3 ? "selected" : ""
            }>管理员</option>
          </select>
        `;

        const row = `
          <tr>
            <td>${user.Username}</td>
            <td>${user.DisplayName || "—"}</td>
            <td>${user.Email || "—"}</td>
            <td>${user.CreatedAt || "—"}</td>
            <td>${roleOptions}</td>
            <td>
              <button class="btn btn-xs btn-warning btn-update-role" data-id="${
                user.ID
              }">
                更新
              </button>
            </td>
          </tr>
        `;
        tbody.append(row);
      });

      // ✅ 为“更新”按钮绑定点击事件（每次刷新后都要重新绑定）
      $(".btn-update-role")
        .off("click")
        .on("click", function () {
          const userId = $(this).data("id");
          const selectedRole = $(`.role-select[data-id=${userId}]`).val();

          // 弹窗确认
          if (
            !confirm(
              `确认要将用户 ID=${userId} 的权限修改为 ${
                selectedRole == 3 ? "管理员" : "普通用户"
              } 吗？`
            )
          ) {
            return;
          }

          // 发起请求
          $.ajax({
            url: API_ENDPOINTS.auth.updateRole,
            method: "POST",
            contentType: "application/json",
            headers: {
              Authorization: "Bearer " + localStorage.getItem("jwt"),
            },
            data: JSON.stringify({ id: userId, role: parseInt(selectedRole) }),
            success: function () {
              alert("✅ 权限修改成功");
              loadUserList(); // 重新加载列表
            },
            error: function (xhr) {
              alert(
                "❌ 修改失败：" + (xhr.responseJSON?.error || xhr.statusText)
              );
            },
          });
        });
    },
    error: function (xhr) {
      alert("❌ 加载用户失败：" + (xhr.responseJSON?.error || xhr.statusText));
    },
  });
}
