"use client";

import { createContext, useContext, useState, useEffect, type ReactNode } from "react";
import type { Language } from "@/types/common";
import type { Translations, I18nContextType } from "@/types/i18n";
import { getTranslations, defaultLanguage } from "./index";

const I18nContext = createContext<I18nContextType | undefined>(undefined);

const STORAGE_KEY = "atlhyper-language";

export function I18nProvider({ children }: { children: ReactNode }) {
  const [language, setLanguageState] = useState<Language>(defaultLanguage);
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    // 优先使用用户手动选择的语言
    const stored = localStorage.getItem(STORAGE_KEY) as Language | null;
    if (stored && (stored === "zh" || stored === "ja")) {
      setLanguageState(stored);
    } else {
      // 未手动选择时，跟随浏览器语言
      const browserLang = navigator.language || (navigator as unknown as { userLanguage?: string }).userLanguage || "";
      // 检测是否为日语（ja, ja-JP 等）
      if (browserLang.toLowerCase().startsWith("ja")) {
        setLanguageState("ja");
      }
      // 其他情况默认中文（defaultLanguage）
    }
    setMounted(true);
  }, []);

  const setLanguage = (newLanguage: Language) => {
    setLanguageState(newLanguage);
    localStorage.setItem(STORAGE_KEY, newLanguage);
  };

  const t = getTranslations(language);

  if (!mounted) {
    return null;
  }

  return (
    <I18nContext.Provider value={{ language, setLanguage, t }}>
      {children}
    </I18nContext.Provider>
  );
}

export function useI18n(): I18nContextType {
  const context = useContext(I18nContext);
  if (!context) {
    throw new Error("useI18n must be used within I18nProvider");
  }
  return context;
}
