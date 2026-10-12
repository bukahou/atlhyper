import { create } from "zustand";
import type { UserInfo, LoginResponse, Permission } from "@/types/auth";
import { getMe } from "@/api/auth";

interface AuthStore {
  token: string | null;
  user: UserInfo | null;
  clusterIds: string[];
  isAuthenticated: boolean;
  // 权限只来自 /api/v2/auth/me（后端唯一权限源）；加载前一律视为无权限
  permissions: Permission[];
  meLoaded: boolean;
  isLoginDialogOpen: boolean;
  pendingAction: (() => void) | null;

  // Actions
  setLoginData: (data: LoginResponse) => void;
  setToken: (token: string) => void;
  setUser: (user: UserInfo) => void;
  setClusterIds: (clusterIds: string[]) => void;
  logout: () => void;
  refreshMe: () => Promise<void>;
  openLoginDialog: (pendingAction?: () => void) => void;
  closeLoginDialog: () => void;
  executePendingAction: () => void;
}

// 从 localStorage 初始化状态
const getInitialState = () => {
  if (typeof window === "undefined") {
    return { token: null, user: null, clusterIds: [], isAuthenticated: false };
  }

  const token = localStorage.getItem("token");
  const userStr = localStorage.getItem("user");
  const clusterIdsStr = localStorage.getItem("clusterIds");

  const parseJSON = <T>(str: string | null, fallback: T): T => {
    if (!str || str === "undefined" || str === "null") return fallback;
    try { return JSON.parse(str); } catch { return fallback; }
  };

  return {
    token,
    user: parseJSON(userStr, null),
    clusterIds: parseJSON(clusterIdsStr, []),
    isAuthenticated: !!token,
  };
};

export const useAuthStore = create<AuthStore>((set, get) => ({
  ...getInitialState(),
  permissions: [],
  meLoaded: false,
  isLoginDialogOpen: false,
  pendingAction: null,

  setLoginData: (data: LoginResponse) => {
    localStorage.setItem("token", data.token);
    localStorage.setItem("user", JSON.stringify(data.user));
    localStorage.setItem("clusterIds", JSON.stringify(data.cluster_ids || []));
    set({
      token: data.token,
      user: data.user,
      clusterIds: data.cluster_ids || [],
      isAuthenticated: true,
    });
    void get().refreshMe();
  },

  setToken: (token: string) => {
    localStorage.setItem("token", token);
    set({ token, isAuthenticated: true });
  },

  setUser: (user: UserInfo) => {
    localStorage.setItem("user", JSON.stringify(user));
    set({ user });
  },

  setClusterIds: (clusterIds: string[]) => {
    localStorage.setItem("clusterIds", JSON.stringify(clusterIds));
    set({ clusterIds });
  },

  logout: () => {
    localStorage.removeItem("token");
    localStorage.removeItem("user");
    localStorage.removeItem("clusterIds");
    set({ token: null, user: null, clusterIds: [], isAuthenticated: false });
    void get().refreshMe(); // 回到访客权限
  },

  refreshMe: async () => {
    try {
      const res = await getMe();
      const me = res.data.data;
      if (!me.user && get().token) {
        // 本地有 token 但后端不认（过期 / 禁用 / 已删除）→ 清掉本地登录态
        localStorage.removeItem("token");
        localStorage.removeItem("user");
        localStorage.removeItem("clusterIds");
        set({ token: null, user: null, clusterIds: [], isAuthenticated: false });
      } else if (me.user) {
        localStorage.setItem("user", JSON.stringify(me.user));
        set({ user: me.user });
      }
      set({ permissions: me.permissions ?? [], meLoaded: true });
    } catch {
      // 拿不到权限时保持「无权限」：失败即关
      set({ permissions: [], meLoaded: true });
    }
  },

  openLoginDialog: (pendingAction?: () => void) => {
    set({ isLoginDialogOpen: true, pendingAction: pendingAction || null });
  },

  closeLoginDialog: () => {
    set({ isLoginDialogOpen: false, pendingAction: null });
  },

  executePendingAction: () => {
    const { pendingAction } = get();
    if (pendingAction) {
      pendingAction();
      set({ pendingAction: null });
    }
  },
}));
