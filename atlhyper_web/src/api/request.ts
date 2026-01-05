/**
 * Axios 请求封装
 *
 * 权限控制策略：
 * - 权限验证集中在后端处理
 * - 前端根据后端返回的 HTTP 状态码和错误信息进行相应处理
 * - 401: 未登录或 Token 失效 -> 触发登录流程
 * - 403: 权限不足 -> 显示权限不足提示
 */

import axios, { type AxiosInstance, type AxiosResponse, type InternalAxiosRequestConfig, type AxiosError } from "axios";
import type { ApiResponse } from "@/types";
import { env } from "@/config/env";

// ============================================================
// 权限错误类型定义
// ============================================================
export interface AuthError {
  type: "unauthorized" | "forbidden";
  message: string;
  originalError?: AxiosError;
}

// 权限错误事件监听器类型
type AuthErrorListener = (error: AuthError) => void;

// 权限错误事件管理器（用于跨组件通信）
class AuthErrorEventManager {
  private listeners: AuthErrorListener[] = [];

  subscribe(listener: AuthErrorListener) {
    this.listeners.push(listener);
    return () => {
      this.listeners = this.listeners.filter((l) => l !== listener);
    };
  }

  emit(error: AuthError) {
    this.listeners.forEach((listener) => listener(error));
  }
}

export const authErrorManager = new AuthErrorEventManager();

// ============================================================
// Axios 实例配置
// ============================================================
const request: AxiosInstance = axios.create({
  baseURL: env.apiUrl,
  timeout: 30000,
  headers: {
    "Content-Type": "application/json",
  },
});

// 请求拦截器
request.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    // 从 localStorage 获取 token
    if (typeof window !== "undefined") {
      const token = localStorage.getItem("token");
      if (token && config.headers) {
        config.headers["Authorization"] = `Bearer ${token}`;
      }
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// 响应拦截器
request.interceptors.response.use(
  (response: AxiosResponse<ApiResponse>) => {
    const res = response.data;

    // 成功码为 20000
    if (res.code !== 20000) {
      console.error("API Error:", res.message);
      return Promise.reject(new Error(res.message || "请求失败"));
    }

    return response;
  },
  (error: AxiosError<{ error?: string; message?: string }>) => {
    const status = error.response?.status;
    const errorMsg = error.response?.data?.error || error.response?.data?.message || error.message;

    // 处理权限相关错误
    if (status === 401) {
      // 未登录或 Token 失效
      const authError: AuthError = {
        type: "unauthorized",
        message: errorMsg || "请先登录",
        originalError: error,
      };
      authErrorManager.emit(authError);
      console.warn("[Auth] 401 Unauthorized:", errorMsg);
    } else if (status === 403) {
      // 权限不足
      const authError: AuthError = {
        type: "forbidden",
        message: errorMsg || "权限不足",
        originalError: error,
      };
      authErrorManager.emit(authError);
      console.warn("[Auth] 403 Forbidden:", errorMsg);
    } else {
      console.error("Request Error:", error.message);
    }

    return Promise.reject(error);
  }
);

export default request;

// 便捷方法
export const get = <T, P = Record<string, unknown>>(
  url: string,
  params?: P
): Promise<AxiosResponse<ApiResponse<T>>> => {
  return request.get(url, { params });
};

export const post = <T, D = Record<string, unknown>>(
  url: string,
  data?: D
): Promise<AxiosResponse<ApiResponse<T>>> => {
  return request.post(url, data);
};
