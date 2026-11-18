import { QueryClient, type MutationOptions } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

// vi.mock 会被提升到文件顶部，替身必须用 vi.hoisted 一起提升
const api = vi.hoisted(() => ({
  getNotifyPolicy: vi.fn(),
  updateNotifyPolicy: vi.fn(),
  getNotifyAlerts: vi.fn(),
  getNotifyDeliveries: vi.fn(),
  getNotifySilences: vi.fn(),
  createNotifySilence: vi.fn(),
  expireNotifySilence: vi.fn(),
}));
vi.mock("@/api/notify", () => api);

import type { NotifyAlertListParams } from "@/api/notify";
import type { NotifyPolicy } from "@/types/notify";
import { notifyKeys } from "./keys";
import {
  createNotifySilenceMutation,
  expireNotifySilenceMutation,
  notifyAlertsQuery,
  notifyDeliveriesQuery,
  notifyPolicyQuery,
  notifySilencesQuery,
  saveNotifyPolicyMutation,
} from "./notify";

const policy: NotifyPolicy = {
  incidentAlertsEnabled: true,
  minSeverity: "low",
  notifyResolved: true,
  warningFlapThreshold: 3,
  groupWaitSeconds: 30,
  groupIntervalSeconds: 300,
  repeatIntervalSeconds: 14400,
  k8sEventAlertsEnabled: false,
  defaultReceiverIds: [],
};
const alert = { fingerprint: "incident:inc-123", status: "firing", title: "api 探针失败" };
const delivery = { id: 1, fingerprint: "incident:inc-123", status: "success" };
const silence = { id: 1, clusterId: "prod", namespace: "shop", subject: "", state: "active" };
const silenceBody = {
  clusterId: "prod",
  namespace: "shop",
  subject: "",
  startsAt: "2026-11-15T11:00:00.000Z",
  endsAt: "2026-11-15T13:00:00.000Z",
  comment: "数据库升级",
};

function runMutation<TResult, TVariables>(
  client: QueryClient,
  options: MutationOptions<TResult, Error, TVariables>,
  variables: TVariables,
) {
  return client.getMutationCache().build(client, options).execute(variables);
}

function isInvalidated(client: QueryClient, queryKey: readonly unknown[]): boolean {
  return client.getQueryState(queryKey)?.isInvalidated ?? false;
}

beforeEach(() => {
  for (const call of Object.values(api)) call.mockReset();
});

describe("没有 notify.manage 权限时不发请求（页面此时整页显示演示数据）", () => {
  it.each([
    ["策略", notifyPolicyQuery(false), notifyPolicyQuery(true)],
    [
      "告警",
      notifyAlertsQuery({ status: "firing", limit: 20, offset: 0 }, false),
      notifyAlertsQuery({ status: "firing", limit: 20, offset: 0 }, true),
    ],
    [
      "送达记录",
      notifyDeliveriesQuery({ status: "", limit: 20, offset: 0 }, false),
      notifyDeliveriesQuery({ status: "", limit: 20, offset: 0 }, true),
    ],
    ["静默", notifySilencesQuery("active", false), notifySilencesQuery("active", true)],
  ])("%s", (_name, withoutPermission, withPermission) => {
    expect(withoutPermission.enabled).toBe(false);
    expect(withPermission.enabled).toBe(true);
  });
});

describe("notifyPolicyQuery / saveNotifyPolicyMutation", () => {
  it("键固定：策略全站一份", () => {
    expect(notifyPolicyQuery(true).queryKey).toEqual(["notify", "policy"]);
  });

  it("返回解包后的策略，带取消信号", async () => {
    api.getNotifyPolicy.mockResolvedValue({ data: { message: "获取成功", data: policy } });

    expect(await new QueryClient().fetchQuery(notifyPolicyQuery(true))).toEqual(policy);
    expect(api.getNotifyPolicy.mock.calls[0][0]).toBeInstanceOf(AbortSignal);
  });

  it("保存后把接口带回的策略换进缓存，不再请求一次", async () => {
    const saved: NotifyPolicy = { ...policy, minSeverity: "high" };
    api.updateNotifyPolicy.mockResolvedValue({ data: { message: "已保存", data: saved } });
    const client = new QueryClient();
    client.setQueryData(notifyKeys.policy(), policy);

    await runMutation(client, saveNotifyPolicyMutation(), saved);

    expect(api.updateNotifyPolicy).toHaveBeenCalledWith(saved);
    expect(client.getQueryData(notifyKeys.policy())).toEqual(saved);
    expect(api.getNotifyPolicy).not.toHaveBeenCalled();
  });

  it("保存失败（如取值超出范围 400）不动缓存，错误原样抛给调用方", async () => {
    api.updateNotifyPolicy.mockRejectedValue(new Error("bad request"));
    const client = new QueryClient();
    client.setQueryData(notifyKeys.policy(), policy);

    await expect(runMutation(client, saveNotifyPolicyMutation(), policy)).rejects.toThrow("bad request");
    expect(client.getQueryData(notifyKeys.policy())).toEqual(policy);
  });
});

