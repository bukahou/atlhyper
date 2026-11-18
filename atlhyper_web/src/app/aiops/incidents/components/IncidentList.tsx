"use client";

import { Fragment, useState } from "react";
import { useI18n } from "@/i18n/context";
import type { Incident } from "@/api/aiops";
import { IncidentImpactedRows } from "./IncidentImpactedRows";
import { GRID_COLUMNS, IncidentListRow } from "./IncidentListRow";
import { incidentListRows } from "./incident-grouping-format";

interface IncidentListProps {
  incidents: Incident[];
  onSelect: (id: string) => void;
  /**
   * 列表是按组折叠取回的（grouped=true）：源头的 impacted 可展开。
   * 分组由后端完成，这里只负责展开 / 收起
   */
  grouped: boolean;
}

export function IncidentList({ incidents, onSelect, grouped }: IncidentListProps) {
  const { t } = useI18n();
  // 展开了哪些源头：默认全部折叠（受影响事件折叠在源头下面）
  const [expandedIds, setExpandedIds] = useState<ReadonlySet<string>>(() => new Set());
  const rows = incidentListRows(incidents);

  const toggleExpanded = (id: string) =>
    setExpandedIds((previous) => {
      const next = new Set(previous);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-x-auto">
      {/* 列多：窄屏时整张表横向滚动，不把列挤扁 */}
      <div className="min-w-[860px]">
        {/* 表头 */}
        <div className={`${GRID_COLUMNS} px-5 py-2 text-[10px] text-muted uppercase tracking-wider border-b border-[var(--border-color)]/50`}>
          <span>{t.aiops.incidentId}</span>
          <span title={t.aiops.incidentStateTooltip} className="cursor-help border-b border-dotted border-current">{t.aiops.incidentState}</span>
          <span title={t.aiops.severityTooltip} className="cursor-help border-b border-dotted border-current">{t.aiops.severity}</span>
          <span>{t.aiops.subject}</span>
          <span>{t.aiops.peakRisk}</span>
          <span>{t.aiops.startedAt}</span>
          <span>{t.aiops.duration}</span>
          <span>{t.aiops.recurrence}</span>
          <span>{t.aiops.escalatedAt}</span>
        </div>

        {/* 行 */}
        {rows.length === 0 ? (
          <div className="px-5 py-8 text-center text-sm text-muted">{t.aiops.noData}</div>
        ) : (
          rows.map((row) => {
            const { incident } = row;
            const expandable = grouped && incident.impacted.length > 0;
            const expanded = expandable && expandedIds.has(incident.id);
            return (
              <Fragment key={incident.id}>
                <IncidentListRow
                  row={row}
                  onSelect={onSelect}
                  expandable={expandable}
                  expanded={expanded}
                  onToggleExpanded={toggleExpanded}
                />
                {expanded && <IncidentImpactedRows impacted={incident.impacted} onSelect={onSelect} />}
              </Fragment>
            );
          })
        )}
      </div>
    </div>
  );
}
