/**
 * 是否已经在浏览器里完成水合
 *
 * 语言、主题、登录态都存在 localStorage 里，服务端渲染时读不到。
 * 用它把依赖这些值的界面推迟到水合之后再渲染，服务端与客户端的首帧才一致。
 */
import { useSyncExternalStore } from "react";

/** 这个值没有「变化」可订阅：服务端与水合那一帧为假，之后恒为真 */
const subscribeToNothing = () => () => {};

export function useHydrated(): boolean {
  return useSyncExternalStore(
    subscribeToNothing,
    () => true,
    () => false,
  );
}
