import { create } from "zustand";
import type { UserInfo, LoginResponse } from "@/types/auth";

interface AuthStore {
  token: string | null;
  user: UserInfo | null;
  clusterIds: string[];
  isAuthenticated: boolean;
  isLoginDialogOpen: boolean;
  pendingAction: (() => void) | null;

  // Actions
  setLoginData: (data: LoginResponse) => void;
  setToken: (token: string) => void;
  setUser: (user: UserInfo) => void;
  setClusterIds: (clusterIds: string[]) => void;
  logout: () => void;
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

  return {
    token,
    user: userStr ? JSON.parse(userStr) : null,
    clusterIds: clusterIdsStr ? JSON.parse(clusterIdsStr) : [],
    isAuthenticated: !!token,
  };
};

export const useAuthStore = create<AuthStore>((set, get) => ({
  ...getInitialState(),
  isLoginDialogOpen: false,
  pendingAction: null,

  setLoginData: (data: LoginResponse) => {
    localStorage.setItem("token", data.token);
    localStorage.setItem("user", JSON.stringify(data.user));
    localStorage.setItem("clusterIds", JSON.stringify(data.cluster_ids));
    set({
      token: data.token,
      user: data.user,
      clusterIds: data.cluster_ids,
      isAuthenticated: true,
    });
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
