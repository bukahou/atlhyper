// src/store/modules/cluster.js
const CURR_KEY = "atlhyper_cluster_id";

const state = {
  clusterIds: [], // 登录返回的列表
  currentId: "", // 当前选中的集群ID（单一来源）
};

const getters = {
  clusterIds: (s) => s.clusterIds,
  currentId: (s) => s.currentId,
};

const mutations = {
  SET_CLUSTER_IDS(state, ids) {
    state.clusterIds = Array.isArray(ids) ? ids.slice() : [];
  },
  SET_CURRENT_ID(state, id) {
    state.currentId = id || "";
  },
};

const actions = {
  // 登录后初始化
  initAfterLogin({ commit }, ids) {
    const list = Array.isArray(ids) ? ids : [];
    commit("SET_CLUSTER_IDS", list);

    const saved = localStorage.getItem(CURR_KEY);
    let use = "";
    if (saved && (!list.length || list.includes(saved))) use = saved;
    else if (list.length) use = list[0];

    commit("SET_CURRENT_ID", use);
    if (use) localStorage.setItem(CURR_KEY, use);

    // 可选：广播一个全局事件，兼容旧页面
    if (use)
      window.dispatchEvent(new CustomEvent("cluster-changed", { detail: use }));
  },

  // 顶部选择切换
  setCurrentId({ state, commit }, id) {
    if (!id || id === state.currentId) return;
    if (state.clusterIds.length && !state.clusterIds.includes(id)) return;
    commit("SET_CURRENT_ID", id);
    localStorage.setItem(CURR_KEY, id);
    window.dispatchEvent(new CustomEvent("cluster-changed", { detail: id }));
  },

  // 刷新后兜底恢复
  ensureInitialized({ state, commit }) {
    if (state.currentId) return;
    const saved = localStorage.getItem(CURR_KEY);
    if (saved) commit("SET_CURRENT_ID", saved);
  },
};

export default { namespaced: true, state, getters, mutations, actions };
export { CURR_KEY };
