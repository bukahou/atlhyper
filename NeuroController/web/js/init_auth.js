// ✅ 拦截器：为所有 axios 请求自动添加 Authorization 头（带 JWT Token）
axios.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem("jwt");
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);
