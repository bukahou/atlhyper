"use client";

import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import { useI18n } from "@/i18n/context";
import type { IncidentDetail, IncidentPattern } from "@/api/aiops";
import { useCurrentCluster } from "@/hooks/useCurrentCluster";
import { incidentPatternsQuery } from "@/queries/aiops";
import { formatDuration } from "./incident-format";

/**
 * 这个主体的历史模式：undefined = 还在查；null = 没有同类事件史（没有主体键、或查询失败也按没有算）。
 */
function resolveSubjectPattern(
  historyKey: string,
  patternsQuery: UseQueryResult<IncidentPattern[]>,
): IncidentPattern | null | undefined {
  if (historyKey === "" || patternsQuery.isError) return null;
  if (patternsQuery.data === undefined) return undefined;
  return patternsQuery.data.find((candidate) => candidate.entityKey === historyKey) ?? null;
}

interface IncidentHistoryProps {
  incident: IncidentDetail;
}

/** 历史（G2）：这个主体是不是老毛病。放在弹窗底部，一句话说明 */
export function IncidentHistory({ incident }: IncidentHistoryProps) {
  const { t } = useI18n();
  const { clusterId } = useCurrentCluster();
  // 按主体而不是领头实体查：领头实体是具体的 Pod，Pod 重建后换了名字，按它查历史会断；
  // 后端按实体查历史时同时匹配主体与成员实体。没有主体键时退回到领头实体
  const historyKey = incident.subject || incident.rootCause || "";
  const patternsQuery = useQuery(incidentPatternsQuery(clusterId, historyKey));
  // 还没有当前集群时查询不启用，按没有事件史算，免得一直显示「…」
  const pattern = resolveSubjectPattern(clusterId === "" ? "" : historyKey, patternsQuery);

  return (
    <div className="rounded-xl border border-dashed border-[var(--border-color)] px-3 py-2.5 text-xs text-muted">
      <div className="text-sm font-semibold text-default mb-1">{t.aiops.detail.historyTitle}</div>
      {pattern === undefined ? (
        <div className="opacity-60">…</div>
      ) : pattern === null || pattern.patternCount <= 1 ? (
        <div>{t.aiops.patternFirst}</div>
      ) : (
        <div className="flex flex-wrap gap-x-5 gap-y-1">
          <span>
            {t.aiops.patternRecur}:{" "}
            <span className="text-default font-medium tabular-nums">{pattern.patternCount}</span>
          </span>
          <span>
            {t.aiops.patternAvgDuration}:{" "}
            <span className="text-default tabular-nums">{formatDuration(pattern.avgDuration, t.aiops.minutes)}</span>
          </span>
          <span>
            {t.aiops.patternLast}: <span className="text-default">{new Date(pattern.lastOccurrence).toLocaleString()}</span>
          </span>
        </div>
      )}
    </div>
  );
}
