"use client";

import { useRouter } from "next/navigation";
import { FileQuestion } from "lucide-react";
import { Layout } from "@/components/layout/Layout";
import { StatusPage } from "@/components/common";

export default function NotFound() {
  const router = useRouter();

  return (
    <Layout>
      <div className="-m-6 h-[calc(100vh-3.5rem)]">
        <StatusPage
          icon={FileQuestion}
          code="404"
          title="页面未找到"
          description="你访问的页面不存在或已被移除"
          action={{ label: "返回首页", onClick: () => router.push("/overview") }}
        />
      </div>
    </Layout>
  );
}
