"use client";

import { useState } from "react";
import { QueryClientProvider } from "@tanstack/react-query";
import { I18nProvider } from "@/i18n/context";
import { ThemeProvider } from "@/theme/context";
import { EntityDetailProvider } from "@/components/common/entity-detail";
import { SessionLoader } from "@/components/session/SessionLoader";
import { createQueryClient } from "@/queries/client";

export function Providers({ children }: { children: React.ReactNode }) {
  // 每个浏览器会话一份；放在 state 里，重新渲染时不会重建
  const [queryClient] = useState(createQueryClient);

  return (
    <QueryClientProvider client={queryClient}>
      <SessionLoader />
      <ThemeProvider>
        <I18nProvider>
          <EntityDetailProvider>{children}</EntityDetailProvider>
        </I18nProvider>
      </ThemeProvider>
    </QueryClientProvider>
  );
}
