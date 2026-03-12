/**
 * GitHub 集成 API
 *
 * 功能一：连接管理
 * 功能二：仓库查看
 */

import { get, post, del } from "./request";

// ============================================================
// 响应类型
// ============================================================

interface ConnectionResponse {
  message: string;
  data: {
    connected: boolean;
    accountLogin: string;
    avatarUrl: string;
    installationId: number;
  };
}

interface ConnectResponse {
  data: {
    authUrl?: string;
    connected?: boolean;
    accountLogin?: string;
    avatarUrl?: string;
    installationId?: number;
  };
}

interface CallbackResponse {
  data: {
    connected: boolean;
    accountLogin: string;
    avatarUrl: string;
    installationId: number;
  };
}

interface ReposResponse {
  data: {
    fullName: string;
    defaultBranch: string;
    private: boolean;
  }[];
}

interface RepoDirsResponse {
  data: string[];
}

// ============================================================
// API 方法
// ============================================================

/**
 * 获取 GitHub 连接状态
 * GET /api/github/connection
 */
export function getConnection() {
  return get<ConnectionResponse>("/api/github/connection");
}

/**
 * 发起 OAuth 连接
 * POST /api/github/connect
 */
export function connect() {
  return post<ConnectResponse>("/api/github/connect");
}

/**
 * GitHub App 安装回调
 * POST /api/github/callback
 */
export function callback(installationId: number, setupAction: string) {
  return post<CallbackResponse>("/api/github/callback", {
    installation_id: installationId,
    setup_action: setupAction,
  });
}

/**
 * 断开连接
 * DELETE /api/github/connection
 */
export function disconnect() {
  return del<{ message: string }>("/api/github/connection");
}

/**
 * 获取已授权仓库列表
 * GET /api/github/repos
 */
export function getRepos() {
  return get<ReposResponse>("/api/github/repos");
}

/**
 * 获取仓库顶层目录
 * GET /api/github/repos/:repo/dirs
 */
export function getRepoDirs(repo: string) {
  return get<RepoDirsResponse>(
    `/api/github/repos/${encodeURIComponent(repo)}/dirs`
  );
}