describe("notifyAlertsQuery", () => {
  const firstPage = { status: "firing", limit: 20, offset: 0 } as const;

  it("筛选条件与页码都进键：换状态、翻页各用各的缓存", () => {
    expect(notifyAlertsQuery(firstPage, true).queryKey).toEqual([
      "notify",
      "alerts",
      { status: "firing", limit: 20, offset: 0 },
    ]);
    expect(notifyAlertsQuery({ ...firstPage, status: "resolved" }, true).queryKey).not.toEqual(
      notifyAlertsQuery(firstPage, true).queryKey,
    );
    expect(notifyAlertsQuery({ ...firstPage, offset: 20 }, true).queryKey).not.toEqual(
      notifyAlertsQuery(firstPage, true).queryKey,
    );
  });

  it("返回一页告警与总数，把筛选条件与取消信号交给接口", async () => {
    api.getNotifyAlerts.mockResolvedValue({ data: { message: "获取成功", data: [alert], total: 41 } });

    expect(await new QueryClient().fetchQuery(notifyAlertsQuery(firstPage, true))).toEqual({
      items: [alert],
      total: 41,
    });
    expect(api.getNotifyAlerts.mock.calls[0][0]).toEqual(firstPage);
    expect(api.getNotifyAlerts.mock.calls[0][1]).toBeInstanceOf(AbortSignal);
  });

  it("没有告警时后端可能返回 null，统一成空列表", async () => {
    api.getNotifyAlerts.mockResolvedValue({ data: { message: "获取成功", data: null, total: 0 } });

    expect(await new QueryClient().fetchQuery(notifyAlertsQuery(firstPage, true))).toEqual({ items: [], total: 0 });
  });

  it("翻页时留着上一页，换状态不留：「已结束」下不会先显示一屏「持续中」的告警", () => {
    const previousPage = { items: [alert], total: 41 };
    const previousQuery = { queryKey: notifyAlertsQuery(firstPage, true).queryKey };
    type KeepPreviousPage = (previous: unknown, query: unknown) => unknown;
    const placeholderFor = (filter: NotifyAlertListParams) =>
      (notifyAlertsQuery(filter, true).placeholderData as unknown as KeepPreviousPage)(previousPage, previousQuery);

    expect(placeholderFor({ ...firstPage, offset: 20 })).toBe(previousPage);
    expect(placeholderFor({ status: "resolved", limit: 20, offset: 0 })).toBeUndefined();
  });

  it("轮询与保鲜时间等长：告警会自己触发与结束", () => {
    const options = notifyAlertsQuery(firstPage, true);
    expect(options.refetchInterval).toBeGreaterThan(0);
    expect(options.staleTime).toBe(options.refetchInterval);
  });
});

