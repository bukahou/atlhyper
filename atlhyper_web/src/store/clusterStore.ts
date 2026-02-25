import { create } from "zustand";

interface ClusterStore {
  // 可用集群列表
  clusterIds: string[];
  // 当前选中的集群
  currentClusterId: string;
  // 是否已初始化
  initialized: boolean;

  // Actions
  setClusterIds: (ids: string[]) => void;
  setCurrentCluster: (id: string) => void;
  initialize: () => void;
}

// 从 localStorage 获取初始状态
const getInitialState = () => {
  if (typeof window === "undefined") {
    return {
      clusterIds: [],
      currentClusterId: "",
      initialized: false,
    };
  }

  // 尝试从 localStorage 读取
  const savedClusterIds = localStorage.getItem("clusterIds");
  const savedCurrentCluster = localStorage.getItem("currentClusterId");

  let clusterIds: string[] = [];
  if (savedClusterIds) {
    try {
      const parsed = JSON.parse(savedClusterIds);
      if (Array.isArray(parsed) && parsed.length > 0) {
        clusterIds = parsed;
      }
    } catch {
      // 解析失败，使用空列表
    }
  }

  // 当前集群：优先使用保存的，否则使用列表第一个
  let currentClusterId = savedCurrentCluster || clusterIds[0] || "";

  // 确保当前集群在列表中
  if (clusterIds.length > 0 && !clusterIds.includes(currentClusterId)) {
    currentClusterId = clusterIds[0];
  }

  return {
    clusterIds,
    currentClusterId,
    initialized: clusterIds.length > 0,
  };
};

export const useClusterStore = create<ClusterStore>((set, get) => ({
  ...getInitialState(),

  setClusterIds: (ids: string[]) => {
    if (ids.length === 0) return;
    localStorage.setItem("clusterIds", JSON.stringify(ids));

    // 如果当前集群不在新列表中，切换到第一个
    const { currentClusterId } = get();
    let newCurrentId = currentClusterId;
    if (!ids.includes(currentClusterId)) {
      newCurrentId = ids[0];
      localStorage.setItem("currentClusterId", newCurrentId);
    }

    set({
      clusterIds: ids,
      currentClusterId: newCurrentId,
      initialized: true,
    });
  },

  setCurrentCluster: (id: string) => {
    const { clusterIds } = get();
    // 验证集群 ID 有效
    if (clusterIds.includes(id)) {
      localStorage.setItem("currentClusterId", id);
      set({ currentClusterId: id });
    }
  },

  initialize: () => {
    if (typeof window === "undefined") return;

    const state = getInitialState();
    set({
      clusterIds: state.clusterIds,
      currentClusterId: state.currentClusterId,
      initialized: state.initialized,
    });
  },
}));
