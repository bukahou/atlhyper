"use client";

import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { toast } from "@/components/common";
import { notifyFailureText } from "@/lib/notify-failure";
import {
  NOTIFY_POLICY_NUMBER_FIELDS,
  NOTIFY_POLICY_RANGES,
  policyNumberDraft,
  policyNumberProblems,
  policyWithNumbers,
  type NotifyPolicyNumberDraft,
  type NotifyPolicyNumberField,
} from "@/lib/notify-policy-input";
import { saveNotifyPolicyMutation } from "@/queries/notify";
import type { NotificationsTranslations } from "@/types/i18n";
import { NOTIFY_SEVERITIES, type NotifyPolicy, type NotifyReceiver, type NotifySeverity } from "@/types/notify";
import { ChipMultiSelect } from "../shared/ChipMultiSelect";
import { HINT_CLASS, LABEL_CLASS, PRIMARY_BUTTON_CLASS, SELECT_CLASS } from "../shared/form-styles";
import { PolicyNumberField } from "./PolicyNumberField";
import { PolicyToggleRow } from "./PolicyToggleRow";

interface PolicyFormProps {
  /** 已保存的策略：表单的初值。外层按它整个重新挂载（key） */
  policy: NotifyPolicy;
  /** 默认接收人的选项 */
  receivers: NotifyReceiver[];
  /** 演示数据：各项照常显示但不可改、不可保存 */
  readOnly: boolean;
  t: NotificationsTranslations;
  saveLabel: string;
}

const SECTION_CLASS = "bg-card rounded-xl border border-[var(--border-color)] p-4 sm:p-6 space-y-5";
const SECTION_TITLE_CLASS = "text-sm font-semibold text-default";

/** 反复阈值的单位是「次」，没有单位文案；其余三项是秒 */
const FIELDS_IN_SECONDS: NotifyPolicyNumberField[] = ["groupWaitSeconds", "groupIntervalSeconds", "repeatIntervalSeconds"];

