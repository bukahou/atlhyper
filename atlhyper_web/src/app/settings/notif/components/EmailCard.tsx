"use client";

import { useState } from "react";
import { Mail, Loader2, AlertCircle } from "lucide-react";
import { useI18n } from "@/i18n/context";
import { getSmtpPresets } from "./smtp-presets";
import { EmailFormFields } from "./EmailFormFields";
import { emailFormFromSaved, emailFormHasChanges, type EmailFormDraft } from "./email-form";
import type { EmailConfig, EmailUpdateData } from "@/api/notify";

interface EmailCardProps {
  config: EmailConfig;
  enabled: boolean;
  effectiveEnabled: boolean;
  validationErrors: string[];
  readOnly: boolean;
  onSave: (data: EmailUpdateData) => Promise<void>;
  onTest: () => Promise<{ success: boolean; message: string }>;
}

export function EmailCard({
  config,
  enabled,
  effectiveEnabled,
  validationErrors,
  readOnly,
  onSave,
  onTest,
}: EmailCardProps) {
  const { t } = useI18n();
  const nt = t.notifications;
  const SMTP_PRESETS = getSmtpPresets(nt.smtpPresets);

  // 已保存的配置的指纹：内容变了才算「外部状态变了」。
  // 不看对象引用 —— 上层每次渲染都可能传来一个内容相同的新对象，那不该冲掉正在编辑的内容
  const savedFingerprint = JSON.stringify({ enabled, config });
  const draftFromSaved = () =>
    emailFormFromSaved(
      config,
      enabled,
      SMTP_PRESETS.map((preset) => preset.value),
    );

  // 表单状态
  const [draft, setDraft] = useState<EmailFormDraft>(draftFromSaved);
  const updateDraft = (changes: Partial<EmailFormDraft>) => setDraft((current) => ({ ...current, ...changes }));

  // UI 状态
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [showPassword, setShowPassword] = useState(false);

  // 同步外部状态：已保存的配置变了（保存成功、别处修改），表单重新按它填（密码不从后端加载，一并清空）。
  // 在渲染期间按「外部值变了」调整，而不是等 effect：不会先画出一帧旧内容
  const [syncedFingerprint, setSyncedFingerprint] = useState(savedFingerprint);
  if (syncedFingerprint !== savedFingerprint) {
    setSyncedFingerprint(savedFingerprint);
    setDraft(draftFromSaved());
  }

  // 有没有未保存的修改：由草稿与已保存的配置直接算出，不另存一份状态
  const hasChanges = emailFormHasChanges(draft, config, enabled);

  const handleToggle = async () => {
    if (readOnly) return;
    const newEnabled = !draft.enabled;
    updateDraft({ enabled: newEnabled });
    setSaving(true);
    try {
      await onSave({ enabled: newEnabled });
    } finally {
      setSaving(false);
    }
  };

  const handleSave = async () => {
    if (readOnly || !hasChanges) return;
    setSaving(true);
    try {
      const data: EmailUpdateData = {
        enabled: draft.enabled,
        smtpHost: draft.smtpHost,
        smtpPort: draft.smtpPort,
        smtpUser: draft.smtpUser,
        smtpTLS: draft.smtpTls,
        fromAddress: draft.smtpUser, // 发件人地址与用户名相同
        toAddresses: draft.recipients,
      };
      // 只有输入了密码才更新
      if (draft.smtpPassword) {
        data.smtpPassword = draft.smtpPassword;
      }
      await onSave(data);
      updateDraft({ smtpPassword: "" }); // 保存后清空密码输入
    } finally {
      setSaving(false);
    }
  };

  const handleTest = async () => {
    setTesting(true);
    try {
      await onTest();
    } finally {
      setTesting(false);
    }
  };

  const handleSmtpHostChange = (host: string, isCustom: boolean, port?: number) => {
    updateDraft({ isCustomSmtp: isCustom, smtpHost: host, ...(port !== undefined ? { smtpPort: port } : {}) });
  };

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
      {/* 头部 */}
      <div className="flex items-center justify-between px-6 py-4 border-b border-[var(--border-color)]">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-lg bg-blue-100 dark:bg-blue-900/40 flex items-center justify-center">
            <Mail className="w-5 h-5 text-blue-600 dark:text-blue-400" />
          </div>
          <div>
            <h3 className="font-medium text-default">{nt.emailNotify}</h3>
            <p className="text-sm text-muted">
              {effectiveEnabled ? (
                <span className="text-green-600">{nt.statusEnabled}</span>
              ) : draft.enabled ? (
                <span className="text-yellow-600">{nt.statusIncomplete}</span>
              ) : (
                nt.statusDisabled
              )}
            </p>
          </div>
        </div>

        {/* 开关 */}
        <button
          onClick={handleToggle}
          disabled={readOnly || saving}
          className={`relative w-12 h-6 rounded-full transition-colors ${
            draft.enabled ? "bg-green-500" : "bg-gray-300 dark:bg-gray-600"
          } ${readOnly ? "opacity-50 cursor-not-allowed" : "cursor-pointer"}`}
        >
          <span
            className={`absolute top-1 left-1 w-4 h-4 rounded-full bg-white transition-transform ${
              draft.enabled ? "translate-x-6" : "translate-x-0"
            }`}
          />
        </button>
      </div>

      {/* 校验错误提示 */}
      {validationErrors.length > 0 && (
        <div className="mx-6 mt-4 p-3 rounded-lg bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800">
          <div className="flex items-start gap-2">
            <AlertCircle className="w-4 h-4 text-yellow-600 mt-0.5 flex-shrink-0" />
            <div className="text-sm text-yellow-700 dark:text-yellow-400">
              <p className="font-medium mb-1">{nt.configIncomplete}</p>
              <ul className="list-disc list-inside space-y-0.5">
                {validationErrors.map((err, i) => (
                  <li key={i}>{err}</li>
                ))}
              </ul>
            </div>
          </div>
        </div>
      )}

      {/* SMTP 配置表单 */}
      <EmailFormFields
        smtpPresets={SMTP_PRESETS}
        isCustomSmtp={draft.isCustomSmtp}
        smtpHost={draft.smtpHost}
        smtpPort={draft.smtpPort}
        smtpUser={draft.smtpUser}
        smtpPassword={draft.smtpPassword}
        smtpTls={draft.smtpTls}
        recipients={draft.recipients}
        showPassword={showPassword}
        readOnly={readOnly}
        onSmtpHostChange={handleSmtpHostChange}
        onSmtpPortChange={(smtpPort) => updateDraft({ smtpPort })}
        onSmtpUserChange={(smtpUser) => updateDraft({ smtpUser })}
        onSmtpPasswordChange={(smtpPassword) => updateDraft({ smtpPassword })}
        onSmtpTlsToggle={() => updateDraft({ smtpTls: !draft.smtpTls })}
        onRecipientsChange={(recipients) => updateDraft({ recipients })}
        onShowPasswordToggle={() => setShowPassword(!showPassword)}
        labels={{
          smtpServer: nt.smtpServer,
          smtpServerPlaceholder: nt.smtpServerPlaceholder,
          smtpCustom: nt.smtpCustom,
          smtpCustomPlaceholder: nt.smtpCustomPlaceholder,
          port: nt.port,
          portHint: nt.portHint,
          emailAccount: nt.emailAccount,
          emailAccountHint: nt.emailAccountHint,
          password: nt.password,
          passwordPlaceholder: nt.passwordPlaceholder,
          tlsEncryption: nt.tlsEncryption,
          tlsHint: nt.tlsHint,
          recipients: nt.recipients,
          recipientsPlaceholder: nt.recipientsPlaceholder,
          tagInputDuplicate: nt.tagInputDuplicate,
          tagInputInvalidFormat: nt.tagInputInvalidFormat,
        }}
      />

      {/* 操作按钮 */}
      {!readOnly && (
        <div className="flex justify-end gap-3 px-6 py-4 border-t border-[var(--border-color)] bg-[var(--bg-secondary)]">
          <button
            onClick={handleTest}
            disabled={testing || !effectiveEnabled}
            className="px-4 py-2 text-sm rounded-lg border border-[var(--border-color)]
              bg-[var(--bg-primary)] text-default
              hover:bg-[var(--bg-secondary)]
              disabled:opacity-50 disabled:cursor-not-allowed
              transition-colors flex items-center gap-2"
          >
            {testing && <Loader2 className="w-4 h-4 animate-spin" />}
            {nt.test}
          </button>
          <button
            onClick={handleSave}
            disabled={saving || !hasChanges}
            className="px-4 py-2 text-sm rounded-lg
              bg-blue-600 text-white
              hover:bg-blue-700
              disabled:opacity-50 disabled:cursor-not-allowed
              transition-colors flex items-center gap-2"
          >
            {saving && <Loader2 className="w-4 h-4 animate-spin" />}
            {nt.save}
          </button>
        </div>
      )}
    </div>
  );
}
