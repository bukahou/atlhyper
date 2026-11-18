"use client";

import { WifiOff } from "lucide-react";
import { useI18n } from "@/i18n/context";

/** 没有任何集群接入时的占位 */
export function NoClusterState() {
  const { t } = useI18n();
  return (
    <div className="flex flex-col items-center justify-center py-16 text-center">
      <WifiOff className="w-12 h-12 mb-4 text-muted" />
      <p className="text-default font-medium mb-2">{t.common.noCluster}</p>
      <p className="text-sm text-muted">{t.common.noClusterDesc}</p>
    </div>
  );
}
