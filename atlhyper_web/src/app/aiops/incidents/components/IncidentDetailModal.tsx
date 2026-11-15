"use client";

import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import { X, Loader2 } from "lucide-react";
import { useI18n } from "@/i18n/context";
import { AIAnalysisSection } from "@/components/aiops/AIAnalysisSection";
import { IncidentFacts } from "./IncidentFacts";
import { IncidentMembers } from "./IncidentMembers";
import { RootCauseCard } from "./RootCauseCard";
import { SubjectTitle } from "./SubjectTitle";
import { TimelineView } from "./TimelineView";
import { formatDuration } from "./incident-format";
import type { IncidentPattern } from "@/api/aiops";
import { useCurrentCluster } from "@/hooks/useCurrentCluster";
import { incidentDetailQuery, incidentPatternsQuery } from "@/queries/aiops";

interface IncidentDetailModalProps {
  incidentId: string | null;
  open: boolean;
  onClose: () => void;
}

/**
 * 这个主体的历史模式：undefined = 还在查；null = 没有同类事件史（没有主体键、或查询失败也按没有算）。
 */
function resolveSubjectPattern(
  detailLoaded: boolean,
  historyKey: string,
  patternsQuery: UseQueryResult<IncidentPattern[]>,
): IncidentPattern | null | undefined {
  if (!detailLoaded) return undefined;
  if (historyKey === "" || patternsQuery.isError) return null;
  if (patternsQuery.data === undefined) return undefined;
  return patternsQuery.data.find((candidate) => candidate.entityKey === historyKey) ?? null;
}

export function IncidentDetailModal({ incidentId, open, onClose }: IncidentDetailModalProps) {
  const { t } = useI18n();
  const { clusterId } = useCurrentCluster();
  // 弹窗关着时不取：incidentId 传空串，查询不启用
  const detailQuery = useQuery(incidentDetailQuery(open && incidentId ? incidentId : ""));
  const detail = detailQuery.data;
  const loading = detailQuery.isPending;

  // 历史模式（G2）：拿到详情后按主体查 30 天内的事件史。
  // 按主体而不是根因实体查：根因实体是具体的 Pod，Pod 重建后换了名字，按它查历史会断；
  // 后端按实体查历史时同时匹配主体与成员实体。没有主体键时退回到根因实体
  const historyKey = detail?.subject || detail?.rootCause || "";
  const patternsQuery = useQuery(incidentPatternsQuery(clusterId, historyKey));
  const pattern = resolveSubjectPattern(detail !== undefined, historyKey, patternsQuery);

  if (!open) return null;

  const rootCauseEntity = detail?.entities?.find((entity) => entity.role === "root_cause");

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* 背景遮罩 */}
      <div className="absolute inset-0 bg-black/50 backdrop-blur-sm" onClick={onClose} />

      {/* 弹窗 */}
      <div className="relative bg-card border border-[var(--border-color)] rounded-2xl shadow-2xl w-full max-w-2xl max-h-[85vh] overflow-y-auto mx-4">
        {/* 头部 */}
        <div className="sticky top-0 bg-card border-b border-[var(--border-color)] px-6 py-4 flex items-center justify-between rounded-t-2xl">
          {/* 标题是主体（类型 + 命名空间 + 名称）；详情还没回来时只有事件号 */}
          <div className="min-w-0">
            {detail && (
              <h2 className="text-base text-default min-w-0">
                <SubjectTitle incident={detail} size="lg" />
              </h2>
            )}
            <p className={detail ? "text-xs text-muted mt-0.5" : "text-base font-bold text-default"}>
              {t.aiops.incidentId} <span className="font-mono">{detail?.id ?? incidentId}</span>
            </p>
          </div>
          <button onClick={onClose} className="p-1 rounded-lg hover:bg-[var(--hover-bg)] text-muted hover:text-default transition-colors">
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* 内容 */}
        <div className="px-6 py-4 space-y-5">
          {loading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="w-6 h-6 animate-spin text-blue-500" />
            </div>
          ) : detail ? (
            <>
              {/* 概况：状态、峰值严重度、时长、复发次数、升级时间、关闭原因 */}
              <IncidentFacts incident={detail} />

              {/* 根因卡片：领头实体（成员里风险最高的那个） */}
              <RootCauseCard entity={rootCauseEntity} />

              {/* 成员实体：这个主体下涉及哪几个实体 */}
              <IncidentMembers entities={detail.entities ?? []} />

              {/* 时间线 */}
              <TimelineView timeline={detail.timeline ?? []} />

              {/* 历史模式（G2）：这个问题是不是老毛病 */}
              <div>
                <h4 className="text-xs font-semibold text-muted uppercase tracking-wider mb-2">
                  {t.aiops.patternTitle}
                </h4>
                {pattern === undefined ? (
                  <div className="text-xs text-muted opacity-60">…</div>
                ) : pattern === null || pattern.patternCount <= 1 ? (
                  <div className="text-xs text-muted">{t.aiops.patternFirst}</div>
                ) : (
                  <div className="flex flex-wrap gap-x-5 gap-y-1 text-xs">
                    <span className="text-muted">
                      {t.aiops.patternRecur}:{" "}
                      <span className="text-default font-medium tabular-nums">{pattern.patternCount}</span>
                    </span>
                    <span className="text-muted">
                      {t.aiops.patternAvgDuration}:{" "}
                      <span className="text-default tabular-nums">
                        {formatDuration(pattern.avgDuration, t.aiops.minutes)}
                      </span>
                    </span>
                    <span className="text-muted">
                      {t.aiops.patternLast}:{" "}
                      <span className="text-default">{new Date(pattern.lastOccurrence).toLocaleString()}</span>
                    </span>
                  </div>
                )}
              </div>

              {/* AI 分析报告 */}
              <AIAnalysisSection incidentId={detail.id} incidentState={detail.state} />
            </>
          ) : (
            <div className="py-8 text-center text-sm text-muted">{t.aiops.noData}</div>
          )}
        </div>
      </div>
    </div>
  );
}
