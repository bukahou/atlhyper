"use client";

import { createContext, useCallback, useContext, useMemo, useSyncExternalStore, type ReactNode } from "react";
import type { Language } from "@/types/common";
import type { I18nContextType } from "@/types/i18n";
import { useHydrated } from "@/hooks/useHydrated";
import { getTranslations } from "./index";
import { languagePreferenceStore } from "./language-preference";

const I18nContext = createContext<I18nContextType | undefined>(undefined);

export function I18nProvider({ children }: { children: ReactNode }) {
  // 语言偏好存在 localStorage 里（React 之外的状态），直接订阅它，不在 effect 里读出来再 setState
  const language = useSyncExternalStore(
    languagePreferenceStore.subscribe,
    languagePreferenceStore.getSnapshot,
    languagePreferenceStore.getServerSnapshot,
  );
  const hydrated = useHydrated();

  const setLanguage = useCallback((newLanguage: Language) => languagePreferenceStore.set(newLanguage), []);
  const contextValue = useMemo(
    () => ({ language, setLanguage, t: getTranslations(language) }),
    [language, setLanguage],
  );

  // 水合完成前不渲染页面：语言、登录态、当前集群都存在 localStorage 里，
  // 服务端读不到，先渲染会与客户端的首帧不一致
  if (!hydrated) {
    return null;
  }

  return <I18nContext.Provider value={contextValue}>{children}</I18nContext.Provider>;
}

export function useI18n(): I18nContextType {
  const context = useContext(I18nContext);
  if (!context) {
    throw new Error("useI18n must be used within I18nProvider");
  }
  return context;
}
