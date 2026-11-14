import { QueryClient, type MutationOptions } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

// vi.mock 会被提升到文件顶部，替身必须用 vi.hoisted 一起提升
const api = vi.hoisted(() => ({
  getClusterRegistry: vi.fn(),
  registerCluster: vi.fn(),
  updateClusterProfile: vi.fn(),
  issueClusterToken: vi.fn(),
  unregisterCluster: vi.fn(),
}));
vi.mock("@/api/cluster-registry", () => api);

import {
  clusterRegistryQuery,
  issueClusterTokenMutation,
  registerClusterMutation,
  unregisterClusterMutation,
  updateClusterProfileMutation,
} from "./cluster-registry";
import { adminKeys } from "./keys";

const registryKey = adminKeys.clusterRegistry();
const registeredCluster = {
  clusterId: "c1",
  name: "生产",
  description: "",
  registered: true,
  tokenIssued: false,
  tokenPrefix: "",
  tokenIssuedAt: "",
  status: "never",
  lastSeen: "",
  authenticated: false,
};
// 测试用的假令牌：只要是一段在别处不会出现的字符串即可
const TOKEN_PLAINTEXT = "atlk_test-plaintext-not-a-real-token";

/** 已有列表缓存的 QueryClient：用来观察写操作成功后列表有没有被标记为失效 */
function clientWithCachedRegistry(): QueryClient {
  const client = new QueryClient();
  client.setQueryData(registryKey, [registeredCluster]);
  return client;
}

/** 把一个值里能走到的字符串全部收集出来（JSON.stringify 会跳过函数与循环引用，不够可靠） */
function collectStrings(value: unknown, seen = new Set<unknown>()): string[] {
  if (typeof value === "string") return [value];
  if (typeof value !== "object" || value === null || seen.has(value)) return [];
  seen.add(value);
  return Object.values(value).flatMap((inner) => collectStrings(inner, seen));
}

function registryInvalidated(client: QueryClient): boolean {
  return client.getQueryState(registryKey)?.isInvalidated ?? false;
}

function runMutation<TResult, TVariables>(
  client: QueryClient,
  options: MutationOptions<TResult, Error, TVariables>,
  variables: TVariables,
) {
  return client.getMutationCache().build(client, options).execute(variables);
}

beforeEach(() => {
  for (const call of Object.values(api)) call.mockReset();
});

describe("clusterRegistryQuery", () => {
  it("没有 clusters.manage 时不启用：不发注定被拒绝的请求", () => {
    expect(clusterRegistryQuery(false).enabled).toBe(false);
    expect(clusterRegistryQuery(true).enabled).toBe(true);
  });

  it("键固定：注册表不分集群，全站一份", () => {
    expect(clusterRegistryQuery(true).queryKey).toEqual(["admin", "cluster-registry"]);
  });

  it("返回解包后的列表，并把取消信号交给接口", async () => {
    api.getClusterRegistry.mockResolvedValue({ data: { message: "获取成功", data: [registeredCluster], total: 1 } });

    expect(await new QueryClient().fetchQuery(clusterRegistryQuery(true))).toEqual([registeredCluster]);

    expect(api.getClusterRegistry.mock.calls[0][0]).toBeInstanceOf(AbortSignal);
  });

  it("没有任何集群时后端可能返回 null，统一成空数组", async () => {
    api.getClusterRegistry.mockResolvedValue({ data: { message: "获取成功", data: null, total: 0 } });

    expect(await new QueryClient().fetchQuery(clusterRegistryQuery(true))).toEqual([]);
  });

  it("轮询与保鲜时间等长：在线状态与认证状态会自己更新", () => {
    const options = clusterRegistryQuery(true);
    expect(options.refetchInterval).toBeGreaterThan(0);
    expect(options.staleTime).toBe(options.refetchInterval);
  });
});

describe("registerClusterMutation", () => {
  const registration = { clusterId: "c1", name: "生产", description: "" };

  it("登记后返回新的一行，并让列表重新取数", async () => {
    api.registerCluster.mockResolvedValue({ data: { message: "已登记", data: registeredCluster } });
    const client = clientWithCachedRegistry();

    expect(await runMutation(client, registerClusterMutation(), registration)).toEqual(registeredCluster);

    expect(api.registerCluster).toHaveBeenCalledWith(registration);
    expect(registryInvalidated(client)).toBe(true);
  });

  it("登记失败（如 409 已登记）时不动列表缓存，错误原样抛给调用方", async () => {
    api.registerCluster.mockRejectedValue(new Error("conflict"));
    const client = clientWithCachedRegistry();

    await expect(runMutation(client, registerClusterMutation(), registration)).rejects.toThrow("conflict");

    expect(registryInvalidated(client)).toBe(false);
  });
});

