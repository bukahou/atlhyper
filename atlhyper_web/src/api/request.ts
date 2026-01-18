/**
 * Axios 请求封装
 *
 * 适配 Master V2 API
 *
 * 权限控制策略：
 * - 权限验证集中在后端处理
 * - 前端根据后端返回的 HTTP 状态码和错误信息进行相应处理
 * - 401: 未登录或 Token 失效 -> 触发登录流程
 * - 403: 权限不足 -> 显示权限不足提示
 *
 * 响应格式（Master V2）：
 * - 成功: HTTP 200，直接返回 JSON 数据
 * - 错误: HTTP 4xx/5xx，返回 { error: "错误信息" }
 */

import axios, { type AxiosInstance, type AxiosResponse, type InternalAxiosRequestConfig, type AxiosError } from "axios";
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
// Master V2 使用 HTTP 状态码判断成功/失败，不再使用 code: 20000
request.interceptors.response.use(
  (response: AxiosResponse) => {
    // HTTP 2xx 都视为成功，直接返回
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
      console.error("[Request] Error:", status, errorMsg);
    }

    return Promise.reject(error);
  }
);

export default request;

// ============================================================
// 便捷方法
// ============================================================

/**
 * GET 请求
 */
export const get = <T>(url: string, params?: object): Promise<AxiosResponse<T>> => {
  return request.get(url, { params });
};

/**
 * POST 请求
 */
export const post = <T>(url: string, data?: object): Promise<AxiosResponse<T>> => {
  return request.post(url, data);
};

/**
 * PUT 请求
 */
export const put = <T>(url: string, data?: object): Promise<AxiosResponse<T>> => {
  return request.put(url, data);
};

/**
 * DELETE 请求
 */
export const del = <T>(url: string, params?: object): Promise<AxiosResponse<T>> => {
  return request.delete(url, { params });
};
