"use client";

import { EntityLink } from "@/components/aiops/EntityLink";
import { useI18n } from "@/i18n/context";
import type { IncidentMember } from "@/types/incident-view";
import { fillTemplate, memberStatusTone, type MemberStatusTone } from "./incident-view-format";

const STATUS_TONES: Record<MemberStatusTone, string> = {
  ok: "bg-emerald-500/15 text-emerald-600 dark:text-emerald-400",
  pending: "bg-yellow-500/15 text-yellow-700 dark:text-yellow-400",
  bad: "bg-red-500/15 text-red-600 dark:text-red-400",
};

interface IncidentMembersProps {
  members: IncidentMember[];
}

/** 成员表：每个成员当前的状态、重启次数、风险；风险最高的那个标出来（不叫「根因」），已删除的成员标「已不存在」 */
export function IncidentMembers({ members }: IncidentMembersProps) {
  const { t } = useI18n();
  const texts = t.aiops.detail;
  const leadFirst = [...members.filter((member) => member.isLead), ...members.filter((member) => !member.isLead)];

  return (
    <div>
      <h4 className="text-xs font-semibold text-muted tracking-wider mb-2">
        {fillTemplate(texts.membersTitle, { count: members.length })}
      </h4>
      {leadFirst.length === 0 ? (
        <p className="text-xs text-muted">{texts.noMembers}</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-[11px] text-muted border-b border-[var(--border-color)]">
                <th className="text-left font-semibold px-2 pb-1.5">{texts.colName}</th>
                <th className="text-left font-semibold px-2 pb-1.5">{texts.colStatus}</th>
                <th className="text-right font-semibold px-2 pb-1.5">{texts.colRestarts}</th>
                <th className="text-right font-semibold px-2 pb-1.5">{texts.colRisk}</th>
              </tr>
            </thead>
            <tbody>
              {leadFirst.map((member) => (
                <tr key={member.entityKey} className="border-b border-[var(--border-color)] last:border-b-0">
                  <td className="px-2 py-1.5">
                    <span className="inline-flex items-center gap-1.5 min-w-0">
                      <EntityLink entityKey={member.entityKey} />
                      {member.isLead && (
                        <span
                          title={texts.leadBadgeHint}
                          className="flex-shrink-0 text-[10px] px-1 rounded border border-red-500/30 text-red-600 dark:text-red-400 whitespace-nowrap"
                        >
                          {texts.leadBadge}
                        </span>
                      )}
                    </span>
                  </td>
                  <td className="px-2 py-1.5">
                    {!member.present ? (
                      <span className="text-xs text-muted">{texts.memberGone}</span>
                    ) : member.status === "" ? (
                      <span className="text-xs text-muted">—</span>
                    ) : (
                      <span className="inline-flex items-center gap-1.5">
                        <span
                          className={`text-xs px-2 py-0.5 rounded-full font-medium whitespace-nowrap ${STATUS_TONES[memberStatusTone(member.status)]}`}
                        >
                          {member.status}
                        </span>
                        {/* 就绪容器数（如 0/1），节点等没有这一项时为空 */}
                        {member.ready !== "" && <span className="text-xs text-muted tabular-nums">{member.ready}</span>}
                      </span>
                    )}
                  </td>
                  <td className="px-2 py-1.5 text-right tabular-nums text-default">
                    {member.present && member.restarts !== null ? member.restarts : "—"}
                  </td>
                  <td className="px-2 py-1.5 text-right tabular-nums text-default">{Math.round(member.rFinal)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
