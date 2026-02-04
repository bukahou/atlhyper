"use client";

import { AlertTriangle } from "lucide-react";
import { Modal } from "@/components/common";
import type { AlertItem } from "./RecentAlertsCard";
import type { useI18n } from "@/i18n/context";

interface AlertDetailModalProps {
  alert: AlertItem | null;
  onClose: () => void;
  t: ReturnType<typeof useI18n>["t"];
}

export function AlertDetailModal({ alert, onClose, t }: AlertDetailModalProps) {
  if (!alert) return null;

  return (
    <Modal
      isOpen={!!alert}
      onClose={onClose}
      title={t.overview.alertDetails}
      size="md"
    >
      <div className="p-6 space-y-4">
        {/* 严重程度标签 */}
        <div className="flex items-center gap-3">
          <AlertTriangle className={`w-6 h-6 ${
            alert.severity === "critical" ? "text-red-500" :
            alert.severity === "warning" ? "text-yellow-500" : "text-blue-500"
          }`} />
          <span className={`inline-flex px-3 py-1 text-sm font-medium rounded-full ${
            alert.severity === "critical" ? "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400" :
            alert.severity === "warning" ? "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400" :
            "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400"
          }`}>
            {alert.severity.toUpperCase()}
          </span>
        </div>

        {/* 详情信息 */}
        <div className="grid grid-cols-2 gap-4 text-sm">
          <div>
            <span className="text-muted">{t.overview.time}</span>
            <p className="text-default font-medium">{new Date(alert.time).toLocaleString()}</p>
          </div>
          <div>
            <span className="text-muted">{t.overview.kind}</span>
            <p className="text-default font-medium">{alert.kind}</p>
          </div>
          <div>
            <span className="text-muted">{t.common.namespace}</span>
            <p className="text-default font-medium">{alert.namespace}</p>
          </div>
          <div>
            <span className="text-muted">{t.common.name}</span>
            <p className="text-default font-medium">{alert.name}</p>
          </div>
          <div>
            <span className="text-muted">{t.overview.reason}</span>
            <p className="text-default font-medium">{alert.reason}</p>
          </div>
        </div>

        {/* 完整消息 */}
        <div>
          <span className="text-sm text-muted">{t.alert.message}</span>
          <div className="mt-2 p-4 bg-[var(--background)] rounded-lg">
            <p className="text-sm text-default whitespace-pre-wrap break-words">
              {alert.message}
            </p>
          </div>
        </div>
      </div>
    </Modal>
  );
}
