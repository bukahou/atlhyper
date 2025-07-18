// js/user_info.js

document.addEventListener("DOMContentLoaded", function () {
  const username = localStorage.getItem("username") || "未知用户";
  const role = parseInt(localStorage.getItem("role"), 10);

  const roleText = role === 3 ? "管理员" : "普通用户";

  // 设置显示内容
  const usernameEl = document.getElementById("username-display");
  const roleEl = document.getElementById("role-display");
  if (usernameEl) usernameEl.textContent = username;
  if (roleEl) roleEl.textContent = roleText;

  // 退出登录
  const logoutLink = document.getElementById("logout-link");
  if (logoutLink) {
    logoutLink.addEventListener("click", function (e) {
      e.preventDefault();
      if (confirm("确定要退出登录吗？")) {
        localStorage.clear(); // 清除全部 localStorage
        window.location.href = "login.html"; // 跳转到登录页
      }
    });
  }
});
