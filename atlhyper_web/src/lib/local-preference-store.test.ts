import { describe, expect, it, vi } from "vitest";
import { createLocalPreferenceStore } from "./local-preference-store";

type Size = "small" | "large";

function parseSize(raw: string | null): Size {
  return raw === "large" ? "large" : "small";
}

/** 只实现用到的两个方法的假 localStorage */
function fakeStorage(initial: Record<string, string> = {}) {
  const entries = new Map(Object.entries(initial));
  return {
    getItem: (key: string) => entries.get(key) ?? null,
    setItem: (key: string, value: string) => void entries.set(key, value),
  };
}

describe("createLocalPreferenceStore", () => {
  it("当前值取自存储，经 parse 解析；脏数据落回默认", () => {
    const stored = createLocalPreferenceStore({
      storageKey: "size",
      parse: parseSize,
      serverValue: "small",
      getStorage: () => fakeStorage({ size: "large" }),
    });
    expect(stored.getSnapshot()).toBe("large");

    const dirty = createLocalPreferenceStore({
      storageKey: "size",
      parse: parseSize,
      serverValue: "small",
      getStorage: () => fakeStorage({ size: "huge" }),
    });
    expect(dirty.getSnapshot()).toBe("small");
  });

  it("取不到存储（服务端）时用 serverValue", () => {
    const store = createLocalPreferenceStore({
      storageKey: "size",
      parse: parseSize,
      serverValue: "large",
      getStorage: () => null,
    });
    expect(store.getSnapshot()).toBe("large");
    expect(store.getServerSnapshot()).toBe("large");
  });

  it("set 写入存储并通知订阅者；取消订阅后不再通知", () => {
    const storage = fakeStorage();
    const store = createLocalPreferenceStore({
      storageKey: "size",
      parse: parseSize,
      serverValue: "small",
      getStorage: () => storage,
    });
    const onChange = vi.fn();
    const unsubscribe = store.subscribe(onChange);

    store.set("large");
    expect(storage.getItem("size")).toBe("large");
    expect(store.getSnapshot()).toBe("large");
    expect(onChange).toHaveBeenCalledTimes(1);

    unsubscribe();
    store.set("small");
    expect(onChange).toHaveBeenCalledTimes(1);
  });
});
