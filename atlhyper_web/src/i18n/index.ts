import type { Language } from "@/types/common";
import type { Translations } from "@/types/i18n";
import { zh } from "./locales/zh";
import { ja } from "./locales/ja";

export const defaultLanguage: Language = "zh";

const translations: Record<Language, Translations> = {
  zh,
  ja,
};

export function getTranslations(lang: Language): Translations {
  return translations[lang] || translations[defaultLanguage];
}

export { zh, ja };
