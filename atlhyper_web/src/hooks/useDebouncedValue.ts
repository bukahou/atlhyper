/**
 * 去抖后的值
 *
 * 搜索框的输入会进查询键。每敲一个字符就换一次键，等于每个字符发一次请求，
 * 所以要等用户停下来再把值交出去。
 */
import { useEffect, useState } from "react";

/** 搜索类输入停顿多久算「输入完了」 */
export const SEARCH_DEBOUNCE_MS = 300;

export interface Debouncer<T> {
  /** 记下最新的值；距离最后一次 push 满 delayMs 毫秒后才交出去 */
  push: (value: T) => void;
  /** 丢弃还在等待的值 */
  cancel: () => void;
}

/**
 * 去抖器：连续 push 时只把最后一个值交给 settle。
 *
 * 从 hook 里抽出来是为了不依赖 React 与 DOM 就能测定时与取消。
 */
export function createDebouncer<T>(settle: (value: T) => void, delayMs: number): Debouncer<T> {
  let pendingTimer: ReturnType<typeof setTimeout> | null = null;
  const cancel = () => {
    if (pendingTimer !== null) clearTimeout(pendingTimer);
    pendingTimer = null;
  };
  return {
    push(value) {
      cancel();
      pendingTimer = setTimeout(() => {
        pendingTimer = null;
        settle(value);
      }, delayMs);
    },
    cancel,
  };
}

/**
 * 返回 value 去抖后的版本：value 连续变化时保持旧值，停下来 delayMs 毫秒后才跟上。
 *
 * @example
 * const [searchInput, setSearchInput] = useState("");
 * const search = useDebouncedValue(searchInput);   // search 进查询键，searchInput 绑输入框
 */
export function useDebouncedValue<T>(value: T, delayMs: number = SEARCH_DEBOUNCE_MS): T {
  const [debouncedValue, setDebouncedValue] = useState(value);

  useEffect(() => {
    const debouncer = createDebouncer<T>(setDebouncedValue, delayMs);
    debouncer.push(value);
    // value 再变或组件卸载时，丢掉还没交出去的旧值
    return debouncer.cancel;
  }, [value, delayMs]);

  return debouncedValue;
}
