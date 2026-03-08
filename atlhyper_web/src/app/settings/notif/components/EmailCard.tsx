"use client";

import { useState, useEffect } from "react";
import { Mail, Loader2, AlertCircle } from "lucide-react";
import { useI18n } from "@/i18n/context";
import { getSmtpPresets } from "./smtp-presets";
import { EmailFormFields } from "./EmailFormFields";
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

  // 表单状态
  const [localEnabled, setLocalEnabled] = useState(enabled);
  const [smtpHost, setSmtpHost] = useState(config.smtpHost || "");
  const [smtpPort, setSmtpPort] = useState(config.smtpPort || 587);
  const [smtpUser, setSmtpUser] = useState(config.smtpUser || "");
  const [smtpPassword, setSmtpPassword] = useState("");
  const [smtpTls, setSmtpTls] = useState(config.smtpTLS ?? true);
  const [recipients, setRecipients] = useState<string[]>(config.toAddresses || []);

  // UI 状态
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [hasChanges, setHasChanges] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  const [isCustomSmtp, setIsCustomSmtp] = useState(false);

  // 同步外部状态
  useEffect(() => {
    setLocalEnabled(enabled);
    setSmtpHost(config.smtpHost || "");
    setSmtpPort(config.smtpPort || 587);
    setSmtpUser(config.smtpUser || "");
    setSmtpTls(config.smtpTLS ?? true);
    setRecipients(config.toAddresses || []);
    setSmtpPassword(""); // 密码不从后端加载
    // 判断是否为自定义 SMTP
    const isPreset = SMTP_PRESETS.some((p) => p.value === config.smtpHost);
    setIsCustomSmtp(config.smtpHost !== "" && !isPreset);
  }, [enabled, config]);

  // 检测变化
  useEffect(() => {
    const originalRecipients = config.toAddresses || [];
    const recipientsChanged =
      recipients.length !== originalRecipients.length ||
      recipients.some((r, i) => r !== originalRecipients[i]);

    const changed =
      localEnabled !== enabled ||
      smtpHost !== (config.smtpHost || "") ||
      smtpPort !== (config.smtpPort || 587) ||
      smtpUser !== (config.smtpUser || "") ||
      smtpPassword !== "" || // 密码有输入即视为变化
      smtpTls !== (config.smtpTLS ?? true) ||
      recipientsChanged;

    setHasChanges(changed);
  }, [
    localEnabled, smtpHost, smtpPort, smtpUser,
    smtpPassword, smtpTls, recipients, enabled, config,
  ]);

  const handleToggle = async () => {
    if (readOnly) return;
    const newEnabled = !localEnabled;
    setLocalEnabled(newEnabled);
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
        enabled: localEnabled,
        smtpHost: smtpHost,
        smtpPort: smtpPort,
        smtpUser: smtpUser,
        smtpTLS: smtpTls,
        fromAddress: smtpUser, // 发件人地址与用户名相同
        toAddresses: recipients,
      };
      // 只有输入了密码才更新
      if (smtpPassword) {
        data.smtpPassword = smtpPassword;
      }
      await onSave(data);
      setSmtpPassword(""); // 保存后清空密码输入
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
    setIsCustomSmtp(isCustom);
    setSmtpHost(host);
    if (port !== undefined) {
      setSmtpPort(port);
    }
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
              ) : localEnabled ? (
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
            localEnabled ? "bg-green-500" : "bg-gray-300 dark:bg-gray-600"
          } ${readOnly ? "opacity-50 cursor-not-allowed" : "cursor-pointer"}`}
        >
          <span
            className={`absolute top-1 left-1 w-4 h-4 rounded-full bg-white transition-transform ${
              localEnabled ? "translate-x-6" : "translate-x-0"
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
        isCustomSmtp={isCustomSmtp}
        smtpHost={smtpHost}
        smtpPort={smtpPort}
        smtpUser={smtpUser}
        smtpPassword={smtpPassword}
        smtpTls={smtpTls}
        recipients={recipients}
        showPassword={showPassword}
        readOnly={readOnly}
        onSmtpHostChange={handleSmtpHostChange}
        onSmtpPortChange={setSmtpPort}
        onSmtpUserChange={setSmtpUser}
        onSmtpPasswordChange={setSmtpPassword}
        onSmtpTlsToggle={() => setSmtpTls(!smtpTls)}
        onRecipientsChange={setRecipients}
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
