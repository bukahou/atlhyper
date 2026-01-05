"use client";

import { I18nProvider } from "@/i18n/context";
import { ThemeProvider } from "@/theme/context";

export function Providers({ children }: { children: React.ReactNode }) {
  return (
    <ThemeProvider>
      <I18nProvider>{children}</I18nProvider>
    </ThemeProvider>
  );
}
