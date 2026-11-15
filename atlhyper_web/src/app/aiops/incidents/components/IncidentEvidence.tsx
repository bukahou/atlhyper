"use client";

import { useI18n } from "@/i18n/context";
import type { IncidentEvidence as IncidentEvidenceItem } from "@/types/incident-view";
import { formatShortTime } from "./incident-format";
import { evidenceTone, fillTemplate, signalReasonText, type EvidenceTone } from "./incident-view-format";

const TONE_BARS: Record<EvidenceTone, string> = {
  red: "bg-red-500",
  amber: "bg-amber-500",
  gray: "bg-gray-300 dark:bg-gray-600",
};

interface IncidentEvidenceProps {
  evidence: IncidentEvidenceItem[];
  /** 成员总数：卡片上写「几个成员中招」的分母 */
  totalMembers: number;
  /** 时间线里有合并的检测记录：证据为空时据此判断是「改版前的事件」还是「确实没有本地信号」 */
  hasDetections: boolean;
}

/** 关键证据：每种信号一张卡，写中招成员数、首次 / 最近、次数；最近一次的原文作为次要文字 */
export function IncidentEvidence({ evidence, totalMembers, hasDetections }: IncidentEvidenceProps) {
  const { t } = useI18n();
  const texts = t.aiops.detail;

  return (
    <div>
      <h4 className="text-xs font-semibold text-muted tracking-wider mb-2">{texts.evidenceTitle}</h4>
      {evidence.length === 0 ? (
        <p className="text-xs text-muted rounded-xl border border-dashed border-[var(--border-color)] px-3 py-2.5">
          {hasDetections ? texts.evidenceLegacy : texts.evidenceNone}
        </p>
      ) : (
        <div className="space-y-2">
          {evidence.map((item) => {
            const reason = signalReasonText(item.signal, item.reason);
            return (
              <div
                key={item.signal}
                className="flex gap-3 rounded-xl border border-[var(--border-color)] px-3 py-2.5"
              >
                <span className={`w-1 flex-shrink-0 self-stretch rounded-full ${TONE_BARS[evidenceTone(item.signal)]}`} />
                <div className="flex-1 min-w-0">
                  <div className="flex flex-wrap items-baseline gap-x-2 gap-y-0.5">
                    <span className="text-sm font-semibold text-default">
                      {texts.signalName[item.signal] ?? texts.signalName.other}
                    </span>
                    {reason !== "" && (
                      <code className="text-xs font-mono px-1 rounded bg-[var(--hover-bg)] text-default">{reason}</code>
                    )}
                    <span className="text-xs font-medium text-default tabular-nums">
                      {fillTemplate(texts.evidenceAffected, { affected: item.affectedMembers, total: totalMembers })}
                    </span>
                  </div>
                  {item.detail !== "" && <p className="text-xs text-muted mt-0.5 break-words">{item.detail}</p>}
                </div>
                <div className="flex-shrink-0 text-right text-xs text-muted tabular-nums whitespace-nowrap">
                  <div>{fillTemplate(texts.evidenceFirstSeen, { time: formatShortTime(item.firstSeen) })}</div>
                  <div>{fillTemplate(texts.evidenceLastSeen, { time: formatShortTime(item.lastSeen) })}</div>
                  <div>{fillTemplate(texts.evidenceHits, { count: item.hits })}</div>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