describe("updateClusterProfileMutation", () => {
  it("按集群 ID 修改显示名与描述，成功后让列表重新取数", async () => {
    api.updateClusterProfile.mockResolvedValue({ data: { message: "已更新", data: registeredCluster } });
    const client = clientWithCachedRegistry();

    await runMutation(client, updateClusterProfileMutation(), { clusterId: "c1", name: "生产", description: "东京" });

    expect(api.updateClusterProfile).toHaveBeenCalledWith("c1", { name: "生产", description: "东京" });
    expect(registryInvalidated(client)).toBe(true);
  });

  it("修改失败时不动列表缓存", async () => {
    api.updateClusterProfile.mockRejectedValue(new Error("not found"));
    const client = clientWithCachedRegistry();

    await expect(
      runMutation(client, updateClusterProfileMutation(), { clusterId: "c1", name: "生产", description: "" }),
    ).rejects.toThrow("not found");

    expect(registryInvalidated(client)).toBe(false);
  });
});

describe("unregisterClusterMutation", () => {
  it("删除登记后让列表重新取数", async () => {
    api.unregisterCluster.mockResolvedValue({ data: { message: "已删除登记" } });
    const client = clientWithCachedRegistry();

    await runMutation(client, unregisterClusterMutation(), "c1");

    expect(api.unregisterCluster).toHaveBeenCalledWith("c1");
    expect(registryInvalidated(client)).toBe(true);
  });

  it("删除失败时不动列表缓存", async () => {
    api.unregisterCluster.mockRejectedValue(new Error("not found"));
    const client = clientWithCachedRegistry();

    await expect(runMutation(client, unregisterClusterMutation(), "c1")).rejects.toThrow("not found");

    expect(registryInvalidated(client)).toBe(false);
  });
});

describe("issueClusterTokenMutation", () => {
  const issued = { token: TOKEN_PLAINTEXT, tokenPrefix: "atlk_tes", tokenIssuedAt: "2026-11-14T10:00:00Z" };

  it("签发后把明文交给调用方一次，并让列表重新取数（前缀与签发时间变了）", async () => {
    api.issueClusterToken.mockResolvedValue({ data: { message: "已签发", data: issued } });
    const client = clientWithCachedRegistry();
    const deliverToken = vi.fn();

    await runMutation(client, issueClusterTokenMutation(), { clusterId: "c1", deliverToken });

    expect(api.issueClusterToken).toHaveBeenCalledWith("c1");
    expect(deliverToken).toHaveBeenCalledTimes(1);
    expect(deliverToken).toHaveBeenCalledWith(TOKEN_PLAINTEXT);
    expect(registryInvalidated(client)).toBe(true);
  });

  it("mutation 的返回值里没有明文，只有前缀与签发时间", async () => {
    api.issueClusterToken.mockResolvedValue({ data: { message: "已签发", data: issued } });

    const summary = await runMutation(new QueryClient(), issueClusterTokenMutation(), {
      clusterId: "c1",
      deliverToken: () => {},
    });

    expect(summary).toEqual({ tokenPrefix: "atlk_tes", tokenIssuedAt: "2026-11-14T10:00:00Z" });
  });

  it("令牌明文不留在任何缓存里：mutation 还在缓存里时，查询缓存与 mutation 缓存里都找不到", async () => {
    api.issueClusterToken.mockResolvedValue({ data: { message: "已签发", data: issued } });
    const client = clientWithCachedRegistry();

    await runMutation(client, issueClusterTokenMutation(), { clusterId: "c1", deliverToken: () => {} });

    // 这条 mutation 此刻仍在缓存里（页面挂着时就是这样），正是要检查的状态
    const cachedMutations = client.getMutationCache().getAll();
    expect(cachedMutations).toHaveLength(1);
    const everyStringInMutationCache = cachedMutations.flatMap((mutation) => collectStrings(mutation.state));
    const everyStringInQueryCache = client
      .getQueryCache()
      .getAll()
      .flatMap((query) => collectStrings(query.state));
    expect(everyStringInMutationCache).toContain("atlk_tes");
    expect(everyStringInMutationCache.join("\n")).not.toContain(TOKEN_PLAINTEXT);
    expect(everyStringInQueryCache.join("\n")).not.toContain(TOKEN_PLAINTEXT);
  });

  it("签发失败时不交出任何东西，也不动列表缓存", async () => {
    api.issueClusterToken.mockRejectedValue(new Error("not found"));
    const client = clientWithCachedRegistry();
    const deliverToken = vi.fn();

    await expect(
      runMutation(client, issueClusterTokenMutation(), { clusterId: "c1", deliverToken }),
    ).rejects.toThrow("not found");

    expect(deliverToken).not.toHaveBeenCalled();
    expect(registryInvalidated(client)).toBe(false);
  });
});
