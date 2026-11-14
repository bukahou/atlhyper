"use client";

import { createContext, useCallback, useContext, useEffect, useMemo, useSyncExternalStore, type ReactNode } from "react";
import type { Theme } from "@/types/common";
import { useHydrated } from "@/hooks/useHydrated";
import {
  resolveTheme,
  subscribeToSystemScheme,
  systemPrefersDark,
  themePreferenceStore,
  type ResolvedTheme,
} from "./theme-preference";

interface ThemeContextType {
  theme: Theme;
  setTheme: (theme: Theme) => void;
  resolvedTheme: ResolvedTheme;
}

const ThemeContext = createContext<ThemeContextType | undefined>(undefined);

/** 服务端不知道用户系统的亮暗色，按亮色算 */
const serverPrefersDark = () => false;

export function ThemeProvider({ children }: { children: ReactNode }) {
  // 主题偏好（localStorage）与系统亮暗色（媒体查询）都是 React 之外的状态：
  // 直接订阅，生效的主题由两者算出来，不再各存一份 state 再用 effect 同步
  const theme = useSyncExternalStore(
    themePreferenceStore.subscribe,
    themePreferenceStore.getSnapshot,
    themePreferenceStore.getServerSnapshot,
  );
  const prefersDark = useSyncExternalStore(subscribeToSystemScheme, systemPrefersDark, serverPrefersDark);
  const resolvedTheme = resolveTheme(theme, prefersDark);
  const hydrated = useHydrated();

  // 样式靠 <html> 上的 dark 类切换，这里把生效的主题同步过去
  useEffect(() => {
    document.documentElement.classList.toggle("dark", resolvedTheme === "dark");
  }, [resolvedTheme]);

  const setTheme = useCallback((newTheme: Theme) => themePreferenceStore.set(newTheme), []);
  const contextValue = useMemo(() => ({ theme, setTheme, resolvedTheme }), [theme, setTheme, resolvedTheme]);

  // 水合完成前不渲染页面：主题存在 localStorage 里，服务端读不到，先渲染会与客户端的首帧不一致
  if (!hydrated) {
    return null;
  }

  return <ThemeContext.Provider value={contextValue}>{children}</ThemeContext.Provider>;
}

export function useTheme(): ThemeContextType {
  const context = useContext(ThemeContext);
  if (!context) {
    throw new Error("useTheme must be used within ThemeProvider");
  }
  return context;
}
