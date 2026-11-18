"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { PageHeader, LoadingSpinner, DemoBanner, QueryView } from "@/components/common";
import { useCan } from "@/hooks/useCan";
import { useSessionReady } from "@/hooks/useSessionReady";
import { auditLogsQuery } from "@/queries/admin";
import type { AuditLogItem } from "@/types/auth";
import { MOCK_AUDIT_LOGS } from "./components/mock-data";
import { AuditItem } from "./components/AuditItem";
import { AuditFilterBar } from "./components/AuditFilterBar";
import { filterAuditLogs, type AuditResultFilter } from "./components/audit-filter";

const NO_AUDIT_LOGS: AuditLogItem[] = [];

export default function AuditPage() {
  const { t } = useI18n();
  const auditT = t.audit;
  // 审计是纯管理功能页：权限加载完才决定显示真实数据还是演示数据，之前显示加载中
  const sessionReady = useSessionReady();
  const canReadAudit = useCan()("audit.read");
  const isDemo = sessionReady && !canReadAudit;

  const auditQuery = useQuery(auditLogsQuery(canReadAudit));
  const logs = isDemo ? MOCK_AUDIT_LOGS : (auditQuery.data ?? NO_AUDIT_LOGS);

  // 过滤器状态
  const [timeRange, setTimeRange] = useState(24); // 默认 24 小时
  const [filterUser, setFilterUser] = useState("");
  const [filterResult, setFilterResult] = useState<AuditResultFilter>("all");

  // 「最近 N 小时」的基准：真实数据用取回来的时刻，演示数据用打开页面的时刻
  const [pageOpenedAt] = useState(() => Date.now());
  const referenceTime = !isDemo && auditQuery.dataUpdatedAt > 0 ? auditQuery.dataUpdatedAt : pageOpenedAt;
  const filteredLogs = filterAuditLogs(
    logs,
    { timeRangeHours: timeRange, username: filterUser, result: filterResult },
    referenceTime,
  );

  // 统计
  const successCount = filteredLogs.filter((log) => log.success).length;
  const stats = { total: filteredLogs.length, success: successCount, failed: filteredLogs.length - successCount };

  const logList =
    filteredLogs.length === 0 ? (
      <div className="text-center py-12 text-muted">{auditT.noRecords}</div>
    ) : (
      <div className="space-y-0">
        {filteredLogs.map((log) => (
          <AuditItem key={log.id} log={log} auditT={auditT} />
        ))}
      </div>
    );

  return (
    <Layout>
      <div className="space-y-6">
        <PageHeader title={t.nav.audit} description={auditT.description} />

        {isDemo && <DemoBanner hint={t.common.demoModeHintAudit} />}

        {/* 过滤器 */}
        <AuditFilterBar
          auditT={auditT}
          commonT={t.common}
          timeRange={timeRange}
          onTimeRangeChange={setTimeRange}
          filterUser={filterUser}
          onFilterUserChange={setFilterUser}
          filterResult={filterResult}
          onFilterResultChange={setFilterResult}
          stats={stats}
        />

        {/* 审计日志列表 */}
        <div className="bg-card rounded-xl border border-[var(--border-color)] p-6">
          {!sessionReady ? (
            <div className="py-12">
              <LoadingSpinner />
            </div>
          ) : isDemo ? (
            logList
          ) : (
            <QueryView query={auditQuery}>{() => logList}</QueryView>
          )}
        </div>
      </div>
    </Layout>
  );
}
