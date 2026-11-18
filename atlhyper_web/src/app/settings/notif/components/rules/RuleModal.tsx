"use client";

import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { Modal, toast } from "@/components/common";
import { notifyFailureText } from "@/lib/notify-failure";
import { NOTIFY_POLICY_RANGES } from "@/lib/notify-policy-input";
import {
  RULE_TIMING_FIELDS,
  ruleBodyFromDraft,
  ruleDraftFrom,
  ruleDraftProblems,
  type NotifyRuleDraft,
} from "@/lib/notify-rule-input";
import { saveNotifyRuleMutation } from "@/queries/notify-routing";
import type { CommonTranslations, NotificationsTranslations } from "@/types/i18n";
import {
  NOTIFY_ALERT_SOURCES,
  NOTIFY_SEVERITIES,
  type NotifyReceiver,
  type NotifyRule,
  type NotifySeverity,
} from "@/types/notify";
import { ChipMultiSelect } from "../shared/ChipMultiSelect";
import {
  FIELD_ERROR_CLASS,
  HINT_CLASS,
  INPUT_CLASS,
  LABEL_CLASS,
  PRIMARY_BUTTON_CLASS,
  SECONDARY_BUTTON_CLASS,
  SELECT_CLASS,
} from "../shared/form-styles";
import { Switch } from "../shared/Switch";
import { RuleClusterField } from "./RuleClusterField";

interface RuleModalProps {
  /** 要编辑的规则；null = 新建（排在最后） */
  editing: NotifyRule | null;
  receivers: NotifyReceiver[];
  /** 可选的集群（已接入的；演示模式下是演示集群） */
  clusterIds: string[];
  onClose: () => void;
  t: NotificationsTranslations;
  commonT: CommonTranslations;
}

/**
 * 新建 / 编辑路由规则的表单。由外层按目标整个重新挂载（key），表单初值直接取自 editing。
 */
