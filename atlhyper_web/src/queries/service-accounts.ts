/**
 * 服务账号管理页的取数与写操作描述
 *
 * 管理域没有 datasource：工厂直接调用 api 层。没有权限时页面整页显示演示数据，
 * 演示数据由页面直接取，不经过这里的查询。
 */
import { mutationOptions, queryOptions, type QueryClient } from "@tanstack/react-query";
import {
  createServiceAccount,
  deleteServiceAccount,
  getServiceAccounts,
  issueAPIToken,
  revokeAPIToken,
  updateServiceAccount,
  type CreateServiceAccountRequest,
  type IssueAPITokenRequest,
  type ServiceAccountLimits,
  type ServiceAccountView,
  type UpdateServiceAccountRequest,
} from "@/api/service-accounts";
import { adminKeys } from "./keys";
import { REFRESH } from "./refresh";

/** 列表查询的结果：账号与后端下发的业务参数 */
export interface ServiceAccountsData {
  accounts: ServiceAccountView[];
  limits: ServiceAccountLimits;
}

/** 让列表重新取数。不等它取完：写操作已经成功，按钮不必为了刷新继续转圈 */
function invalidateServiceAccounts(client: QueryClient): void {
  void client.invalidateQueries({ queryKey: adminKeys.serviceAccounts() });
}

/**
 * 服务账号列表与业务参数。
 *
 * canManage 由调用方用 useSessionReady 与 useCan("user.manage") 得到：没有权限就不发请求。
 * 慢速轮询：令牌的最近使用时间与是否过期都会随时间变化。
 */
export function serviceAccountsQuery(canManage: boolean) {
  return queryOptions({
    queryKey: adminKeys.serviceAccounts(),
    queryFn: async ({ signal }): Promise<ServiceAccountsData> => {
      const { data, limits } = (await getServiceAccounts(signal)).data;
      return { accounts: data, limits };
    },
    refetchInterval: REFRESH.slow,
    staleTime: REFRESH.slow,
    enabled: canManage,
  });
}

/** 新建服务账号；返回新建的那一行。账号名被占时后端返回 409，错误原样抛给调用方 */
export function createServiceAccountMutation() {
  return mutationOptions({
    mutationFn: async (request: CreateServiceAccountRequest) => (await createServiceAccount(request)).data.data,
    onSuccess: (_created, _request, _onMutateResult, context) => invalidateServiceAccounts(context.client),
  });
}

export interface ServiceAccountChange {
  accountId: number;
  request: UpdateServiceAccountRequest;
}

/** 修改显示名、等级与启停（整体替换）；返回修改后的那一行 */
export function updateServiceAccountMutation() {
  return mutationOptions({
    mutationFn: async ({ accountId, request }: ServiceAccountChange) =>
      (await updateServiceAccount(accountId, request)).data.data,
    onSuccess: (_updated, _change, _onMutateResult, context) => invalidateServiceAccounts(context.client),
  });
}

/** 删除服务账号（它的令牌随之删除） */
export function deleteServiceAccountMutation() {
  return mutationOptions({
    mutationFn: async (accountId: number) => {
      await deleteServiceAccount(accountId);
    },
    onSuccess: (_nothing, _accountId, _onMutateResult, context) => invalidateServiceAccounts(context.client),
  });
}

export interface APITokenRevocation {
  accountId: number;
  tokenId: number;
}

/** 吊销一枚令牌（行保留，立即失效） */
export function revokeAPITokenMutation() {
  return mutationOptions({
    mutationFn: async ({ accountId, tokenId }: APITokenRevocation) => {
      await revokeAPIToken(accountId, tokenId);
    },
    onSuccess: (_nothing, _revocation, _onMutateResult, context) => invalidateServiceAccounts(context.client),
  });
}

export interface APITokenIssueRequest {
  accountId: number;
  request: IssueAPITokenRequest;
  /** 接收令牌明文。只会被调用一次；调用方把明文放进弹窗的本地状态，关闭弹窗即丢弃 */
  deliverToken: (token: string) => void;
}

/** 一次签发里可以留存的部分：前缀与到期时间（列表里本来就显示它们） */
export interface IssuedAPITokenSummary {
  tokenPrefix: string;
  expiresAt: string;
}

/**
 * 给服务账号签发一枚令牌。
 *
 * ⛔ 明文不作为 mutation 的返回值：mutation 的结果会留在 MutationCache 里，
 * 开发者工具与任何拿得到 QueryClient 的代码都读得到。所以明文在取数函数里直接交给调用方的 deliverToken，
 * 缓存里只留前缀与到期时间。
 */
export function issueAPITokenMutation() {
  return mutationOptions({
    mutationFn: async ({ accountId, request, deliverToken }: APITokenIssueRequest): Promise<IssuedAPITokenSummary> => {
      const { token, tokenPrefix, expiresAt } = (await issueAPIToken(accountId, request)).data.data;
      deliverToken(token);
      return { tokenPrefix, expiresAt };
    },
    onSuccess: (_summary, _request, _onMutateResult, context) => invalidateServiceAccounts(context.client),
  });
}