/** 通知策略表单：对应 GET / PUT /api/v2/notify/policy 的全部字段 */
export function PolicyForm({ policy, receivers, readOnly, t, saveLabel }: PolicyFormProps) {
  const policyT = t.policy;
  // 开关、下拉、接收人直接改这份草稿；数值项以文本另存（输到一半、清空都要能表示）
  const [draft, setDraft] = useState<NotifyPolicy>({ ...policy, defaultReceiverIds: policy.defaultReceiverIds ?? [] });
  const [numbers, setNumbers] = useState<NotifyPolicyNumberDraft>(() => policyNumberDraft(policy));
  // 没点过保存就不报错
  const [submitAttempted, setSubmitAttempted] = useState(false);
  const [serverError, setServerError] = useState("");

  // 成功后工厂把接口带回的策略换进缓存，这里只管提示
  const savePolicy = useMutation(saveNotifyPolicyMutation());
  const updateDraft = (changes: Partial<NotifyPolicy>) => setDraft((current) => ({ ...current, ...changes }));

  const problems = policyNumberProblems(numbers);
  const numberError = (field: NotifyPolicyNumberField) => {
    if (!submitAttempted || !problems.includes(field)) return "";
    const range = NOTIFY_POLICY_RANGES[field];
    return t.rangeError.replace("{min}", String(range.min)).replace("{max}", String(range.max));
  };

  const handleSubmit = (event: React.FormEvent) => {
    event.preventDefault();
    setSubmitAttempted(true);
    setServerError("");
    if (problems.length > 0) return;

    savePolicy.mutate(policyWithNumbers(draft, numbers), {
      onSuccess: () => toast.success(policyT.savedToast),
      onError: (error) => setServerError(notifyFailureText(error, t.errors)),
    });
  };

  const numberField = (field: NotifyPolicyNumberField) => (
    <PolicyNumberField
      key={field}
      id={`notify-policy-${field}`}
      label={policyT[field]}
      hint={policyT[`${field}Hint`]}
      value={numbers[field]}
      onChange={(value) => setNumbers((current) => ({ ...current, [field]: value }))}
      range={NOTIFY_POLICY_RANGES[field]}
      unit={FIELDS_IN_SECONDS.includes(field) ? t.seconds : ""}
      error={numberError(field)}
      disabled={readOnly}
    />
  );

  return (
    <form onSubmit={handleSubmit} className="space-y-4" noValidate>
      <section className={SECTION_CLASS}>
        <h3 className={SECTION_TITLE_CLASS}>{policyT.incidentSection}</h3>
        <PolicyToggleRow
          label={policyT.incidentAlertsEnabled}
          hint={policyT.incidentAlertsEnabledHint}
          checked={draft.incidentAlertsEnabled}
          onChange={(incidentAlertsEnabled) => updateDraft({ incidentAlertsEnabled })}
          disabled={readOnly}
        />
        <PolicyToggleRow
          label={policyT.notifyResolved}
          hint={policyT.notifyResolvedHint}
          checked={draft.notifyResolved}
          onChange={(notifyResolved) => updateDraft({ notifyResolved })}
          disabled={readOnly}
        />
        <div>
          <label htmlFor="notify-policy-min-severity" className={LABEL_CLASS}>
            {policyT.minSeverity}
          </label>
          <select
            id="notify-policy-min-severity"
            value={draft.minSeverity}
            onChange={(event) => updateDraft({ minSeverity: event.target.value as NotifySeverity })}
            disabled={readOnly}
            className={`${SELECT_CLASS} max-w-[10rem]`}
          >
            {NOTIFY_SEVERITIES.map((severity) => (
              <option key={severity} value={severity}>
                {t.severity[severity]}
              </option>
            ))}
          </select>
          <p className={HINT_CLASS}>{policyT.minSeverityHint}</p>
        </div>
        {numberField("warningFlapThreshold")}
        <div>
          <span className={LABEL_CLASS}>{policyT.defaultReceivers}</span>
          <ChipMultiSelect
            options={receivers.map((receiver) => ({
              value: receiver.id,
              label: receiver.enabled ? receiver.name : `${receiver.name}（${t.disabledLabel}）`,
            }))}
            selected={draft.defaultReceiverIds}
            onChange={(defaultReceiverIds) => updateDraft({ defaultReceiverIds })}
            disabled={readOnly}
            emptyText={policyT.noReceiversYet}
          />
          <p className={HINT_CLASS}>{policyT.defaultReceiversHint}</p>
        </div>
      </section>

      <section className={SECTION_CLASS}>
        <h3 className={SECTION_TITLE_CLASS}>{policyT.timingSection}</h3>
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-5">
          {NOTIFY_POLICY_NUMBER_FIELDS.filter((field) => FIELDS_IN_SECONDS.includes(field)).map(numberField)}
        </div>
      </section>

      <section className={SECTION_CLASS}>
        <h3 className={SECTION_TITLE_CLASS}>{policyT.legacySection}</h3>
        <PolicyToggleRow
          label={policyT.k8sEventAlertsEnabled}
          hint={policyT.k8sEventAlertsEnabledHint}
          checked={draft.k8sEventAlertsEnabled}
          onChange={(k8sEventAlertsEnabled) => updateDraft({ k8sEventAlertsEnabled })}
          disabled={readOnly}
        />
      </section>

      {serverError && (
        <div className="p-3 bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 rounded-lg text-sm" role="alert">
          {serverError}
        </div>
      )}

      <div className="flex justify-end">
        {/* 演示模式下写操作不可用 */}
        <button type="submit" disabled={readOnly || savePolicy.isPending} className={PRIMARY_BUTTON_CLASS}>
          {savePolicy.isPending ? t.saving : saveLabel}
        </button>
      </div>
    </form>
  );
}