export function RuleModal({ editing, receivers, clusterIds, onClose, t, commonT }: RuleModalProps) {
  const rulesT = t.rules;
  const [draft, setDraft] = useState<NotifyRuleDraft>(() => ruleDraftFrom(editing ?? undefined));
  // 没点过提交就不报错：一打开就满屏红字没有意义
  const [submitAttempted, setSubmitAttempted] = useState(false);
  const [serverError, setServerError] = useState("");

  // 成功后工厂会让列表重新取数，这里只管提示与关弹窗
  const saveRule = useMutation(saveNotifyRuleMutation());
  const updateDraft = (changes: Partial<NotifyRuleDraft>) => setDraft((current) => ({ ...current, ...changes }));

  const problems = ruleDraftProblems(draft);
  const showProblems = submitAttempted && problems.length > 0;

  const handleSubmit = (event: React.FormEvent) => {
    event.preventDefault();
    setSubmitAttempted(true);
    setServerError("");
    if (problems.length > 0) return;

    saveRule.mutate(
      { ruleId: editing?.id, rule: ruleBodyFromDraft(draft) },
      {
        onSuccess: () => {
          toast.success(editing ? rulesT.updatedToast : rulesT.createdToast);
          onClose();
        },
        onError: (error) => setServerError(notifyFailureText(error, t.errors)),
      },
    );
  };

  return (
    <Modal isOpen onClose={onClose} title={editing ? rulesT.editTitle : rulesT.createTitle} size="md">
      <form onSubmit={handleSubmit} className="p-6 space-y-5" noValidate>
        <div className="flex items-end gap-4">
          <div className="flex-1">
            <label htmlFor="notify-rule-name" className={LABEL_CLASS}>
              {rulesT.name} *
            </label>
            <input
              id="notify-rule-name"
              type="text"
              value={draft.name}
              onChange={(event) => updateDraft({ name: event.target.value })}
              autoComplete="off"
              className={INPUT_CLASS}
            />
          </div>
          <div className="flex items-center gap-2 pb-2">
            <span className="text-sm text-muted">{rulesT.enabled}</span>
            <Switch
              checked={draft.enabled}
              onChange={(enabled) => updateDraft({ enabled })}
              label={draft.enabled ? t.enabledLabel : t.disabledLabel}
            />
          </div>
        </div>
        {showProblems && problems.includes("nameRequired") && (
          <p className={FIELD_ERROR_CLASS}>{rulesT.problems.nameRequired}</p>
        )}

        <RuleClusterField
          knownClusterIds={clusterIds}
          selected={draft.clusterIds}
          onChange={(selected) => updateDraft({ clusterIds: selected })}
          t={rulesT}
        />

        <div>
          <span className={LABEL_CLASS}>{rulesT.sources}</span>
          <ChipMultiSelect
            options={NOTIFY_ALERT_SOURCES.map((source) => ({ value: source, label: t.source[source] }))}
            selected={draft.sources}
            onChange={(sources) => updateDraft({ sources })}
            emptyText=""
          />
          <p className={HINT_CLASS}>{rulesT.sourcesHint}</p>
        </div>

        <div>
          <label htmlFor="notify-rule-min-severity" className={LABEL_CLASS}>
            {rulesT.minSeverity}
          </label>
          <select
            id="notify-rule-min-severity"
            value={draft.minSeverity}
            onChange={(event) => updateDraft({ minSeverity: event.target.value as NotifySeverity })}
            className={SELECT_CLASS}
          >
            {NOTIFY_SEVERITIES.map((severity) => (
              <option key={severity} value={severity}>
                {t.severity[severity]}
              </option>
            ))}
          </select>
        </div>

        <div>
          <span className={LABEL_CLASS}>{rulesT.receivers}</span>
          <ChipMultiSelect
            options={receivers.map((receiver) => ({
              value: receiver.id,
              label: receiver.enabled ? receiver.name : `${receiver.name}（${t.disabledLabel}）`,
            }))}
            selected={draft.receiverIds}
            onChange={(receiverIds) => updateDraft({ receiverIds })}
            emptyText={t.policy.noReceiversYet}
          />
          {draft.receiverIds.length === 0 && (
            <p className="text-xs text-amber-600 dark:text-amber-400 mt-1">{rulesT.noReceiversHint}</p>
          )}
        </div>

        <div>
          <span className={LABEL_CLASS}>{rulesT.timing}</span>
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
            {RULE_TIMING_FIELDS.map((field) => {
              const range = NOTIFY_POLICY_RANGES[field];
              return (
                <div key={field}>
                  <label htmlFor={`notify-rule-${field}`} className="block text-xs text-muted mb-1">
                    {rulesT[field]}（{t.seconds}）
                  </label>
                  <input
                    id={`notify-rule-${field}`}
                    type="text"
                    inputMode="numeric"
                    value={draft[field]}
                    onChange={(event) => updateDraft({ [field]: event.target.value })}
                    placeholder={rulesT.timingUseGlobal}
                    autoComplete="off"
                    className={INPUT_CLASS}
                  />
                  {showProblems && problems.includes(field) && (
                    <p className={FIELD_ERROR_CLASS}>
                      {t.rangeError.replace("{min}", String(range.min)).replace("{max}", String(range.max))}
                    </p>
                  )}
                </div>
              );
            })}
          </div>
          <p className={HINT_CLASS}>{rulesT.timingHint}</p>
        </div>

        {serverError && (
          <div className="p-3 bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 rounded-lg text-sm" role="alert">
            {serverError}
          </div>
        )}

        <div className="flex justify-end gap-3 pt-2">
          <button type="button" onClick={onClose} className={SECONDARY_BUTTON_CLASS}>
            {commonT.cancel}
          </button>
          <button type="submit" disabled={saveRule.isPending} className={PRIMARY_BUTTON_CLASS}>
            {saveRule.isPending ? t.saving : commonT.save}
          </button>
        </div>
      </form>
    </Modal>
  );
}
