import { QueryClient, type MutationOptions } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

// vi.mock 会被提升到文件顶部，替身必须用 vi.hoisted 一起提升
const api = vi.hoisted(() => ({
  getNotifySMTP: vi.fn(),
  updateNotifySMTP: vi.fn(),
}));
vi.mock("@/api/notify", () => api);

import { sealCredential } from "@/lib/notify-receiver-input";
import type { NotifySMTP, NotifySMTPSettings } from "@/types/notify";
import { notifyKeys } from "./keys";
import { notifySMTPQuery, saveNotifySMTPMutation } from "./notify-smtp";

const smtpKey = notifyKeys.smtp();
const savedSMTP: NotifySMTP = {
  configured: true,
  host: "smtp.example.com",
  port: 587,
  user: "ops@example.com",
  tls: true,
  fromAddress: "",
  passwordConfigured: true,
  updatedAt: "2026-11-15T00:00:00Z",
};
const settings: NotifySMTPSettings = {
  host: "smtp.example.com",
  port: 587,
  user: "ops@example.com",
  tls: true,
  fromAddress: "",
};
// 测试用的假密码：只要是在别处不会出现的字符串即可
const PASSWORD_PLAINTEXT = "not-a-real-smtp-password";

function runMutation<TResult, TVariables>(
  client: QueryClient,
  options: MutationOptions<TResult, Error, TVariables>,
  variables: TVariables,
) {
  return client.getMutationCache().build(client, options).execute(variables);
}

/** 把一个值里能走到的字符串全部收集出来（JSON.stringify 会跳过函数与循环引用，不够可靠） */
function collectStrings(value: unknown, seen = new Set<unknown>()): string[] {
  if (typeof value === "string") return [value];
  if (typeof value !== "object" || value === null || seen.has(value)) return [];
  seen.add(value);
  return Object.values(value).flatMap((inner) => collectStrings(inner, seen));
}

function everyCachedString(client: QueryClient): string {
  const fromMutations = client.getMutationCache().getAll().flatMap((mutation) => collectStrings(mutation.state));
  const fromQueries = client.getQueryCache().getAll().flatMap((query) => collectStrings(query.state));
  return [...fromMutations, ...fromQueries].join("\n");
}

beforeEach(() => {
  for (const call of Object.values(api)) call.mockReset();
});

describe("notifySMTPQuery", () => {
  it("没有 notify.manage 时不启用", () => {
    expect(notifySMTPQuery(false).enabled).toBe(false);
    expect(notifySMTPQuery(true).enabled).toBe(true);
  });

  it("键固定；返回解包后的设置，带取消信号", async () => {
    expect(notifySMTPQuery(true).queryKey).toEqual(["notify", "smtp"]);

    api.getNotifySMTP.mockResolvedValueOnce({ data: { message: "获取成功", data: savedSMTP } });
    expect(await new QueryClient().fetchQuery(notifySMTPQuery(true))).toEqual(savedSMTP);
    expect(api.getNotifySMTP.mock.calls[0][0]).toBeInstanceOf(AbortSignal);
  });
});

describe("saveNotifySMTPMutation", () => {
  it("密码拼进请求体；请求体只有契约里的六项；保存后的设置换进缓存", async () => {
    api.updateNotifySMTP.mockResolvedValue({ data: { message: "已保存", data: savedSMTP } });
    const client = new QueryClient();

    // 调用方误把整份已保存的设置当成 settings 传进来：只读字段不得进请求体
    const saved = await runMutation(client, saveNotifySMTPMutation(), {
      settings: { ...savedSMTP },
      takePassword: sealCredential(PASSWORD_PLAINTEXT),
    });

    expect(saved).toEqual(savedSMTP);
    expect(api.updateNotifySMTP).toHaveBeenCalledWith({ ...settings, password: PASSWORD_PLAINTEXT });
    expect(client.getQueryData(smtpKey)).toEqual(savedSMTP);
  });

  it("密码留空：请求体里是空串（保持原值）", async () => {
    api.updateNotifySMTP.mockResolvedValue({ data: { message: "已保存", data: savedSMTP } });

    await runMutation(new QueryClient(), saveNotifySMTPMutation(), {
      settings,
      takePassword: sealCredential(""),
    });

    expect(api.updateNotifySMTP).toHaveBeenCalledWith({ ...settings, password: "" });
  });

  it("密码明文不留在任何缓存里：mutation 还在缓存里时，查询缓存与 mutation 缓存里都找不到", async () => {
    api.updateNotifySMTP.mockResolvedValue({ data: { message: "已保存", data: savedSMTP } });
    const client = new QueryClient();

    await runMutation(client, saveNotifySMTPMutation(), {
      settings,
      takePassword: sealCredential(PASSWORD_PLAINTEXT),
    });

    expect(client.getMutationCache().getAll()).toHaveLength(1);
    const cached = everyCachedString(client);
    expect(cached).toContain("smtp.example.com");
    expect(cached).not.toContain(PASSWORD_PLAINTEXT);
  });

  it("保存失败（如第一次保存没填密码 400）不动缓存，错误原样抛给调用方", async () => {
    api.updateNotifySMTP.mockRejectedValue(new Error("bad request"));
    const client = new QueryClient();
    const unconfigured: NotifySMTP = { ...savedSMTP, configured: false, passwordConfigured: false, updatedAt: null };
    client.setQueryData(smtpKey, unconfigured);

    await expect(
      runMutation(client, saveNotifySMTPMutation(), {
        settings,
        takePassword: sealCredential(PASSWORD_PLAINTEXT),
      }),
    ).rejects.toThrow("bad request");

    expect(client.getQueryData(smtpKey)).toEqual(unconfigured);
    expect(everyCachedString(client)).not.toContain(PASSWORD_PLAINTEXT);
  });
});
