"use client";

import { EntityLink } from "@/components/aiops/EntityLink";
import { useI18n } from "@/i18n/context";
import { incidentSubject, type IncidentSubjectSource } from "@/lib/entity-key";

interface SubjectTitleProps {
  incident: IncidentSubjectSource;
  /** 详情弹窗的标题用大一号 */
  size?: "sm" | "lg";
}

/**
 * 事件的主体：类型 + 命名空间 + 名称。
 *
 * 事件以主体为单位（同一工作负载的 Pod 归到一个事件），所以主体是事件的标题；
 * 名称可以点开资源详情。
 */
export function SubjectTitle({ incident, size = "sm" }: SubjectTitleProps) {
  const { t } = useI18n();
  const subject = incidentSubject(incident);
  const typeLabels: Record<string, string> = t.aiops.subjectTypeLabel;

  return (
    <span className="inline-flex items-center gap-1.5 min-w-0 max-w-full" title={subject.key}>
      <span
        className={`flex-shrink-0 rounded font-medium bg-indigo-500/10 text-indigo-600 dark:text-indigo-400 ${
          size === "lg" ? "text-xs px-2 py-0.5" : "text-[10px] px-1.5 py-0.5"
        }`}
      >
        {typeLabels[subject.type] ?? subject.type}
      </span>
      {subject.namespace !== "" && (
        <span className={`flex-shrink-0 text-muted ${size === "lg" ? "text-sm" : "text-xs"}`}>{subject.namespace} /</span>
      )}
      <span className={`min-w-0 ${size === "lg" ? "font-semibold" : "font-medium"}`}>
        <EntityLink entityKey={subject.key} showType={false} />
      </span>
    </span>
  );
}
