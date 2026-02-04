"use client";

import { useState, useEffect } from "react";
import { Mail, Loader2, AlertCircle, Eye, EyeOff } from "lucide-react";
import { useI18n } from "@/i18n/context";
import { TagInput, emailValidator } from "./TagInput";
import type { EmailConfig, EmailUpdateData } from "@/api/notify";

// 常见 SMTP 服务器列表
const SMTP_PRESETS = [
  { value: "smtp.gmail.com", label: "Gmail", port: 587 },
  { value: "smtp.office365.com", label: "Outlook / Office 365", port: 587 },
  { value: "smtp.qq.com", label: "QQ 邮箱", port: 587 },
  { value: "smtp.163.com", label: "163 邮箱", port: 465 },
  { value: "smtp.126.com", label: "126 邮箱", port: 465 },
  { value: "smtp.exmail.qq.com", label: "腾讯企业邮", port: 465 },
  { value: "smtp.mxhichina.com", label: "阿里企业邮", port: 465 },
  { value: "smtp.feishu.cn", label: "飞书邮箱", port: 465 },
];

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

  // 表单状态
  const [localEnabled, setLocalEnabled] = useState(enabled);
  const [smtpHost, setSmtpHost] = useState(config.smtp_host || "");
  const [smtpPort, setSmtpPort] = useState(config.smtp_port || 587);
  const [smtpUser, setSmtpUser] = useState(config.smtp_user || "");
  const [smtpPassword, setSmtpPassword] = useState("");
  const [smtpTls, setSmtpTls] = useState(config.smtp_tls ?? true);
  const [recipients, setRecipients] = useState<string[]>(config.to_addresses || []);

  // UI 状态
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [hasChanges, setHasChanges] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  const [isCustomSmtp, setIsCustomSmtp] = useState(false);

  // 判断当前 SMTP 是否在预设列表中
  const isPresetSmtp = SMTP_PRESETS.some((p) => p.value === smtpHost);

  // 同步外部状态
  useEffect(() => {
    setLocalEnabled(enabled);
    setSmtpHost(config.smtp_host || "");
    setSmtpPort(config.smtp_port || 587);
    setSmtpUser(config.smtp_user || "");
    setSmtpTls(config.smtp_tls ?? true);
    setRecipients(config.to_addresses || []);
    setSmtpPassword(""); // 密码不从后端加载
    // 判断是否为自定义 SMTP
    const isPreset = SMTP_PRESETS.some((p) => p.value === config.smtp_host);
    setIsCustomSmtp(config.smtp_host !== "" && !isPreset);
  }, [enabled, config]);

  // 检测变化
  useEffect(() => {
    const originalRecipients = config.to_addresses || [];
    const recipientsChanged =
      recipients.length !== originalRecipients.length ||
      recipients.some((r, i) => r !== originalRecipients[i]);

    const changed =
      localEnabled !== enabled ||
      smtpHost !== (config.smtp_host || "") ||
      smtpPort !== (config.smtp_port || 587) ||
      smtpUser !== (config.smtp_user || "") ||
      smtpPassword !== "" || // 密码有输入即视为变化
      smtpTls !== (config.smtp_tls ?? true) ||
      recipientsChanged;

    setHasChanges(changed);
  }, [
    localEnabled,
    smtpHost,
    smtpPort,
    smtpUser,
    smtpPassword,
    smtpTls,
    recipients,
    enabled,
    config,
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
        smtp_host: smtpHost,
        smtp_port: smtpPort,
        smtp_user: smtpUser,
        smtp_tls: smtpTls,
        from_address: smtpUser, // 发件人地址与用户名相同
        to_addresses: recipients,
      };
      // 只有输入了密码才更新
      if (smtpPassword) {
        data.smtp_password = smtpPassword;
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
      <div className="px-6 py-4 space-y-4">
        {/* SMTP 服务器 */}
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-default mb-1">
              {nt.smtpServer} <span className="text-red-500">*</span>
            </label>
            <select
              value={isCustomSmtp ? "__custom__" : smtpHost}
              onChange={(e) => {
                const val = e.target.value;
                if (val === "__custom__") {
                  setIsCustomSmtp(true);
                  setSmtpHost("");
                } else {
                  setIsCustomSmtp(false);
                  setSmtpHost(val);
                  // 自动设置对应的端口
                  const preset = SMTP_PRESETS.find((p) => p.value === val);
                  if (preset) {
                    setSmtpPort(preset.port);
                  }
                }
              }}
              disabled={readOnly}
              className="w-full px-3 py-2 rounded-lg border border-[var(--border-color)]
                bg-[var(--bg-primary)] text-default
                focus:outline-none focus:ring-2 focus:ring-blue-500
                disabled:opacity-50 disabled:cursor-not-allowed
                [&>option]:bg-white [&>option]:text-gray-900
                dark:[&>option]:bg-gray-800 dark:[&>option]:text-gray-100"
            >
              <option value="">{nt.smtpServerPlaceholder}</option>
              {SMTP_PRESETS.map((preset) => (
                <option key={preset.value} value={preset.value}>
                  {preset.label} ({preset.value})
                </option>
              ))}
              <option value="__custom__">{nt.smtpCustom}</option>
            </select>
            {isCustomSmtp && (
              <input
                type="text"
                value={smtpHost}
                onChange={(e) => setSmtpHost(e.target.value)}
                placeholder={nt.smtpCustomPlaceholder}
                disabled={readOnly}
                className="w-full mt-2 px-3 py-2 rounded-lg border border-[var(--border-color)]
                  bg-[var(--bg-primary)] text-default
                  placeholder:text-muted
                  focus:outline-none focus:ring-2 focus:ring-blue-500
                  disabled:opacity-50 disabled:cursor-not-allowed"
              />
            )}
          </div>
          <div>
            <label className="block text-sm font-medium text-default mb-1">
              {nt.port} <span className="text-red-500">*</span>
            </label>
            <input
              type="number"
              value={smtpPort}
              onChange={(e) => setSmtpPort(parseInt(e.target.value) || 587)}
              placeholder="587"
              disabled={readOnly}
              className="w-full px-3 py-2 rounded-lg border border-[var(--border-color)]
                bg-[var(--bg-primary)] text-default
                placeholder:text-muted
                focus:outline-none focus:ring-2 focus:ring-blue-500
                disabled:opacity-50 disabled:cursor-not-allowed"
            />
            <p className="mt-1 text-xs text-muted">
              {nt.portHint}
            </p>
          </div>
        </div>

        {/* 邮箱账号和密码 */}
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-default mb-1">
              {nt.emailAccount} <span className="text-red-500">*</span>
            </label>
            <input
              type="email"
              value={smtpUser}
              onChange={(e) => setSmtpUser(e.target.value)}
              placeholder="user@example.com"
              disabled={readOnly}
              className="w-full px-3 py-2 rounded-lg border border-[var(--border-color)]
                bg-[var(--bg-primary)] text-default
                placeholder:text-muted
                focus:outline-none focus:ring-2 focus:ring-blue-500
                disabled:opacity-50 disabled:cursor-not-allowed"
            />
            <p className="mt-1 text-xs text-muted">{nt.emailAccountHint}</p>
          </div>
          <div>
            <label className="block text-sm font-medium text-default mb-1">
              {nt.password} <span className="text-red-500">*</span>
            </label>
            <div className="relative">
              <input
                type={showPassword ? "text" : "password"}
                value={smtpPassword}
                onChange={(e) => setSmtpPassword(e.target.value)}
                placeholder={nt.passwordPlaceholder}
                disabled={readOnly}
                className="w-full px-3 py-2 pr-10 rounded-lg border border-[var(--border-color)]
                  bg-[var(--bg-primary)] text-default
                  placeholder:text-muted
                  focus:outline-none focus:ring-2 focus:ring-blue-500
                  disabled:opacity-50 disabled:cursor-not-allowed"
              />
              <button
                type="button"
                onClick={() => setShowPassword(!showPassword)}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-muted hover:text-default"
              >
                {showPassword ? (
                  <EyeOff className="w-4 h-4" />
                ) : (
                  <Eye className="w-4 h-4" />
                )}
              </button>
            </div>
          </div>
        </div>

        {/* TLS 加密 */}
        <div className="flex items-center justify-between py-2">
          <div>
            <label className="block text-sm font-medium text-default">
              {nt.tlsEncryption}
            </label>
            <p className="text-xs text-muted">{nt.tlsHint}</p>
          </div>
          <button
            onClick={() => !readOnly && setSmtpTls(!smtpTls)}
            disabled={readOnly}
            className={`relative w-12 h-6 rounded-full transition-colors ${
              smtpTls ? "bg-green-500" : "bg-gray-300 dark:bg-gray-600"
            } ${readOnly ? "opacity-50 cursor-not-allowed" : "cursor-pointer"}`}
          >
            <span
              className={`absolute top-1 left-1 w-4 h-4 rounded-full bg-white transition-transform ${
                smtpTls ? "translate-x-6" : "translate-x-0"
              }`}
            />
          </button>
        </div>

        {/* 收件人 */}
        <div>
          <label className="block text-sm font-medium text-default mb-1">
            {nt.recipients}
          </label>
          <TagInput
            value={recipients}
            onChange={setRecipients}
            placeholder={nt.recipientsPlaceholder}
            disabled={readOnly}
            validator={emailValidator}
          />
        </div>
      </div>

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
