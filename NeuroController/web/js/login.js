document.getElementById("login-button").addEventListener("click", function () {
  const username = document.getElementById("login-username").value.trim();
  const password = document.getElementById("login-password").value.trim();

  if (!username || !password) {
    alert("请输入用户名和密码");
    return;
  }

  axios
    .post(API_ENDPOINTS.auth.login, {
      username: username,
      password: password,
    })
    .then((response) => {
      const { token, user } = response.data;

      try {
        // 储存 token
        localStorage.setItem("jwt", token);
        localStorage.setItem("username", user.username);
        localStorage.setItem("role", user.role);

        // 🔍 从 localStorage 中读取并验证
        const savedToken = localStorage.getItem("jwt");
        if (!savedToken) {
          alert("❌ token 写入 localStorage 失败！");
          return;
        }

        // ✅ 提示成功，并展示截断后的 token（避免太长）
        alert(
          `✅ 登录成功！Token 写入成功\n前段片段: ${savedToken.slice(0, 16)}...`
        );
        window.location.href = "index.html";
      } catch (err) {
        alert("❌ Token 写入发生异常：" + err.message);
        console.error("写入 localStorage 错误", err);
      }
    })
    .catch((error) => {
      const msg =
        error.response?.data?.error ||
        error.message ||
        "登录失败，请检查用户名密码";
      alert("❌ " + msg);
    });
});