describe("notifyDeliveriesQuery", () => {
  const firstPage = { status: "failed", limit: 20, offset: 0 } as const;

  it("状态与页码进键", () => {
    expect(notifyDeliveriesQuery(firstPage, true).queryKey).toEqual([
      "notify",
      "deliveries",
      { status: "failed", limit: 20, offset: 0 },
    ]);
  });

  it("返回一页送达记录与总数", async () => {
    api.getNotifyDeliveries.mockResolvedValue({ data: { message: "获取成功", data: [delivery], total: 7 } });

    expect(await new QueryClient().fetchQuery(notifyDeliveriesQuery(firstPage, true))).toEqual({
      items: [delivery],
      total: 7,
    });
    expect(api.getNotifyDeliveries.mock.calls[0][0]).toEqual(firstPage);
    expect(api.getNotifyDeliveries.mock.calls[0][1]).toBeInstanceOf(AbortSignal);
  });

  it("没有记录时统一成空列表", async () => {
    api.getNotifyDeliveries.mockResolvedValue({ data: { message: "获取成功", data: null, total: 0 } });

    expect(await new QueryClient().fetchQuery(notifyDeliveriesQuery(firstPage, true))).toEqual({
      items: [],
      total: 0,
    });
  });
});

describe("静默", () => {
  /** 三个状态的列表都已在缓存里：用来观察写操作成功后它们有没有被标记为失效 */
  function clientWithCachedSilences(): QueryClient {
    const client = new QueryClient();
    client.setQueryData(notifyKeys.silences("active"), [silence]);
    client.setQueryData(notifyKeys.silences("pending"), []);
    client.setQueryData(notifyKeys.silences("expired"), []);
    return client;
  }

  it("状态进键：三个状态各用各的缓存", () => {
    expect(notifySilencesQuery("active", true).queryKey).toEqual(["notify", "silences", { state: "active" }]);
    expect(notifySilencesQuery("expired", true).queryKey).toEqual(["notify", "silences", { state: "expired" }]);
  });

  it("按状态取数，返回解包后的列表；没有静默时统一成空数组", async () => {
    api.getNotifySilences.mockResolvedValueOnce({ data: { message: "获取成功", data: [silence], total: 1 } });
    expect(await new QueryClient().fetchQuery(notifySilencesQuery("active", true))).toEqual([silence]);
    expect(api.getNotifySilences.mock.calls[0][0]).toBe("active");
    expect(api.getNotifySilences.mock.calls[0][1]).toBeInstanceOf(AbortSignal);

    api.getNotifySilences.mockResolvedValueOnce({ data: { message: "获取成功", data: null, total: 0 } });
    expect(await new QueryClient().fetchQuery(notifySilencesQuery("pending", true))).toEqual([]);
  });

  it("新建后三个状态的列表都重新取数（新静默可能立即生效，也可能未开始）", async () => {
    api.createNotifySilence.mockResolvedValue({ data: { message: "已创建", data: silence } });
    const client = clientWithCachedSilences();

    expect(await runMutation(client, createNotifySilenceMutation(), silenceBody)).toEqual(silence);

    expect(api.createNotifySilence).toHaveBeenCalledWith(silenceBody);
    expect(isInvalidated(client, notifyKeys.silences("active"))).toBe(true);
    expect(isInvalidated(client, notifyKeys.silences("pending"))).toBe(true);
    expect(isInvalidated(client, notifyKeys.silences("expired"))).toBe(true);
  });

  it("新建失败（校验不过 400）不动列表缓存", async () => {
    api.createNotifySilence.mockRejectedValue(new Error("bad request"));
    const client = clientWithCachedSilences();

    await expect(runMutation(client, createNotifySilenceMutation(), silenceBody)).rejects.toThrow("bad request");
    expect(isInvalidated(client, notifyKeys.silences("active"))).toBe(false);
  });

  it("提前结束后列表重新取数：它从生效中挪到了已过期", async () => {
    api.expireNotifySilence.mockResolvedValue({ data: { message: "已结束" } });
    const client = clientWithCachedSilences();

    await runMutation(client, expireNotifySilenceMutation(), 1);

    expect(api.expireNotifySilence).toHaveBeenCalledWith(1);
    expect(isInvalidated(client, notifyKeys.silences("active"))).toBe(true);
    expect(isInvalidated(client, notifyKeys.silences("expired"))).toBe(true);
  });

  it("静默的写操作不碰别的通知数据", async () => {
    api.expireNotifySilence.mockResolvedValue({ data: { message: "已结束" } });
    const client = clientWithCachedSilences();
    client.setQueryData(notifyKeys.policy(), policy);

    await runMutation(client, expireNotifySilenceMutation(), 1);

    expect(isInvalidated(client, notifyKeys.policy())).toBe(false);
  });
});
