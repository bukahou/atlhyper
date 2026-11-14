"use client";

import { useEffect, useRef } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { useMutation } from "@tanstack/react-query";
import { completeGitHubInstallationMutation } from "@/queries/gitops";

/** 从失败的请求里取出给人看的原因：优先后端返回的 error 字段 */
function installationErrorMessage(error: unknown): string {
  const failure = error as { response?: { data?: { error?: string } }; message?: string } | null;
  return failure?.response?.data?.error || failure?.message || String(error);
}

export default function GitHubCallbackPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const calledRef = useRef(false);
  const { mutate: completeInstallation, error: installationError } = useMutation(completeGitHubInstallationMutation());
  const error = installationError ? installationErrorMessage(installationError) : null;

  // GitHub 装完 App 后跳回这里：把 installation 交给后端登记，只做一次
  useEffect(() => {
    if (calledRef.current) return;
    calledRef.current = true;

    const installationId = searchParams.get("installation_id");
    const setupAction = searchParams.get("setup_action");

    if (!installationId) {
      router.replace("/settings/github");
      return;
    }

    completeInstallation(
      { installationId: Number(installationId), setupAction: setupAction || "install" },
      {
        onSuccess: () => router.replace("/settings/github"),
        onError: (err) => console.error("GitHub App installation callback failed:", installationErrorMessage(err), err),
      },
    );
  }, [searchParams, router, completeInstallation]);

  if (error) {
    return (
      <div style={{ display: "flex", flexDirection: "column", justifyContent: "center", alignItems: "center", height: "60vh", gap: "16px" }}>
        <p style={{ color: "#ef4444" }}>GitHub 连接失败: {error}</p>
        <button
          onClick={() => router.replace("/settings/github")}
          style={{ padding: "8px 16px", borderRadius: "8px", border: "1px solid #666", cursor: "pointer" }}
        >
          返回设置
        </button>
      </div>
    );
  }

  return (
    <div style={{ display: "flex", justifyContent: "center", alignItems: "center", height: "60vh" }}>
      <p>GitHub 连接中...</p>
    </div>
  );
}
