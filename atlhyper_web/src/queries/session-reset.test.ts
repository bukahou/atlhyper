import { QueryClient } from "@tanstack/react-query";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useAuthStore } from "@/store/authStore";
import { resetQueriesWhenIdentityChanges } from "./session-reset";

describe("resetQueriesWhenIdentityChanges", () => {
  afterEach(() => useAuthStore.setState({ token: null }));

  it("token 变化（登录 / 登出）时重置全部查询", () => {
    const queryClient = new QueryClient();
    const reset = vi.spyOn(queryClient, "resetQueries").mockResolvedValue();
    const unsubscribe = resetQueriesWhenIdentityChanges(queryClient);

    useAuthStore.setState({ token: "token-a" });
    expect(reset).toHaveBeenCalledTimes(1);

    useAuthStore.setState({ token: null });
    expect(reset).toHaveBeenCalledTimes(2);
    unsubscribe();
  });

  it("token 没变时不重置（权限刷新、打开登录框等其他状态变化）", () => {
    const queryClient = new QueryClient();
    const reset = vi.spyOn(queryClient, "resetQueries").mockResolvedValue();
    const unsubscribe = resetQueriesWhenIdentityChanges(queryClient);

    useAuthStore.setState({ meLoaded: true, isLoginDialogOpen: true });
    expect(reset).not.toHaveBeenCalled();
    unsubscribe();
  });

  it("取消订阅后不再响应", () => {
    const queryClient = new QueryClient();
    const reset = vi.spyOn(queryClient, "resetQueries").mockResolvedValue();
    resetQueriesWhenIdentityChanges(queryClient)();

    useAuthStore.setState({ token: "token-b" });
    expect(reset).not.toHaveBeenCalled();
  });
});
