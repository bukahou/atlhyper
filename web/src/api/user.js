import request from '@/utils/request'

// 登录
export function login(data) {
  return request({
    url: '/uiapi/auth/login',
    method: 'post',
    data
  })
}

// 注册用户
export function register(data) {
  return request({
    url: '/uiapi/auth/user/register',
    method: 'post',
    data
  })
}

// 修改用户角色
export function changeRole(data) {
  return request({
    url: '/uiapi/auth/user/update-role',
    method: 'post',
    data
  })
}

// 获取用户列表
export function listUsers() {
  return request({
    url: '/uiapi/auth/user/list',
    method: 'get'
  })
}

// 修改用户角色
export function updateUserRole(data) {
  return request({
    url: '/uiapi/auth/user/update-role',
    method: 'post',
    data
  })
}

// 获取审计日志列表
export function listUserAuditLogs() {
  return request({
    url: '/uiapi/auth/userauditlogs/list',
    method: 'get'
  })
}

// ====================================================================待办事项====================================================================
// 获取所有待办事项
export function listUserTodos() {
  return request({
    url: '/uiapi/user/todos/all',
    method: 'get'
  })
}

// 根据用户名获取待办事项
export function getUserTodosByUsername(username) {
  return request({
    url: '/uiapi/user/todos/by-username',
    method: 'post',
    data: {
      username: username
    }
  })
}

// 删除待办事项
export function deleteUserTodo(id) {
  return request({
    url: '/uiapi/user/todo/delete',
    method: 'post',
    data: {
      id: id
    }
  })
}

//
export function createUserTodo(data) {
  return request({
    url: '/uiapi/user/todo/create',
    method: 'post',
    data: cleanPayload({
      username: data.username,
      title: data.title,
      content: data.content,
      is_done: data.is_done, // 0/1
      priority: data.priority, // 1/2/3，后端默认 2
      category: data.category,
      due_date: data.due_date // 'YYYY-MM-DD'；不传则不发送
    })
  })
}
// 创建待办事项
export function updateUserTodo(data) {
  return request({
    url: '/uiapi/user/todo/update',
    method: 'post',
    data: cleanPayload({
      id: data.id, // 必填
      title: data.title,
      content: data.content,
      is_done: data.is_done, // 0/1
      priority: data.priority, // 1/2/3
      deleted: data.deleted, // 0/1
      due_date: data.due_date, // 传 "" => 清空；不传 => 不修改
      category: data.category
    })
  })
}

function cleanPayload(obj) {
  const out = {}
  Object.keys(obj || {}).forEach(function(k) {
    const v = obj[k]
    if (v !== undefined) out[k] = v // 保留 null 和 ""（比如 due_date 清空要用 ""）
  })
  return out
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
