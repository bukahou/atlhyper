const getters = {
  sidebar: (state) => state.app.sidebar,
  size: (state) => state.app.size,
  device: (state) => state.app.device,
  visitedViews: (state) => state.tagsView.visitedViews,
  cachedViews: (state) => state.tagsView.cachedViews,
  token: (state) => state.user.token,
  avatar: (state) => state.user.avatar,
  name: (state) => state.user.name,
  introduction: (state) => state.user.introduction,
  roles: (state) => state.user.roles,
  permission_routes: (state) => state.permission.routes,
  errorLogs: (state) => state.errorLog.logs,

  // ✅ 新增 cluster 模块相关
  clusterIds: (state) => state.cluster.clusterIds, // 登录时返回的集群列表
  currentClusterId: (state) => state.cluster.currentId, // 当前选中的集群 ID
};
export default getters;
