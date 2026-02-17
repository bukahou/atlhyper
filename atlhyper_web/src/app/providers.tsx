"use client";

import { I18nProvider } from "@/i18n/context";
import { ThemeProvider } from "@/theme/context";
import { EntityDetailProvider } from "@/components/common/entity-detail";

export function Providers({ children }: { children: React.ReactNode }) {
  return (
    <ThemeProvider>
      <I18nProvider>
        <EntityDetailProvider>{children}</EntityDetailProvider>
      </I18nProvider>
    </ThemeProvider>
  );
}
