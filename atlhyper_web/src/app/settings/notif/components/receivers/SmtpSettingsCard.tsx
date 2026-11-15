"use client";

import { useState, type ReactNode } from "react";
import type { UseQueryResult } from "@tanstack/react-query";
import { AlertTriangle, Mail, Pencil } from "lucide-react";
import { QueryView } from "@/components/common";
import type { CommonTranslations, NotificationsTranslations } from "@/types/i18n";
import type { NotifySMTP } from "@/types/notify";
import { formatTimestamp } from "../shared/format";
import { PRIMARY_BUTTON_CLASS, SECONDARY_BUTTON_CLASS } from "../shared/form-styles";
import { SmtpSettingsForm } from "./SmtpSettingsForm";

interface SmtpSettingsCardProps {
  /** 真实数据的查询；演示模式下不用 */
  smtpQuery: UseQueryResult<NotifySMTP>;
  /** 演示数据：有值时显示它、不可编辑；null = 显示真实数据 */
  demoSMTP: NotifySMTP | null;
  t: NotificationsTranslations;
  commonT: CommonTranslations;
}

/** 查看状态下的一项：标签 + 值 */
function SummaryItem({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="min-w-0">
      <dt className="text-xs text-muted">{label}</dt>
      <dd className="text-sm text-default break-all">{children}</dd>
    </div>
  );
}

/** 查看状态：当前设置一览 + 编辑按钮；还没配置时给出醒目提示 */
function SmtpSummary({
  smtp,
  readOnly,
  onEdit,
  t,
  editLabel,
}: {
  smtp: NotifySMTP;
  readOnly: boolean;
  onEdit: () => void;
  t: NotificationsTranslations["smtp"];
  editLabel: string;
}) {
  const shown = (value: string) => value || "-";
  return (
    <div className="space-y-4">
      {!smtp.configured && (
        <div
          className="flex items-start gap-2 rounded-lg border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-amber-700 dark:text-amber-400"
          role="status"
        >
          <AlertTriangle className="w-4 h-4 mt-0.5 flex-shrink-0" />
          <div>
            <p className="text-sm font-medium">{t.notConfigured}</p>
            <p className="text-xs">{t.notConfiguredHint}</p>
          </div>
        </div>
      )}

      <dl className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-x-6 gap-y-3">
        <SummaryItem label={t.server}>{smtp.host ? `${smtp.host}:${smtp.port}` : "-"}</SummaryItem>
        <SummaryItem label={t.account}>{shown(smtp.user)}</SummaryItem>
        <SummaryItem label={t.password}>
          {smtp.passwordConfigured ? (
            t.passwordConfigured
          ) : (
            <span className="text-amber-600 dark:text-amber-400">{t.passwordNotConfigured}</span>
          )}
        </SummaryItem>
        <SummaryItem label={t.tls}>{smtp.tls ? t.tlsOn : t.tlsOff}</SummaryItem>
        <SummaryItem label={t.fromAddress}>
          {smtp.fromAddress || (smtp.user ? t.fromAddressUsesAccount.replace("{user}", smtp.user) : "-")}
        </SummaryItem>
        <SummaryItem label={t.updatedAt}>{smtp.updatedAt ? formatTimestamp(smtp.updatedAt) : t.neverSaved}</SummaryItem>
      </dl>

      <div className="flex justify-end">
        {/* 演示模式下写操作不可用；还没配置时用主按钮引导去配置 */}
        <button
          type="button"
          onClick={onEdit}
          disabled={readOnly}
          className={smtp.configured ? `${SECONDARY_BUTTON_CLASS} flex items-center gap-1.5` : PRIMARY_BUTTON_CLASS}
        >
          <Pencil className="w-4 h-4" />
          {smtp.configured ? editLabel : t.configure}
        </button>
      </div>
    </div>
  );
}

/**
 * 「接收人」页签顶部的邮件发件设置（全局 SMTP，所有邮件接收人共用）。
 *
 * 查看与编辑两种状态；保存成功或取消后回到查看状态。
 */
export function SmtpSettingsCard({ smtpQuery, demoSMTP, t, commonT }: SmtpSettingsCardProps) {
  const smtpT = t.smtp;
  const [editing, setEditing] = useState(false);

  const renderSMTP = (smtp: NotifySMTP) =>
    editing && demoSMTP === null ? (
      <SmtpSettingsForm smtp={smtp} onDone={() => setEditing(false)} t={t} commonT={commonT} />
    ) : (
      <SmtpSummary
        smtp={smtp}
        readOnly={demoSMTP !== null}
        onEdit={() => setEditing(true)}
        t={smtpT}
        editLabel={commonT.edit}
      />
    );

  return (
    <section className="bg-card rounded-xl border border-[var(--border-color)] p-4 sm:p-6 space-y-4">
      <div className="flex items-start gap-3">
        <Mail className="w-5 h-5 mt-0.5 text-muted flex-shrink-0" />
        <div>
          <h3 className="text-sm font-semibold text-default">{smtpT.title}</h3>
          <p className="text-xs text-muted mt-0.5">{smtpT.description}</p>
        </div>
      </div>
      {demoSMTP !== null ? renderSMTP(demoSMTP) : <QueryView query={smtpQuery}>{renderSMTP}</QueryView>}
    </section>
  );
}
