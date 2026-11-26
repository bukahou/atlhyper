/**
 * 服务账号与访问令牌 API（需要 user.manage）
 *
 * 服务账号是不能登录的账号，凭访问令牌调用 MCP 入口（不是 K8s 的 ServiceAccount）。
 * 类型由 api/openapi.yaml 生成（types/api.gen.ts），这里只起常用的名字。
 */

import type { components } from "@/types/api.gen";
import { del, get, post, put } from "./request";

// ============================================================
// 类型
// ============================================================

type Schemas = components["schemas"];

/** 服务账号管理页的一行：账号 + 它的令牌元数据（不含明文与哈希） */
export type ServiceAccountView = Schemas["ServiceAccountView"];
export type APITokenView = Schemas["APITokenView"];
/** 令牌状态由后端按当前时间判定（吊销优先于过期） */
export type APITokenStatus = Schemas["APITokenStatus"];
/** 业务参数：可选等级、每账号令牌上限、有效期上限（后端下发，前端不另写一份） */
export type ServiceAccountLimits = Schemas["ServiceAccountLimits"];
export type CreateServiceAccountRequest = Schemas["CreateServiceAccountRequest"];
export type UpdateServiceAccountRequest = Schemas["UpdateServiceAccountRequest"];
export type IssueAPITokenRequest = Schemas["IssueAPITokenRequest"];
/**
 * 一次签发的结果。
 *
 * token 是明文，只在签发的这一次响应里出现，之后任何接口都取不回它。
 * ⛔ 不得写进查询缓存、localStorage 或日志。
 */
export type APITokenIssued = Schemas["APITokenIssued"];

type ServiceAccountListResponse = Schemas["ServiceAccountListResponse"];
type ServiceAccountResponse = Schemas["ServiceAccountResponse"];
type APITokenIssuedResponse = Schemas["APITokenIssuedResponse"];
type MessageResponse = Schemas["MessageResponse"];

// ============================================================
// API 方法
// ============================================================

const SERVICE_ACCOUNTS_PATH = "/api/v2/admin/service-accounts";

function accountPath(accountId: number): string {
  return `${SERVICE_ACCOUNTS_PATH}/${accountId}`;
}

/**
 * 服务账号列表（按账号名排序）与业务参数
 * GET /api/v2/admin/service-accounts
 */
export function getServiceAccounts(signal?: AbortSignal) {
  return get<ServiceAccountListResponse>(SERVICE_ACCOUNTS_PATH, undefined, signal);
}

/**
 * 新建服务账号（成功 201；账号名被占 409；输入不合规 400）
 * POST /api/v2/admin/service-accounts
 */
export function createServiceAccount(request: CreateServiceAccountRequest) {
  return post<ServiceAccountResponse>(SERVICE_ACCOUNTS_PATH, request);
}

/**
 * 修改显示名、等级与启停（整体替换；不存在 404；输入不合规 400）
 * PUT /api/v2/admin/service-accounts/{id}
 */
export function updateServiceAccount(accountId: number, request: UpdateServiceAccountRequest) {
  return put<ServiceAccountResponse>(accountPath(accountId), request);
}

/**
 * 删除服务账号，它的令牌随之删除（不存在 404）
 * DELETE /api/v2/admin/service-accounts/{id}
 */
export function deleteServiceAccount(accountId: number) {
  return del<MessageResponse>(accountPath(accountId));
}

/**
 * 签发访问令牌（成功 201；账号不存在 404；未吊销的令牌已达上限 409）
 * POST /api/v2/admin/service-accounts/{id}/tokens
 */
export function issueAPIToken(accountId: number, request: IssueAPITokenRequest) {
  return post<APITokenIssuedResponse>(`${accountPath(accountId)}/tokens`, request);
}

/**
 * 吊销访问令牌（行保留，立即失效；该账号下没有这枚令牌 404）
 * DELETE /api/v2/admin/service-accounts/{id}/tokens/{tokenId}
 */
export function revokeAPIToken(accountId: number, tokenId: number) {
  return del<MessageResponse>(`${accountPath(accountId)}/tokens/${tokenId}`);
}
