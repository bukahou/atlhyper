"use client";

import { ArrowDown, ArrowUp, Pencil, Trash2 } from "lucide-react";
import { DataTable, StatusBadge, type TableColumn } from "@/components/common";
import { RULE_TIMING_FIELDS } from "@/lib/notify-rule-input";
import type { NotificationsTranslations } from "@/types/i18n";
import type { NotifyReceiver, NotifyRule } from "@/types/notify";
import { ICON_BUTTON_CLASS, NOWRAP_CLASS } from "../shared/form-styles";

interface RulesTableProps {
  /** 已按 position 从小到大排好 */
  rules: NotifyRule[];
  /** 用来把规则里的接收人 id 显示成名称 */
  receivers: NotifyReceiver[];
  /** 演示数据：操作按钮照常显示但不可用 */
  readOnly: boolean;
  /** 调序请求还没返回：期间不让再点上移 / 下移 */
  reordering: boolean;
  onMove: (rule: NotifyRule, direction: "up" | "down") => void;
  onEdit: (rule: NotifyRule) => void;
  onDelete: (rule: NotifyRule) => void;
  t: NotificationsTranslations;
  editLabel: string;
  deleteLabel: string;
  actionLabel: string;
}

const CHIP_CLASS = "inline-flex px-2 py-0.5 rounded-full text-xs bg-[var(--background)] border border-[var(--border-color)] text-default";

/** 路由规则列表：自上而下就是匹配顺序 */
export function RulesTable({
  rules,
  receivers,
  readOnly,
  reordering,
  onMove,
  onEdit,
  onDelete,
  t,
  editLabel,
  deleteLabel,
  actionLabel,
}: RulesTableProps) {
  const rulesT = t.rules;
  const receiverName = (receiverId: number) =>
    receivers.find((receiver) => receiver.id === receiverId)?.name ??
    rulesT.unknownReceiver.replace("{id}", String(receiverId));

  const columns: TableColumn<NotifyRule>[] = [
    {
      key: "position",
      header: rulesT.position,
      render: (rule) => <span className="text-sm text-muted tabular-nums">{rules.indexOf(rule) + 1}</span>,
    },
    {
      key: "name",
      header: rulesT.name,
      mobileTitle: true,
      render: (rule) => <span className="text-sm font-medium text-default">{rule.name}</span>,
    },
    {
      key: "enabled",
      header: rulesT.enabled,
      render: (rule) => (
        <span className={NOWRAP_CLASS}>
          <StatusBadge
            status={rule.enabled ? t.enabledLabel : t.disabledLabel}
            type={rule.enabled ? "success" : "default"}
          />
        </span>
      ),
    },
    {
      key: "conditions",
      header: rulesT.conditions,
      render: (rule) => (
        <div className="flex flex-wrap gap-1.5 max-w-sm">
          {(rule.clusterIds ?? []).length === 0 ? (
            <span className={CHIP_CLASS}>{rulesT.allClusters}</span>
          ) : (
            rule.clusterIds.map((clusterId) => (
              <span key={clusterId} className={`${CHIP_CLASS} font-mono`}>
                {clusterId}
              </span>
            ))
          )}
          {(rule.sources ?? []).length === 0 ? (
            <span className={CHIP_CLASS}>{rulesT.allSources}</span>
          ) : (
            rule.sources.map((source) => (
              <span key={source} className={CHIP_CLASS}>
                {t.source[source] ?? source}
              </span>
            ))
          )}
          <span className={CHIP_CLASS}>
            {rulesT.minSeverityAtLeast.replace("{severity}", t.severity[rule.minSeverity] ?? rule.minSeverity)}
          </span>
        </div>
      ),
    },
    {
      key: "receivers",
      header: rulesT.receivers,
      render: (rule) =>
        (rule.receiverIds ?? []).length === 0 ? (
          <span className="text-sm text-amber-600 dark:text-amber-400">{rulesT.noReceivers}</span>
        ) : (
          <div className="flex flex-wrap gap-1.5 max-w-xs">
            {rule.receiverIds.map((receiverId) => (
              <span key={receiverId} className={CHIP_CLASS}>
                {receiverName(receiverId)}
              </span>
            ))}
          </div>
        ),
    },
    {
      key: "timing",
      header: rulesT.timing,
      mobileVisible: false,
      render: (rule) => {
        const overrides = RULE_TIMING_FIELDS.filter((field) => rule[field] !== null);
        if (overrides.length === 0) return <span className="text-sm text-muted">{rulesT.timingUseGlobal}</span>;
        return (
          <div className="space-y-0.5">
            {overrides.map((field) => (
              <div key={field} className={`text-xs text-default ${NOWRAP_CLASS}`}>
                {rulesT[field]}: <span className="tabular-nums">{rule[field]}</span> {t.seconds}
              </div>
            ))}
          </div>
        );
      },
    },
    {
      key: "actions",
      header: actionLabel,
      render: (rule) => {
        const index = rules.indexOf(rule);
        return (
          <div className="flex items-center gap-1">
            <button
              type="button"
              onClick={() => onMove(rule, "up")}
              disabled={readOnly || reordering || index === 0}
              className={ICON_BUTTON_CLASS}
              title={rulesT.moveUp}
              aria-label={rulesT.moveUp}
            >
              <ArrowUp className="w-4 h-4" />
            </button>
            <button
              type="button"
              onClick={() => onMove(rule, "down")}
              disabled={readOnly || reordering || index === rules.length - 1}
              className={ICON_BUTTON_CLASS}
              title={rulesT.moveDown}
              aria-label={rulesT.moveDown}
            >
              <ArrowDown className="w-4 h-4" />
            </button>
            <button
              type="button"
              onClick={() => onEdit(rule)}
              disabled={readOnly}
              className={ICON_BUTTON_CLASS}
              title={editLabel}
              aria-label={editLabel}
            >
              <Pencil className="w-4 h-4" />
            </button>
            <button
              type="button"
              onClick={() => onDelete(rule)}
              disabled={readOnly}
              className={`${ICON_BUTTON_CLASS} hover:text-red-500`}
              title={deleteLabel}
              aria-label={deleteLabel}
            >
              <Trash2 className="w-4 h-4" />
            </button>
          </div>
        );
      },
    },
  ];

  return (
    <DataTable
      columns={columns}
      data={rules}
      loading={false}
      error=""
      keyExtractor={(rule) => String(rule.id)}
      // 规则要整份看（顺序就是语义），不分页
      pageSize={Math.max(rules.length, 1)}
      showPagination={false}
    />
  );
}
