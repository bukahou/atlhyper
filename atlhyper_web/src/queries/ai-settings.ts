/**
 * AI 设置页的取数与写操作描述（Provider、角色分配、预算、调用记录、全局设置）
 *
 * 没有 datasource：工厂直接调用 api 层。读需要 ai.settings.read —— 有没有权限由调用方用 useCan 得到后
 * 传进来；没有权限时页面整页显示演示数据（由页面自己取，不经过这里）。
 */
import { infiniteQueryOptions, mutationOptions, queryOptions, type QueryClient } from "@tanstack/react-query";
import {
  createProvider,
  deleteProvider,
  getAIReports,
  getBudgets,
  getRolesOverview,
  listProviders,
  updateAISettings,
  updateBudget,
  updateProvider,
  updateProviderRoles,
  type AIProvider,
  type AISettingsUpdateRequest,
  type BudgetUpdateRequest,
  type ProviderCreateRequest,
  type ProviderUpdateRequest,
} from "@/api/ai-provider";
import { aiSettingsKeys } from "./keys";

/** 调用记录一页多少条 */
export const USAGE_REPORT_PAGE_SIZE = 10;

// ============================================================
// 读
// ============================================================

/**
 * Provider 列表（连同各厂商的可选模型与全局设置，接口一次返回）。
 *
 * 页面上 Provider 列表、角色分配、预算里的「备用 Provider」都用它：同一个键，只请求一次。
 */
export function aiProvidersQuery(canViewAISettings: boolean) {
  return queryOptions({
    queryKey: aiSettingsKeys.providers(),
    queryFn: async ({ signal }) => (await listProviders(signal)).data,
    enabled: canViewAISettings,
  });
}

/** 各角色当前分配到的 Provider */
export function aiRolesOverviewQuery(canViewAISettings: boolean) {
  return queryOptions({
    queryKey: aiSettingsKeys.roles(),
    queryFn: async ({ signal }) => (await getRolesOverview(signal)).data.data ?? [],
    enabled: canViewAISettings,
  });
}

/** 各角色的预算与已用量 */
export function aiBudgetsQuery(canViewAISettings: boolean) {
  return queryOptions({
    queryKey: aiSettingsKeys.budgets(),
    queryFn: async ({ signal }) => (await getBudgets(signal)).data.data ?? [],
    enabled: canViewAISettings,
  });
}

/**
 * AI 调用记录，按「加载更多」一页一页往后取。role 为空串 = 全部角色。
 *
 * 下一页从已加载的条数开始；已加载的条数达到总数就没有下一页。
 */
export function aiUsageReportsQuery(role: string, canViewAISettings: boolean) {
  return infiniteQueryOptions({
    queryKey: aiSettingsKeys.usageReports(role),
    queryFn: async ({ pageParam, signal }) => {
      const page = (await getAIReports({ role: role || undefined, limit: USAGE_REPORT_PAGE_SIZE, offset: pageParam }, signal)).data;
      return { reports: page.data ?? [], total: page.total ?? 0 };
    },
    initialPageParam: 0,
    getNextPageParam: (lastPage, loadedPages) => {
      const loadedCount = loadedPages.reduce((count, page) => count + page.reports.length, 0);
      // 最后一页是空的也停下：避免总数与实际条数对不上时一直「还有更多」
      return lastPage.reports.length > 0 && loadedCount < lastPage.total ? loadedCount : undefined;
    },
    enabled: canViewAISettings,
  });
}

// ============================================================
// 写
// ============================================================

/**
 * Provider 或角色分配变了：Provider 列表、角色总览、「AI 对话是否可用」都跟着变。
 * 返回的 Promise 在重取完成后结束 —— 保存按钮转圈停下时，界面上已经是新数据。
 */
function invalidateProviderAssignments(client: QueryClient): Promise<unknown> {
  return Promise.all([
    client.invalidateQueries({ queryKey: aiSettingsKeys.providers() }),
    client.invalidateQueries({ queryKey: aiSettingsKeys.roles() }),
    client.invalidateQueries({ queryKey: aiSettingsKeys.chatReadiness() }),
  ]);
}

/** 新增 Provider */
export function createAIProviderMutation() {
  return mutationOptions({
    mutationFn: (newProvider: ProviderCreateRequest) => createProvider(newProvider),
    onSuccess: (_created, _newProvider, _onMutateResult, context) => invalidateProviderAssignments(context.client),
  });
}

/** 修改 Provider。不带 apiKey = 保持已存的 key 不变 */
export function updateAIProviderMutation() {
  return mutationOptions({
    mutationFn: ({ providerId, changes }: { providerId: number; changes: ProviderUpdateRequest }) =>
      updateProvider(providerId, changes),
    onSuccess: (_updated, _change, _onMutateResult, context) => invalidateProviderAssignments(context.client),
  });
}

/** 删除 Provider */
export function deleteAIProviderMutation() {
  return mutationOptions({
    mutationFn: (providerId: number) => deleteProvider(providerId),
    onSuccess: (_deleted, _providerId, _onMutateResult, context) => invalidateProviderAssignments(context.client),
  });
}

/** 保存全局设置（工具调用超时） */
export function updateAIGlobalSettingsMutation() {
  return mutationOptions({
    mutationFn: (settings: AISettingsUpdateRequest) => updateAISettings(settings),
    onSuccess: (_saved, _settings, _onMutateResult, context) => invalidateProviderAssignments(context.client),
  });
}

export interface AIRoleAssignment {
  role: string;
  /** 分配给哪个 Provider；null = 不分配 */
  providerId: number | null;
  /** 当前的 Provider 列表（用来找出现在持有该角色的 Provider，以及各 Provider 已有的角色） */
  providers: AIProvider[];
}

/**
 * 把一个角色分配给某个 Provider（或取消分配）。
 *
 * 一个角色同一时间只能由一个 Provider 持有，所以分两步：先从现在的持有者身上拿掉，再加到新的 Provider 上。
 * 无论成败都重取：第一步成功、第二步失败时，界面要如实显示「该角色现在没人持有」。
 */
export function assignAIRoleMutation() {
  return mutationOptions({
    mutationFn: async ({ role, providerId, providers }: AIRoleAssignment) => {
      const currentHolder = providers.find((provider) => provider.roles?.includes(role));
      if (currentHolder) {
        await updateProviderRoles(
          currentHolder.id,
          currentHolder.roles.filter((heldRole) => heldRole !== role),
        );
      }
      const newHolder = providerId === null ? undefined : providers.find((provider) => provider.id === providerId);
      if (newHolder) {
        await updateProviderRoles(newHolder.id, [...(newHolder.roles ?? []).filter((heldRole) => heldRole !== role), role]);
      }
    },
    onSettled: (_result, _error, _assignment, _onMutateResult, context) => invalidateProviderAssignments(context.client),
  });
}

/** 保存某个角色的预算。成功后等预算重取完才算结束 */
export function updateAIBudgetMutation() {
  return mutationOptions({
    mutationFn: ({ role, changes }: { role: string; changes: BudgetUpdateRequest }) => updateBudget(role, changes),
    onSuccess: (_saved, _change, _onMutateResult, context) =>
      context.client.invalidateQueries({ queryKey: aiSettingsKeys.budgets() }),
  });
}
