"use client";

import { Eye, EyeOff } from "lucide-react";
import { TagInput, emailValidator } from "./TagInput";
import type { SmtpPreset } from "./smtp-presets";

interface EmailFormFieldsProps {
  smtpPresets: SmtpPreset[];
  isCustomSmtp: boolean;
  smtpHost: string;
  smtpPort: number;
  smtpUser: string;
  smtpPassword: string;
  smtpTls: boolean;
  recipients: string[];
  showPassword: boolean;
  readOnly: boolean;
  onSmtpHostChange: (host: string, isCustom: boolean, port?: number) => void;
  onSmtpPortChange: (port: number) => void;
  onSmtpUserChange: (user: string) => void;
  onSmtpPasswordChange: (password: string) => void;
  onSmtpTlsToggle: () => void;
  onRecipientsChange: (recipients: string[]) => void;
  onShowPasswordToggle: () => void;
  labels: {
    smtpServer: string;
    smtpServerPlaceholder: string;
    smtpCustom: string;
    smtpCustomPlaceholder: string;
    port: string;
    portHint: string;
    emailAccount: string;
    emailAccountHint: string;
    password: string;
    passwordPlaceholder: string;
    tlsEncryption: string;
    tlsHint: string;
    recipients: string;
    recipientsPlaceholder: string;
    tagInputDuplicate: string;
    tagInputInvalidFormat: string;
  };
}

const inputClass = `w-full px-3 py-2 rounded-lg border border-[var(--border-color)]
  bg-[var(--bg-primary)] text-default
  placeholder:text-muted
  focus:outline-none focus:ring-2 focus:ring-blue-500
  disabled:opacity-50 disabled:cursor-not-allowed`;

const selectClass = `w-full px-3 py-2 rounded-lg border border-[var(--border-color)]
  bg-[var(--bg-primary)] text-default
  focus:outline-none focus:ring-2 focus:ring-blue-500
  disabled:opacity-50 disabled:cursor-not-allowed
  [&>option]:bg-white [&>option]:text-gray-900
  dark:[&>option]:bg-gray-800 dark:[&>option]:text-gray-100`;

export function EmailFormFields({
  smtpPresets,
  isCustomSmtp,
  smtpHost,
  smtpPort,
  smtpUser,
  smtpPassword,
  smtpTls,
  recipients,
  showPassword,
  readOnly,
  onSmtpHostChange,
  onSmtpPortChange,
  onSmtpUserChange,
  onSmtpPasswordChange,
  onSmtpTlsToggle,
  onRecipientsChange,
  onShowPasswordToggle,
  labels,
}: EmailFormFieldsProps) {
  return (
    <div className="px-6 py-4 space-y-4">
      {/* SMTP 服务器 */}
      <div className="grid grid-cols-2 gap-4">
        <div>
          <label className="block text-sm font-medium text-default mb-1">
            {labels.smtpServer} <span className="text-red-500">*</span>
          </label>
          <select
            value={isCustomSmtp ? "__custom__" : smtpHost}
            onChange={(e) => {
              const val = e.target.value;
              if (val === "__custom__") {
                onSmtpHostChange("", true);
              } else {
                const preset = smtpPresets.find((p) => p.value === val);
                onSmtpHostChange(val, false, preset?.port);
              }
            }}
            disabled={readOnly}
            className={selectClass}
          >
            <option value="">{labels.smtpServerPlaceholder}</option>
            {smtpPresets.map((preset) => (
              <option key={preset.value} value={preset.value}>
                {preset.label} ({preset.value})
              </option>
            ))}
            <option value="__custom__">{labels.smtpCustom}</option>
          </select>
          {isCustomSmtp && (
            <input
              type="text"
              value={smtpHost}
              onChange={(e) => onSmtpHostChange(e.target.value, true)}
              placeholder={labels.smtpCustomPlaceholder}
              disabled={readOnly}
              className={`${inputClass} mt-2`}
            />
          )}
        </div>
        <div>
          <label className="block text-sm font-medium text-default mb-1">
            {labels.port} <span className="text-red-500">*</span>
          </label>
          <input
            type="number"
            value={smtpPort}
            onChange={(e) => onSmtpPortChange(parseInt(e.target.value) || 587)}
            placeholder="587"
            disabled={readOnly}
            className={inputClass}
          />
          <p className="mt-1 text-xs text-muted">
            {labels.portHint}
          </p>
        </div>
      </div>

      {/* 邮箱账号和密码 */}
      <div className="grid grid-cols-2 gap-4">
        <div>
          <label className="block text-sm font-medium text-default mb-1">
            {labels.emailAccount} <span className="text-red-500">*</span>
          </label>
          <input
            type="email"
            value={smtpUser}
            onChange={(e) => onSmtpUserChange(e.target.value)}
            placeholder="user@example.com"
            disabled={readOnly}
            className={inputClass}
          />
          <p className="mt-1 text-xs text-muted">{labels.emailAccountHint}</p>
        </div>
        <div>
          <label className="block text-sm font-medium text-default mb-1">
            {labels.password} <span className="text-red-500">*</span>
          </label>
          <div className="relative">
            <input
              type={showPassword ? "text" : "password"}
              value={smtpPassword}
              onChange={(e) => onSmtpPasswordChange(e.target.value)}
              placeholder={labels.passwordPlaceholder}
              disabled={readOnly}
              className={`${inputClass} pr-10`}
            />
            <button
              type="button"
              onClick={onShowPasswordToggle}
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
            {labels.tlsEncryption}
          </label>
          <p className="text-xs text-muted">{labels.tlsHint}</p>
        </div>
        <button
          onClick={() => !readOnly && onSmtpTlsToggle()}
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
          {labels.recipients}
        </label>
        <TagInput
          value={recipients}
          onChange={onRecipientsChange}
          placeholder={labels.recipientsPlaceholder}
          disabled={readOnly}
          validator={emailValidator}
          duplicateMessage={labels.tagInputDuplicate}
          invalidFormatMessage={labels.tagInputInvalidFormat}
        />
      </div>
    </div>
  );
}
