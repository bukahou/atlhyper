// import { asyncRoutes, constantRoutes } from "@/router";

// const state = {
//   routes: [],
//   addRoutes: [],
// };

// const mutations = {
//   SET_ROUTES: (state, routes) => {
//     state.addRoutes = routes;
//     state.routes = constantRoutes.concat(routes);
//   },
// };

// const actions = {
//   generateRoutes({ commit }) {
//     return new Promise((resolve) => {
//       const accessedRoutes = asyncRoutes || [];
//       commit("SET_ROUTES", accessedRoutes);
//       resolve(accessedRoutes);
//     });
//   },
// };

// export default {
//   namespaced: true,
//   state,
//   mutations,
//   actions,
// };

import { constantRoutes } from '@/router'

const state = {
  routes: [],
  addRoutes: []
}

const mutations = {
  SET_ROUTES(state, routes) {
    state.addRoutes = routes
    state.routes = constantRoutes.concat(routes)
  }
}

const actions = {
  generateRoutes({ commit }) {
    return new Promise((resolve) => {
      const accessedRoutes = [] // 不做动态路由
      commit('SET_ROUTES', accessedRoutes)
      resolve(accessedRoutes)
    })
  }
}

export default {
  namespaced: true,
  state,
  mutations,
  actions
}
