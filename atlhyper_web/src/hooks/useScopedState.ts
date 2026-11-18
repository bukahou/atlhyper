/**
 * 只在某个范围内有效的状态
 *
 * 页码、brush 选区这类状态依附于一组筛选条件：条件一变，它们就该回到默认值。
 * 用 useEffect 监听条件再 setState 会多渲染一轮，中间那一轮还会带着旧页码发一次请求；
 * 这里改成把「这个值属于哪个范围」和值存在一起，读的时候范围对不上就当它不存在。
 */
import { useCallback, useState } from "react";

export interface ScopedValue<T> {
  /** 这个值是在哪个范围下设的 */
  scope: string;
  value: T;
}

/** 保存的值属于当前范围就用它，否则用默认值 */
export function resolveScopedValue<T>(held: ScopedValue<T>, currentScope: string, fallback: T): T {
  return held.scope === currentScope ? held.value : fallback;
}

/**
 * 用法同 useState，多一个 scope：scope 变了，值自动回到 initialValue。
 *
 * @example
 * // 筛选条件变了，页码回到第一页
 * const [page, setPage] = useScopedState(JSON.stringify(filter), 1);
 */
export function useScopedState<T>(scope: string, initialValue: T): [T, (next: T) => void] {
  const [held, setHeld] = useState<ScopedValue<T>>({ scope, value: initialValue });
  const setValue = useCallback((next: T) => setHeld({ scope, value: next }), [scope]);
  return [resolveScopedValue(held, scope, initialValue), setValue];
}
