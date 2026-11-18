/**
 * 存在 localStorage 里的界面偏好（语言、主题）
 *
 * 做成 useSyncExternalStore 能直接订阅的形态：localStorage 是 React 之外的状态，
 * 用「effect 里读出来再 setState」会多渲染一轮，首帧还会先用默认值闪一下。
 */

export interface LocalPreferenceStore<T extends string> {
  /** 订阅变化（参数与返回值的形状由 useSyncExternalStore 规定） */
  subscribe: (onChange: () => void) => () => void;
  /** 浏览器里的当前值 */
  getSnapshot: () => T;
  /** 服务端渲染与水合那一帧用的值 */
  getServerSnapshot: () => T;
  /** 写入新值并通知订阅者 */
  set: (value: T) => void;
}

interface LocalPreferenceOptions<T extends string> {
  storageKey: string;
  /** 把存着的原始值（可能没有、可能是脏数据）解析成合法的偏好 */
  parse: (raw: string | null) => T;
  /** 读不到 localStorage 时（服务端）用的值 */
  serverValue: T;
  /** 取存储对象；取不到返回 null。默认用浏览器的 localStorage，测试里换成假的 */
  getStorage?: () => Pick<Storage, "getItem" | "setItem"> | null;
}

function browserLocalStorage(): Storage | null {
  return typeof window === "undefined" ? null : window.localStorage;
}

export function createLocalPreferenceStore<T extends string>({
  storageKey,
  parse,
  serverValue,
  getStorage = browserLocalStorage,
}: LocalPreferenceOptions<T>): LocalPreferenceStore<T> {
  const listeners = new Set<() => void>();

  return {
    subscribe(onChange) {
      listeners.add(onChange);
      return () => {
        listeners.delete(onChange);
      };
    },
    // 每次都现读：值是字符串，内容不变时引用相等，不会引起多余的渲染
    getSnapshot: () => {
      const storage = getStorage();
      return storage ? parse(storage.getItem(storageKey)) : serverValue;
    },
    getServerSnapshot: () => serverValue,
    set(value) {
      getStorage()?.setItem(storageKey, value);
      for (const notify of listeners) notify();
    },
  };
}
