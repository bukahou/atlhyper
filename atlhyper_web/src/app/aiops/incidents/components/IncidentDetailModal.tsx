"use client";

import { useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { X, Loader2 } from "lucide-react";
import { useI18n } from "@/i18n/context";
import { AIAnalysisSection } from "@/components/aiops/AIAnalysisSection";
import { incidentSubject } from "@/lib/entity-key";
import { incidentDetailQuery } from "@/queries/aiops";
import { IncidentEvidence } from "./IncidentEvidence";
import { IncidentHistory } from "./IncidentHistory";
import { IncidentImpactScope } from "./IncidentImpactScope";
import { IncidentMembers } from "./IncidentMembers";
import { IncidentNextSteps } from "./IncidentNextSteps";
import { IncidentRootLink } from "./IncidentRootLink";
import { IncidentVerdict } from "./IncidentVerdict";
import { SubjectTitle } from "./SubjectTitle";
import { TimelineView } from "./TimelineView";
import { withDetailGroupingDefaults } from "./incident-grouping-format";
import { fillTemplate, withDetailViewDefaults } from "./incident-view-format";

interface IncidentDetailModalProps {
  incidentId: string | null;
  open: boolean;
  onClose: () => void;
  /** 切到另一个事件的详情（源头 / 受影响事件之间跳转） */
  onSelectIncident: (incidentId: string) => void;
}

/**
 * 事件详情（docs/design/active/incident-notification-design.md「功能四」）。从上到下：
 * 结论（受影响事件下方一行「源头：…经…」）→ 关键证据 →（源头事件）影响范围 → 下一步 → 成员
 * → 时间线（默认折叠）→ 历史与 AI 分析。拓扑归并见 docs/design/active/aiops-topology-grouping-design.md「D5」。
 */
export function IncidentDetailModal({ incidentId, open, onClose, onSelectIncident }: IncidentDetailModalProps) {
  const { t } = useI18n();
  // 弹窗关着时不取：incidentId 传空串，查询不启用
  const detailQuery = useQuery(incidentDetailQuery(open && incidentId ? incidentId : ""));
  const detail = detailQuery.data ? withDetailGroupingDefaults(withDetailViewDefaults(detailQuery.data)) : undefined;
  const loading = detailQuery.isPending;
  const aiSectionRef = useRef<HTMLDivElement>(null);
  // 没有证据时区分「改版前的事件」与「只有依赖传播」：由后端判定（evidenceStatus）
  const hasDetections = detail?.evidenceStatus === "not_recorded";

  if (!open) return null;

  const scrollToAIAnalysis = () => aiSectionRef.current?.scrollIntoView({ behavior: "smooth", block: "start" });

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* 背景遮罩 */}
      <div className="absolute inset-0 bg-black/50 backdrop-blur-sm" onClick={onClose} />

      {/* 弹窗 */}
      <div className="relative bg-card border border-[var(--border-color)] rounded-2xl shadow-2xl w-full max-w-2xl max-h-[85vh] overflow-y-auto mx-4">
        {/* 头部 */}
        <div className="sticky top-0 z-10 bg-card border-b border-[var(--border-color)] px-6 py-4 flex items-center justify-between rounded-t-2xl">
          {/* 标题是主体（类型 + 命名空间 + 名称）；详情还没回来时只有事件号 */}
          <div className="min-w-0">
            {detail && (
              <h2 className="text-base text-default min-w-0">
                <SubjectTitle incident={detail} size="lg" />
              </h2>
            )}
            <p className={detail ? "text-xs text-muted mt-0.5" : "text-base font-bold text-default"}>
              {t.aiops.incidentId} <span className="font-mono">{detail?.id ?? incidentId}</span>
              {detail && detail.clusterId !== "" && (
                <> · {fillTemplate(t.aiops.detail.clusterLabel, { cluster: detail.clusterId })}</>
              )}
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
              {/* 结论：状态、严重度、时长、一句话、影响数字 */}
              <IncidentVerdict incident={detail} hasDetections={hasDetections} />

              {/* 受影响事件：源头与依赖路径，点击切到源头 */}
              {detail.root && <IncidentRootLink root={detail.root} onOpenIncident={onSelectIncident} />}

              {/* 关键证据：按信号去重 */}
              <IncidentEvidence
                evidence={detail.evidence}
                totalMembers={detail.headline?.totalMembers ?? detail.members.length}
                hasDetections={hasDetections}
              />

              {/* 源头事件：影响范围（归到它下面的受影响事件） */}
              {detail.impacted.length > 0 && (
                <IncidentImpactScope impacted={detail.impacted} onOpenIncident={onSelectIncident} />
              )}

              {/* 下一步：去哪查 */}
              <IncidentNextSteps
                subject={incidentSubject(detail)}
                members={detail.members}
                onOpenAIAnalysis={scrollToAIAnalysis}
              />

              {/* 成员：状态、重启、风险；「风险最高」标记 */}
              <IncidentMembers members={detail.members} />

              {/* 时间线：默认折叠 */}
              <TimelineView items={detail.timelineView} />

              {/* 收尾：历史与 AI 分析 */}
              <IncidentHistory incident={detail} />
              <div
                ref={aiSectionRef}
                className="scroll-mt-24 rounded-xl border border-dashed border-[var(--border-color)] px-3 py-2.5"
              >
                <AIAnalysisSection incidentId={detail.id} incidentState={detail.state} />
              </div>
            </>
          ) : (
            <div className="py-8 text-center text-sm text-muted">{t.aiops.noData}</div>
          )}
        </div>
      </div>
    </div>
  );
}
