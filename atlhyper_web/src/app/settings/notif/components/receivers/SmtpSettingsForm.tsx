"use client";

import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { toast } from "@/components/common";
import { notifyFailureText } from "@/lib/notify-failure";
import { sealCredential } from "@/lib/notify-receiver-input";
import { saveNotifySMTPMutation } from "@/queries/notify-smtp";
import type { CommonTranslations, NotificationsTranslations } from "@/types/i18n";
import type { NotifySMTP } from "@/types/notify";
import { EmailFormFields } from "../EmailFormFields";
import { smtpFormFromSaved, smtpFormProblems, smtpSettingsFromDraft, type SmtpFormDraft } from "../email-form";
import { FIELD_ERROR_CLASS, PRIMARY_BUTTON_CLASS, SECONDARY_BUTTON_CLASS } from "../shared/form-styles";
import { getSmtpPresets } from "../smtp-presets";

interface SmtpSettingsFormProps {
  /** 已保存的发件设置：表单的初值 */
  smtp: NotifySMTP;
  /** 取消或保存成功后回到查看状态 */
  onDone: () => void;
  t: NotificationsTranslations;
  commonT: CommonTranslations;
}

/**
 * 编辑全局发件设置。
 *
 * 密码只写不读：输入框初始为空，已配置时留空 = 保持原值；
 * 明文只在发请求的那一刻经 sealCredential 取出，不进查询缓存与 mutation 缓存。
 */
export function SmtpSettingsForm({ smtp, onDone, t, commonT }: SmtpSettingsFormProps) {
  const smtpT = t.smtp;
  const smtpPresets = getSmtpPresets(smtpT.presets);
  const [draft, setDraft] = useState<SmtpFormDraft>(() =>
    smtpFormFromSaved(
      smtp,
      smtpPresets.map((preset) => preset.value),
    ),
  );
  const [showPassword, setShowPassword] = useState(false);
  // 没点过保存就不报错
  const [submitAttempted, setSubmitAttempted] = useState(false);
  const [serverError, setServerError] = useState("");

  // 成功后工厂把接口带回的设置换进缓存，这里只管提示与回到查看状态
  const saveSMTP = useMutation(saveNotifySMTPMutation());

  const problems = smtpFormProblems(draft, smtp.passwordConfigured);

  const handleSubmit = (event: React.FormEvent) => {
    event.preventDefault();
    setSubmitAttempted(true);
    setServerError("");
    if (problems.length > 0) return;

    saveSMTP.mutate(
      { settings: smtpSettingsFromDraft(draft), takePassword: sealCredential(draft.password) },
      {
        onSuccess: () => {
          toast.success(smtpT.savedToast);
          onDone();
        },
        onError: (error) => setServerError(notifyFailureText(error, t.errors)),
      },
    );
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4" noValidate>
      <EmailFormFields
        draft={draft}
        onChange={(changes) => setDraft((current) => ({ ...current, ...changes }))}
        smtpPresets={smtpPresets}
        passwordConfigured={smtp.passwordConfigured}
        showPassword={showPassword}
        onShowPasswordToggle={() => setShowPassword((shown) => !shown)}
        disabled={saveSMTP.isPending}
        labels={smtpT}
        enabledLabel={t.enabledLabel}
        disabledLabel={t.disabledLabel}
      />

      {submitAttempted && problems.length > 0 && (
        <ul className="space-y-0.5" role="alert">
          {problems.map((problem) => (
            <li key={problem} className={FIELD_ERROR_CLASS}>
              {smtpT.problems[problem]}
            </li>
          ))}
        </ul>
      )}

      {serverError && (
        <div className="p-3 bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 rounded-lg text-sm" role="alert">
          {serverError}
        </div>
      )}

      <div className="flex justify-end gap-3">
        <button type="button" onClick={onDone} disabled={saveSMTP.isPending} className={SECONDARY_BUTTON_CLASS}>
          {commonT.cancel}
        </button>
        <button type="submit" disabled={saveSMTP.isPending} className={PRIMARY_BUTTON_CLASS}>
          {saveSMTP.isPending ? t.saving : commonT.save}
        </button>
      </div>
    </form>
  );
}
