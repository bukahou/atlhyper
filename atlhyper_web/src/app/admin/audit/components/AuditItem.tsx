import { User, Clock, Activity, CheckCircle, XCircle } from "lucide-react";
import type { AuditLogItem } from "@/types/auth";
import type { AuditTranslations } from "@/types/i18n";
import { getActionLabel, getRoleLabel, getResourceLabel } from "./audit-utils";

interface AuditItemProps {
  log: AuditLogItem;
  auditT: AuditTranslations;
}

/** 单条审计记录组件 */
export function AuditItem({ log, auditT }: AuditItemProps) {
  const resultStyle = log.success
    ? "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400"
    : "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400";

  const actionLabel = getActionLabel(log.action, log.resource, auditT);
  const roleLabel = getRoleLabel(log.role, auditT);
  const resourceLabel = getResourceLabel(log.resource, auditT);

  return (
    <div className="flex gap-4">
      <div className="flex flex-col items-center">
        <div className={`w-3 h-3 rounded-full flex-shrink-0 ${log.success ? "bg-green-500" : "bg-red-500"}`} />
        <div className="w-px flex-1 bg-[var(--border-color)]" />
      </div>
      <div className="flex-1 pb-6">
        <div className="flex items-center gap-4 mb-2 flex-wrap">
          <div className="flex items-center gap-2">
            <User className="w-4 h-4 text-gray-400" />
            <span className="font-medium text-default">{log.username}</span>
            <span className="text-xs text-muted px-1.5 py-0.5 bg-[var(--background)] rounded">
              {roleLabel}
            </span>
          </div>
          <span className={`inline-flex items-center gap-1 px-2 py-0.5 text-xs font-medium rounded-full ${resultStyle}`}>
            {log.success ? (
              <CheckCircle className="w-3 h-3" />
            ) : (
              <XCircle className="w-3 h-3" />
            )}
            {log.success ? auditT.successOnly : auditT.failedOnly}
          </span>
        </div>

        <div className="flex items-center gap-2 mb-2">
          <Activity className="w-4 h-4 text-primary" />
          <span className="text-default font-medium">{actionLabel}</span>
          <span className="text-xs px-1.5 py-0.5 rounded bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400">
            {resourceLabel}
          </span>
          {log.status > 0 && (
            <span className={`text-xs px-1.5 py-0.5 rounded font-mono ${
              log.status >= 400 ? "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400" : "bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300"
            }`}>
              {log.status}
            </span>
          )}
        </div>

        <div className="flex items-center gap-4 text-sm text-gray-500 flex-wrap">
          <div className="flex items-center gap-1">
            <Clock className="w-3 h-3" />
            {new Date(log.timestamp).toLocaleString()}
          </div>
          <span>IP: {log.ip}</span>
        </div>
      </div>
    </div>
  );
}
