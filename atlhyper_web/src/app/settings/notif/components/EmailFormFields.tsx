"use client";

import { Eye, EyeOff } from "lucide-react";
import type { NotifySMTPTranslations } from "@/types/i18n-notify";
import type { SmtpFormDraft } from "./email-form";
import { HINT_CLASS, INPUT_CLASS, LABEL_CLASS, SELECT_CLASS } from "./shared/form-styles";
import { Switch } from "./shared/Switch";
import type { SmtpPreset } from "./smtp-presets";

interface EmailFormFieldsProps {
  draft: SmtpFormDraft;
  onChange: (changes: Partial<SmtpFormDraft>) => void;
  smtpPresets: SmtpPreset[];
  /** 已存过密码：密码框可以留空（保持原值），提示文案随之变化 */
  passwordConfigured: boolean;
  showPassword: boolean;
  onShowPasswordToggle: () => void;
  /** 保存进行中 */
  disabled: boolean;
  labels: NotifySMTPTranslations;
  /** TLS 开关读屏用的状态文案 */
  enabledLabel: string;
  disabledLabel: string;
}

/** 下拉框里「自定义」这一项的值：不会与任何服务器地址撞上 */
const CUSTOM_HOST = "__custom__";

/**
 * 邮件发件设置（全局 SMTP）的表单字段：服务器、端口、账号、密码、TLS、发件地址。
 *
 * 密码只写不读：输入框只放用户新输入的值，已配置时留空 = 保持原值。
 */
export function EmailFormFields({
  draft,
  onChange,
  smtpPresets,
  passwordConfigured,
  showPassword,
  onShowPasswordToggle,
  disabled,
  labels,
  enabledLabel,
  disabledLabel,
}: EmailFormFieldsProps) {
  const handleServerSelect = (selected: string) => {
    if (selected === CUSTOM_HOST) {
      onChange({ host: "", isCustomHost: true });
      return;
    }
    // 选了预设服务器就顺带填上它的常用端口
    const preset = smtpPresets.find((candidate) => candidate.value === selected);
    onChange({ host: selected, isCustomHost: false, ...(preset ? { port: preset.port } : {}) });
  };

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div>
          <label htmlFor="notify-smtp-server" className={LABEL_CLASS}>
            {labels.server} *
          </label>
          <select
            id="notify-smtp-server"
            value={draft.isCustomHost ? CUSTOM_HOST : draft.host}
            onChange={(event) => handleServerSelect(event.target.value)}
            disabled={disabled}
            className={SELECT_CLASS}
          >
            <option value="">{labels.serverPlaceholder}</option>
            {smtpPresets.map((preset) => (
              <option key={preset.value} value={preset.value}>
                {preset.label} ({preset.value})
              </option>
            ))}
            <option value={CUSTOM_HOST}>{labels.custom}</option>
          </select>
          {draft.isCustomHost && (
            <input
              type="text"
              value={draft.host}
              onChange={(event) => onChange({ host: event.target.value })}
              placeholder={labels.customPlaceholder}
              aria-label={labels.customPlaceholder}
              autoComplete="off"
              spellCheck={false}
              disabled={disabled}
              className={`${INPUT_CLASS} mt-2`}
            />
          )}
        </div>
        <div>
          <label htmlFor="notify-smtp-port" className={LABEL_CLASS}>
            {labels.port} *
          </label>
          <input
            id="notify-smtp-port"
            type="number"
            value={Number.isNaN(draft.port) ? "" : draft.port}
            // 清空时记为 NaN，由校验报「端口不合规」，不悄悄换成默认值
            onChange={(event) => onChange({ port: event.target.value === "" ? Number.NaN : Number(event.target.value) })}
            placeholder="587"
            min={1}
            max={65535}
            disabled={disabled}
            className={INPUT_CLASS}
          />
          <p className={HINT_CLASS}>{labels.portHint}</p>
        </div>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div>
          <label htmlFor="notify-smtp-user" className={LABEL_CLASS}>
            {labels.account} *
          </label>
          <input
            id="notify-smtp-user"
            type="text"
            value={draft.user}
            onChange={(event) => onChange({ user: event.target.value })}
            placeholder="user@example.com"
            autoComplete="off"
            spellCheck={false}
            disabled={disabled}
            className={INPUT_CLASS}
          />
          <p className={HINT_CLASS}>{labels.accountHint}</p>
        </div>
        <div>
          <label htmlFor="notify-smtp-password" className={LABEL_CLASS}>
            {labels.password} {!passwordConfigured && "*"}
          </label>
          <div className="relative">
            <input
              id="notify-smtp-password"
              type={showPassword ? "text" : "password"}
              value={draft.password}
              onChange={(event) => onChange({ password: event.target.value })}
              placeholder={labels.passwordPlaceholder}
              // 不让浏览器把登录本站用的密码自动填进来
              autoComplete="new-password"
              disabled={disabled}
              className={`${INPUT_CLASS} pr-10`}
            />
            <button
              type="button"
              onClick={onShowPasswordToggle}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-muted hover:text-default"
            >
              {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
            </button>
          </div>
          {passwordConfigured && <p className={HINT_CLASS}>{labels.passwordKeepHint}</p>}
        </div>
      </div>

      <div>
        <label htmlFor="notify-smtp-from" className={LABEL_CLASS}>
          {labels.fromAddress}
        </label>
        <input
          id="notify-smtp-from"
          type="email"
          value={draft.fromAddress}
          onChange={(event) => onChange({ fromAddress: event.target.value })}
          placeholder={labels.fromAddressPlaceholder}
          autoComplete="off"
          spellCheck={false}
          disabled={disabled}
          className={INPUT_CLASS}
        />
        <p className={HINT_CLASS}>{labels.fromAddressHint}</p>
      </div>

      <div className="flex items-center justify-between gap-4">
        <div>
          <span className="block text-sm font-medium text-default">{labels.tls}</span>
          <p className="text-xs text-muted">{labels.tlsHint}</p>
        </div>
        <Switch
          checked={draft.tls}
          onChange={(tls) => onChange({ tls })}
          disabled={disabled}
          label={draft.tls ? enabledLabel : disabledLabel}
        />
      </div>
    </div>
  );
}
