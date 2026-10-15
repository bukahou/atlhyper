import { describe, it, expect, vi, beforeEach } from "vitest";

const getMe = vi.fn();
vi.mock("@/api/auth", () => ({ getMe: () => getMe() }));

// node 环境没有 localStorage：给一个内存实现
const mem = new Map<string, string>();
globalThis.localStorage = {
  getItem: (k: string) => mem.get(k) ?? null,
  setItem: (k: string, v: string) => void mem.set(k, v),
  removeItem: (k: string) => void mem.delete(k),
  clear: () => mem.clear(),
  key: () => null,
  length: 0,
} as Storage;

const { useAuthStore } = await import("./authStore");
const { canNow } = await import("@/hooks/useCan");

const guestMe = { user: null, level: 0, permissions: ["cluster.read", "logs.stats.read"] };

beforeEach(() => {
  mem.clear();
  getMe.mockReset();
  useAuthStore.setState({ token: null, user: null, isAuthenticated: false, permissions: [], meLoaded: false });
});

describe("refreshMe", () => {
  it("访客拿到 0 级权限", async () => {
    getMe.mockResolvedValue({ data: { data: guestMe } });
    await useAuthStore.getState().refreshMe();
    expect(useAuthStore.getState().meLoaded).toBe(true);
    expect(canNow("cluster.read")).toBe(true);
    expect(canNow("logs.read")).toBe(false);
  });

  it("本地有 token 但后端不认（过期 / 禁用）→ 清掉本地登录态", async () => {
    mem.set("token", "stale");
    useAuthStore.setState({ token: "stale", isAuthenticated: true });
    getMe.mockResolvedValue({ data: { data: guestMe } });
    await useAuthStore.getState().refreshMe();
    const st = useAuthStore.getState();
    expect(st.isAuthenticated).toBe(false);
    expect(st.token).toBeNull();
    expect(mem.has("token")).toBe(false);
  });

  it("已登录：用后端的当前用户与权限覆盖本地（角色调整即时反映）", async () => {
    useAuthStore.setState({ token: "t", isAuthenticated: true });
    const user = { id: 7, username: "alice", displayName: "Alice", email: "", role: 1 };
    getMe.mockResolvedValue({ data: { data: { user, level: 1, permissions: ["cluster.read", "logs.read"] } } });
    await useAuthStore.getState().refreshMe();
    expect(useAuthStore.getState().user?.role).toBe(1);
    expect(canNow("logs.read")).toBe(true);
  });

  it("请求失败：失败即关，一律无权限", async () => {
    useAuthStore.setState({ permissions: ["logs.read"] });
    getMe.mockRejectedValue(new Error("network"));
    await useAuthStore.getState().refreshMe();
    expect(useAuthStore.getState().meLoaded).toBe(true);
    expect(canNow("logs.read")).toBe(false);
  });
});
