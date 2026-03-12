"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";

/**
 * 部署管理已合并到 GitHub & GitOps 页面。
 * 此页面仅用于重定向旧路由。
 */
export default function DeployPage() {
  const router = useRouter();
  useEffect(() => {
    router.replace("/settings/github");
  }, [router]);
  return null;
}
