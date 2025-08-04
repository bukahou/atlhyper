import request from "@/utils/request";

// 登录
export function login(data) {
  return request({
    url: "/uiapi/auth/login",
    method: "post",
    data,
  });
}

// 注册用户
export function register(data) {
  return request({
    url: "/uiapi/auth/user/register",
    method: "post",
    data,
  });
}

// 修改用户角色
export function changeRole(data) {
  return request({
    url: "/uiapi/auth/user/update-role",
    method: "post",
    data,
  });
}

// 获取用户列表
export function listUsers() {
  return request({
    url: "/uiapi/auth/user/list",
    method: "get",
  });
}

export function updateUserRole(data) {
  return request({
    url: "/uiapi/auth/user/update-role",
    method: "post",
    data,
  });
}

// export function login(data) {
//   return request({
//     url: "/vue-element-admin/user/login",
//     method: "post",
//     data,
//   });
// }

// export function getInfo(token) {
//   return request({
//     url: "/vue-element-admin/user/info",
//     method: "get",
//     params: { token },
//   });
// }

// export function logout() {
//   return request({
//     url: "/vue-element-admin/user/logout",
//     method: "post",
//   });
// }
