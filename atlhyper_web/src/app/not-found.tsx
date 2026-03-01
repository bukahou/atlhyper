"use client";

import { useRouter } from "next/navigation";
import { FileQuestion } from "lucide-react";
import { Layout } from "@/components/layout/Layout";
import { StatusPage } from "@/components/common";
import { useI18n } from "@/i18n/context";

export default function NotFound() {
  const router = useRouter();
  const { t } = useI18n();

  return (
    <Layout>
      <div className="-m-6 h-[calc(100vh-3.5rem)]">
        <StatusPage
          icon={FileQuestion}
          code="404"
          title={t.common.notFoundTitle}
          description={t.common.notFoundDescription}
          action={{ label: t.common.backToHome, onClick: () => router.push("/overview") }}
        />
      </div>
    </Layout>
  );
}
