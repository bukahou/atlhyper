"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import * as githubAPI from "@/api/github";

export default function GitHubCallbackPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const calledRef = useRef(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (calledRef.current) return;
    calledRef.current = true;

    const installationId = searchParams.get("installation_id");
    const setupAction = searchParams.get("setup_action");

    if (!installationId) {
      router.replace("/settings/github");
      return;
    }

    githubAPI
      .callback(Number(installationId), setupAction || "install")
      .then(() => {
        router.replace("/settings/github");
      })
      .catch((err) => {
        const msg = err?.response?.data?.error || err?.message || String(err);
        console.error("GitHub App installation callback failed:", msg, err);
        setError(msg);
      });
  }, [searchParams, router]);

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
