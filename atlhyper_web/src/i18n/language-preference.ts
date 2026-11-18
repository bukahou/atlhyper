/**
 * 界面语言的偏好：用户手动选过就用选的，没选过跟随浏览器语言
 */
import type { Language } from "@/types/common";
import { createLocalPreferenceStore } from "@/lib/local-preference-store";
import { defaultLanguage } from "./index";

const STORAGE_KEY = "atlhyper-language";

/**
 * @param stored 存着的手动选择（没选过为 null）
 * @param browserLanguage 浏览器语言（如 "ja-JP"）
 */
export function resolveLanguage(stored: string | null, browserLanguage: string): Language {
  if (stored === "zh" || stored === "ja") return stored;
  return browserLanguage.toLowerCase().startsWith("ja") ? "ja" : defaultLanguage;
}

function browserLanguage(): string {
  if (typeof navigator === "undefined") return "";
  return navigator.language || (navigator as unknown as { userLanguage?: string }).userLanguage || "";
}

export const languagePreferenceStore = createLocalPreferenceStore<Language>({
  storageKey: STORAGE_KEY,
  parse: (stored) => resolveLanguage(stored, browserLanguage()),
  // 服务端读不到浏览器语言，先按默认语言渲染（水合完成前页面本来也不显示）
  serverValue: defaultLanguage,
});
