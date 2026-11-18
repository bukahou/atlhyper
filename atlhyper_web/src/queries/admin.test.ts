import { QueryClient, type MutationOptions } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

// vi.mock 会被提升到文件顶部，替身必须用 vi.hoisted 一起提升
const authApi = vi.hoisted(() => ({
  getPermissionTable: vi.fn(),
  getUserList: vi.fn(),
  getAuditLogs: vi.fn(),
  registerUser: vi.fn(),
  updateUserRole: vi.fn(),
  updateUserStatus: vi.fn(),
  deleteUser: vi.fn(),
}));
const commandsApi = vi.hoisted(() => ({ getCommandHistory: vi.fn() }));
vi.mock("@/api/auth", () => authApi);
vi.mock("@/api/commands", () => commandsApi);

import {
  auditLogsQuery,
  commandHistoryQuery,
  deleteUserMutation,
  registerUserMutation,
  updateUserRoleMutation,
  updateUserStatusMutation,
  userListQuery,
} from "./admin";
import { adminKeys } from "./keys";
import { REFRESH } from "./refresh";

const filter = { source: "", status: "", action: "", search: "", limit: 20, offset: 0 };

function runMutation<TResult, TVariables>(
  client: QueryClient,
  options: MutationOptions<TResult, Error, TVariables>,
  variables: TVariables,
) {
  return client.getMutationCache().build(client, options).execute(variables);
}

beforeEach(() => {
  for (const call of [...Object.values(authApi), ...Object.values(commandsApi)]) call.mockReset();
});

describe("userListQuery / auditLogsQuery", () => {
  it("没有权限时不发请求（页面此时整页显示演示数据）", () => {
    expect(userListQuery(false).enabled).toBe(false);
    expect(userListQuery(true).enabled).toBe(true);
    expect(auditLogsQuery(false).enabled).toBe(false);
    expect(auditLogsQuery(true).enabled).toBe(true);
  });

  it("返回解包后的列表，带取消信号", async () => {
    authApi.getUserList.mockResolvedValue({ data: { data: [{ id: 1 }] } });
    authApi.getAuditLogs.mockResolvedValue({ data: { data: [{ id: 7 }], total: 1 } });
    const client = new QueryClient();

    expect(await client.fetchQuery(userListQuery(true))).toEqual([{ id: 1 }]);
    expect(await client.fetchQuery(auditLogsQuery(true))).toEqual([{ id: 7 }]);

    expect(authApi.getUserList.mock.calls[0][0]).toBeInstanceOf(AbortSignal);
    expect(authApi.getAuditLogs.mock.calls[0][1]).toBeInstanceOf(AbortSignal);
  });

  it("后端返回 null 时按空列表处理", async () => {
    authApi.getUserList.mockResolvedValue({ data: { data: null } });
    expect(await new QueryClient().fetchQuery(userListQuery(true))).toEqual([]);
  });
});

describe("用户的写操作", () => {
  it.each([
    ["新增", () => registerUserMutation(), { username: "u", password: "p" }, "registerUser"],
    ["改角色", () => updateUserRoleMutation(), { userId: 2, role: 1 }, "updateUserRole"],
    ["启停", () => updateUserStatusMutation(), { userId: 2, status: 0 }, "updateUserStatus"],
    ["删除", () => deleteUserMutation(), 2, "deleteUser"],
  ] as const)("%s：调用接口，成功后让用户列表重新取数", async (_label, factory, variables, apiName) => {
    authApi[apiName].mockResolvedValue({ data: { message: "ok" } });
    const client = new QueryClient();
    client.setQueryData(adminKeys.users(), []);

    await runMutation(client, factory() as MutationOptions<unknown, Error, unknown>, variables);

    expect(authApi[apiName]).toHaveBeenCalledWith(variables);
    expect(client.getQueryState(adminKeys.users())?.isInvalidated).toBe(true);
  });

  it("失败时不动用户列表", async () => {
    authApi.deleteUser.mockRejectedValue(new Error("forbidden"));
    const client = new QueryClient();
    client.setQueryData(adminKeys.users(), []);

    await expect(runMutation(client, deleteUserMutation(), 2)).rejects.toThrow("forbidden");
    expect(client.getQueryState(adminKeys.users())?.isInvalidated).toBe(false);
  });
});

describe("commandHistoryQuery", () => {
  it("没有 commands.read 权限时不发请求；不要求选定集群（空串 = 全部集群）", () => {
    expect(commandHistoryQuery("c1", filter, false).enabled).toBe(false);
    expect(commandHistoryQuery("c1", filter, true).enabled).toBe(true);
    expect(commandHistoryQuery("", filter, true).enabled).toBe(true);
  });

  it("键里带集群、筛选条件与分页：换任何一个都是另一份数据", () => {
    const base = commandHistoryQuery("c1", filter, true).queryKey;
    expect(base).toEqual(["admin", "command-history", { clusterId: "c1", ...filter }]);
    expect(base).not.toEqual(commandHistoryQuery("c2", filter, true).queryKey);
    expect(base).not.toEqual(commandHistoryQuery("c1", { ...filter, offset: 20 }, true).queryKey);
    expect(base).not.toEqual(commandHistoryQuery("c1", { ...filter, search: "web" }, true).queryKey);
  });

  it("把当前集群、筛选条件与取消信号交给接口，返回解包后的一页", async () => {
    commandsApi.getCommandHistory.mockResolvedValue({ data: { commands: [{ commandId: "a" }], total: 41 } });

    const page = await new QueryClient().fetchQuery(commandHistoryQuery("c1", { ...filter, status: "failed" }, true));

    expect(page).toEqual({ commands: [{ commandId: "a" }], total: 41 });
    const [params, signal] = commandsApi.getCommandHistory.mock.calls[0];
    expect(params).toEqual({ clusterId: "c1", ...filter, status: "failed" });
    expect(signal).toBeInstanceOf(AbortSignal);
  });

  it("按默认档轮询，保鲜时间等长", () => {
    const options = commandHistoryQuery("c1", filter, true);
    expect(options.refetchInterval).toBe(REFRESH.standard);
    expect(options.staleTime).toBe(REFRESH.standard);
  });
});
