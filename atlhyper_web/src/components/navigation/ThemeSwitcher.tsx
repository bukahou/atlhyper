"use client";

import { Sun, Moon, Monitor } from "lucide-react";
import { useTheme } from "@/theme/context";
import { useI18n } from "@/i18n/context";
import type { Theme } from "@/types/common";

export function ThemeSwitcher() {
  const { theme, setTheme } = useTheme();
  const { t } = useI18n();

  const themes: { value: Theme; icon: typeof Sun; label: string }[] = [
    { value: "light", icon: Sun, label: t.common.themeLight },
    { value: "dark", icon: Moon, label: t.common.themeDark },
    { value: "system", icon: Monitor, label: t.common.themeSystem },
  ];

  const currentTheme = themes.find((t) => t.value === theme) || themes[2];
  const Icon = currentTheme.icon;

  return (
    <div className="relative group">
      <button
        className="p-2 rounded-lg hover-bg"
        aria-label="Switch theme"
      >
        <Icon className="w-5 h-5 text-secondary" />
      </button>
      <div className="absolute right-0 mt-2 w-32 dropdown-menu rounded-lg shadow-lg border opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all z-50">
        {themes.map((t) => {
          const ThemeIcon = t.icon;
          return (
            <button
              key={t.value}
              onClick={() => setTheme(t.value)}
              className={`w-full px-4 py-2 text-left text-sm flex items-center gap-2 hover-bg first:rounded-t-lg last:rounded-b-lg ${
                theme === t.value
                  ? "text-primary font-medium"
                  : "text-secondary"
              }`}
            >
              <ThemeIcon className="w-4 h-4" />
              {t.label}
            </button>
          );
        })}
      </div>
    </div>
  );
}
