"use client";

import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { PageHeader } from "@/components/common";
import { Construction } from "lucide-react";

export default function AgentsPage() {
  const { t } = useI18n();

  return (
    <Layout>
      <div className="space-y-6">
        <PageHeader
          title={t.nav.agents}
          description={t.agents.pageDescription}
        />

        <div className="bg-card rounded-xl border border-[var(--border-color)] p-12">
          <div className="flex flex-col items-center justify-center text-center">
            <Construction className="w-16 h-16 text-yellow-500 mb-4" />
            <h2 className="text-xl font-semibold text-default mb-2">
              {t.placeholder.developingTitle}
            </h2>
            <p className="text-muted max-w-md">
              {t.placeholder.developingMessage}
            </p>
          </div>
        </div>
      </div>
    </Layout>
  );
}
