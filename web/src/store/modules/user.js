// import { login } from "@/api/user";
// import { getToken, setToken, removeToken } from "@/utils/auth";
// import router, { resetRouter } from "@/router";

// const state = {
//   token: getToken(),
//   name: "",
//   displayName: "",
//   avatar: "",
//   introduction: "",
//   roles: [],
// };

// const mutations = {
//   SET_TOKEN: (state, token) => {
//     state.token = token;
//   },
//   SET_INTRODUCTION: (state, introduction) => {
//     state.introduction = introduction;
//   },
//   SET_NAME: (state, name) => {
//     state.name = name;
//   },
//   SET_AVATAR: (state, avatar) => {
//     state.avatar = avatar;
//   },
//   SET_ROLES: (state, roles) => {
//     state.roles = roles;
//   },
//   SET_DISPLAY_NAME: (state, displayName) => {
//     state.displayName = displayName;
//   },
// };

// const actions = {
//   // user login
//   login({ commit }, userInfo) {
//     const { username, password } = userInfo;
//     return new Promise((resolve, reject) => {
//       login({ username: username.trim(), password: password })
//         .then((response) => {
//           const { data } = response;
//           if (!data.token || !data.user) {
//             return reject(new Error("登录响应缺少 token 或 user"));
//           }

//           const rolesMap = {
//             0: "disabled",
//             1: "user",
//             2: "admin",
//             3: "super-admin",
//           };

//           const rolesString = rolesMap[data.user.role] || "guest";
//           commit("SET_TOKEN", data.token);
//           commit("SET_NAME", data.user.username);
//           commit("SET_DISPLAY_NAME", data.user.displayName); // 可选填显示名称
//           commit("SET_ROLES", [rolesString]); // ✅ 注意：需要是数组
//           commit("SET_AVATAR", require("@/assets/img/avatar.png")); // 可选填默认头像
//           commit("SET_INTRODUCTION", ""); // 可选
//           setToken(data.token);
//           resolve();
//         })
//         .catch((error) => {
//           reject(error);
//         });
//     });
//   },
//   logout({ commit, dispatch }) {
//     return new Promise((resolve) => {
//       commit("SET_TOKEN", "");
//       commit("SET_ROLES", []);
//       removeToken();
//       resetRouter();

//       // 清空所有打开的标签页
//       dispatch("tagsView/delAllViews", null, { root: true });

//       resolve();
//     });
//   },

//   // remove token
//   resetToken({ commit }) {
//     return new Promise((resolve) => {
//       commit("SET_TOKEN", "");
//       commit("SET_ROLES", []);
//       removeToken();
//       resolve();
//     });
//   },

//   // dynamically modify permissions
//   async changeRoles({ commit, dispatch }, role) {
//     const token = role + "-token";

//     commit("SET_TOKEN", token);
//     setToken(token);

//     const { roles } = await dispatch("getInfo");

//     resetRouter();

//     // generate accessible routes map based on roles
//     const accessRoutes = await dispatch("permission/generateRoutes", roles, {
//       root: true,
//     });
//     // dynamically add accessible routes
//     router.addRoutes(accessRoutes);

//     // reset visited views and cached views
//     dispatch("tagsView/delAllViews", null, { root: true });
//   },
// };

// export default {
//   namespaced: true,
//   state,
//   mutations,
//   actions,
// };

// src/store/modules/user.js
import { login } from '@/api/user'
import { getToken, setToken, removeToken } from '@/utils/auth'
import router, { resetRouter } from '@/router'

const state = {
  token: getToken(),
  name: '',
  displayName: '',
  avatar: '',
  introduction: '',
  roles: [],
  // ✅ 可选：如果你不再需要从 user 里读集群列表，可以删掉这行
  clusterIds: []
}

const mutations = {
  SET_TOKEN: (state, token) => {
    state.token = token
  },
  SET_INTRODUCTION: (state, introduction) => {
    state.introduction = introduction
  },
  SET_NAME: (state, name) => {
    state.name = name
  },
  SET_AVATAR: (state, avatar) => {
    state.avatar = avatar
  },
  SET_ROLES: (state, roles) => {
    state.roles = roles
  },
  SET_DISPLAY_NAME: (state, displayName) => {
    state.displayName = displayName
  },

  // ✅ 如果还想在 user 里保留一份列表（可选）
  SET_CLUSTER_IDS: (state, ids) => {
    state.clusterIds = Array.isArray(ids) ? ids : []
  }
}

const actions = {
  // user login
  login({ commit, dispatch }, userInfo) {
    const { username, password } = userInfo
    return new Promise((resolve, reject) => {
      login({ username: username.trim(), password })
        .then((response) => {
          const { data } = response
          if (!data.token || !data.user) {
            return reject(new Error('登录响应缺少 token 或 user'))
          }

          const rolesMap = {
            0: 'disabled',
            1: 'user',
            2: 'admin',
            3: 'super-admin'
          }
          const rolesString = rolesMap[data.user.role] || 'guest'

          // 基本资料
          commit('SET_TOKEN', data.token)
          setToken(data.token)
          commit('SET_NAME', data.user.username)
          commit('SET_DISPLAY_NAME', data.user.displayName)
          commit('SET_ROLES', [rolesString])
          commit('SET_AVATAR', require('@/assets/img/avatar.png'))
          commit('SET_INTRODUCTION', '')

          // ✅ 把 cluster_ids 交给 cluster 模块进行初始化（单一来源）
          const list = Array.isArray(data.cluster_ids) ? data.cluster_ids : []
          commit('SET_CLUSTER_IDS', list) // （可选）如果你还想在 user 里留一份
          dispatch('cluster/initAfterLogin', list, { root: true })

          resolve()
        })
        .catch(reject)
    })
  },

  logout({ commit, dispatch }) {
    return new Promise((resolve) => {
      // 清用户信息
      commit('SET_TOKEN', '')
      commit('SET_ROLES', [])
      removeToken()
      resetRouter()

      // ✅ 清除 tabs
      dispatch('tagsView/delAllViews', null, { root: true })

      // ✅ 可选：如果你在 cluster 模块里想做彻底清理，可以加一个 action
      // dispatch("cluster/clear", null, { root: true });

      resolve()
    })
  },

  resetToken({ commit /*, dispatch*/ }) {
    return new Promise((resolve) => {
      commit('SET_TOKEN', '')
      commit('SET_ROLES', [])
      removeToken()

      // ✅ 同上，可选
      // dispatch("cluster/clear", null, { root: true });

      resolve()
    })
  },

  // dynamically modify permissions（保持原样）
  async changeRoles({ commit, dispatch }, role) {
    const token = role + '-token'
    commit('SET_TOKEN', token)
    setToken(token)

    const { roles } = await dispatch('getInfo')
    resetRouter()

    const accessRoutes = await dispatch('permission/generateRoutes', roles, {
      root: true
    })
    router.addRoutes(accessRoutes)

    dispatch('tagsView/delAllViews', null, { root: true })
  }
}

export default {
  namespaced: true,
  state,
  mutations,
  actions
}
