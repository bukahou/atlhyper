"use client";

import { useState } from "react";
import Link from "next/link";
import { ArrowRight, Boxes, FileText, Lock, ScrollText, Sparkles } from "lucide-react";
import { useI18n } from "@/i18n/context";
import { useCan } from "@/hooks/useCan";
import { useRequireAuth } from "@/hooks/useRequireAuth";
import { useEntityDetail } from "@/components/common/entity-detail";
import { PodLogsViewer } from "@/components/pod";
import type { IncidentSubject } from "@/lib/entity-key";
import type { Permission } from "@/types/auth";
import type { IncidentMember } from "@/types/incident-view";
import { buildNextSteps, type IncidentNextStep } from "./incident-next-steps";
import { fillTemplate } from "./incident-view-format";

const BUTTON = "inline-flex items-center gap-1.5 text-sm px-3 py-1.5 rounded-lg border transition-colors";
const NORMAL = `${BUTTON} border-[var(--border-color)] text-default hover:bg-[var(--hover-bg)]`;
const PRIMARY = `${BUTTON} border-blue-500/30 bg-blue-500/10 text-blue-600 dark:text-blue-400 font-medium hover:bg-blue-500/20`;
const LOCKED = `${BUTTON} border-[var(--border-color)] text-muted opacity-70 hover:opacity-100`;

/** 需要权限的两步：容器日志、AI 分析 */
const STEP_PERMISSIONS: Partial<Record<IncidentNextStep["kind"], Permission>> = {
  logs: "pod.logs.read",
  ai: "ai.analyze",
};

interface IncidentNextStepsProps {
  subject: IncidentSubject;
  /** 主体的对象已删除：「查看 …」按钮不可用，旁边写「已删除」 */
  subjectGone?: boolean;
  members: IncidentMember[];
  /** 跳到弹窗底部的 AI 分析区 */
  onOpenAIAnalysis: () => void;
}

/** 下一步：去主体详情、Pod 列表、相关 K8s 事件、容器日志、AI 分析。无权限的显示锁定，点了按权限提示登录或联系管理员 */
export function IncidentNextSteps({ subject, subjectGone, members, onOpenAIAnalysis }: IncidentNextStepsProps) {
  const { t } = useI18n();
  const texts = t.aiops.detail;
  const can = useCan();
  const requireAuth = useRequireAuth();
  const { openEntityDetail } = useEntityDetail();
  const [logsPod, setLogsPod] = useState<{ namespace: string; podName: string } | null>(null);
  const typeLabels: Record<string, string> = t.aiops.subjectTypeLabel;
  const steps = buildNextSteps(subject, members);

  const labelOf = (step: IncidentNextStep): string => {
    switch (step.kind) {
      case "subject":
        return fillTemplate(texts.viewSubject, { type: typeLabels[step.target.type] ?? step.target.type });
      case "pods":
        return fillTemplate(texts.viewPods, { count: step.count });
      case "nodePods":
        return texts.viewNodePods;
      case "events":
        return texts.relatedEvents;
      case "logs":
        return texts.containerLogs;
      case "ai":
        return texts.aiAnalysis;
    }
  };

  const renderStep = (step: IncidentNextStep) => {
    const label = labelOf(step);
    const permission = STEP_PERMISSIONS[step.kind];
    if (permission && !can(permission)) {
      // 锁定：仍然可点，点了由 requireAuth 提示登录或「权限不足」
      return (
        <button key={step.kind} type="button" className={LOCKED} onClick={() => requireAuth(() => {}, permission)}>
          <Lock className="w-3.5 h-3.5" />
          {label}
          <span className="text-[11px] text-muted">{t.permission.lockedTitle}</span>
        </button>
      );
    }
    switch (step.kind) {
      case "subject":
        if (subjectGone) {
          return (
            <button key={step.kind} type="button" className={LOCKED} disabled>
              {label}
              <span className="text-[11px] text-muted">{texts.subjectGone}</span>
            </button>
          );
        }
        return (
          <button key={step.kind} type="button" className={PRIMARY} onClick={() => openEntityDetail(step.target)}>
            {label}
            <ArrowRight className="w-3.5 h-3.5" />
          </button>
        );
      case "pods":
      case "nodePods":
      case "events":
        return (
          <Link key={step.kind} href={step.href} className={NORMAL}>
            {step.kind === "events" ? <ScrollText className="w-3.5 h-3.5" /> : <Boxes className="w-3.5 h-3.5" />}
            {label}
          </Link>
        );
      case "logs":
        return (
          <button
            key={step.kind}
            type="button"
            className={NORMAL}
            onClick={() => setLogsPod({ namespace: step.namespace, podName: step.podName })}
            title={step.podName}
          >
            <FileText className="w-3.5 h-3.5" />
            {label}
          </button>
        );
      case "ai":
        return (
          <button key={step.kind} type="button" className={NORMAL} onClick={onOpenAIAnalysis}>
            <Sparkles className="w-3.5 h-3.5" />
            {label}
          </button>
        );
    }
  };

  return (
    <div>
      <h4 className="text-xs font-semibold text-muted tracking-wider mb-2">{texts.nextStepsTitle}</h4>
      <div className="flex flex-wrap gap-2">{steps.map(renderStep)}</div>

      {/* 容器日志：风险最高、且还在的那个 Pod；不指定容器时由后端取默认容器 */}
      {logsPod && (
        <PodLogsViewer
          isOpen
          onClose={() => setLogsPod(null)}
          namespace={logsPod.namespace}
          podName={logsPod.podName}
          containerName=""
        />
      )}
    </div>
  );
}
