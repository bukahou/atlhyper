/**
 * 主题偏好：亮 / 暗 / 跟随系统
 */
import type { Theme } from "@/types/common";
import { createLocalPreferenceStore } from "@/lib/local-preference-store";

const STORAGE_KEY = "atlhyper-theme";
const DARK_SCHEME_QUERY = "(prefers-color-scheme: dark)";

export type ResolvedTheme = "light" | "dark";

/** 存着的值不合法（没选过、脏数据）时跟随系统 */
export function parseStoredTheme(stored: string | null): Theme {
  return stored === "light" || stored === "dark" || stored === "system" ? stored : "system";
}

/** 实际生效的主题：选了「跟随系统」就看系统当前是不是暗色 */
export function resolveTheme(theme: Theme, systemPrefersDark: boolean): ResolvedTheme {
  if (theme === "system") return systemPrefersDark ? "dark" : "light";
  return theme;
}

export const themePreferenceStore = createLocalPreferenceStore<Theme>({
  storageKey: STORAGE_KEY,
  parse: parseStoredTheme,
  serverValue: "system",
});

/** 订阅系统亮暗色的变化（形状由 useSyncExternalStore 规定） */
export function subscribeToSystemScheme(onChange: () => void): () => void {
  const mediaQuery = window.matchMedia(DARK_SCHEME_QUERY);
  mediaQuery.addEventListener("change", onChange);
  return () => mediaQuery.removeEventListener("change", onChange);
}

export function systemPrefersDark(): boolean {
  return window.matchMedia(DARK_SCHEME_QUERY).matches;
}
