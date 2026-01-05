"use client";

import { Languages } from "lucide-react";
import { useI18n } from "@/i18n/context";
import type { Language } from "@/types/common";

const languages: { code: Language; label: string }[] = [
  { code: "zh", label: "中文" },
  { code: "ja", label: "日本語" },
];

export function LanguageSwitcher() {
  const { language, setLanguage } = useI18n();

  return (
    <div className="relative group">
      <button
        className="p-2 rounded-lg hover-bg"
        aria-label="Switch language"
      >
        <Languages className="w-5 h-5 text-secondary" />
      </button>
      <div className="absolute right-0 mt-2 w-32 dropdown-menu rounded-lg shadow-lg border opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all z-50">
        {languages.map((lang) => (
          <button
            key={lang.code}
            onClick={() => setLanguage(lang.code)}
            className={`w-full px-4 py-2 text-left text-sm hover-bg first:rounded-t-lg last:rounded-b-lg ${
              language === lang.code
                ? "text-primary font-medium"
                : "text-secondary"
            }`}
          >
            {lang.label}
          </button>
        ))}
      </div>
    </div>
  );
}
