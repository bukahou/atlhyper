"use client";

import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { AlertTriangle, Eye, EyeOff } from "lucide-react";
import { Modal, toast } from "@/components/common";
import { notifyFailureText } from "@/lib/notify-failure";
import { sealCredential } from "@/lib/notify-receiver-input";
import { saveNotifyReceiverMutation } from "@/queries/notify-routing";
import type { CommonTranslations, NotificationsTranslations } from "@/types/i18n";
import type { NotifyReceiver, NotifyReceiverType } from "@/types/notify";
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
import { TagInput, emailValidator } from "../TagInput";
import {
  receiverCredentialFromForm,
  receiverFormFrom,
  receiverFormProblems,
  receiverSettingsFromForm,
  type ReceiverFormDraft,
} from "./receiver-form";

interface ReceiverModalProps {
  /** 要编辑的接收人；null = 新建 */
  editing: NotifyReceiver | null;
  /** 全局发件设置还没配置：邮件接收人发不出去，表单里给出提示 */
  smtpMissing: boolean;
  onClose: () => void;
  t: NotificationsTranslations;
  commonT: CommonTranslations;
}

/**
 * 新建 / 编辑接收人的表单。
 *
 * 由外层按目标整个重新挂载（key），所以表单初值直接取自 editing。
 * 凭据只写不读：编辑时输入框是空的，留空则保持已存的值不变。
 * 邮件接收人只填收件人：发件服务器与账号是全局的邮件发件设置。
 */
export function ReceiverModal({ editing, smtpMissing, onClose, t, commonT }: ReceiverModalProps) {
  const receiversT = t.receivers;
  const [draft, setDraft] = useState<ReceiverFormDraft>(() => receiverFormFrom(editing));
  // Webhook 地址默认遮住，可以点开核对
  const [showCredential, setShowCredential] = useState(false);
  // 没点过提交就不报错：一打开就满屏红字没有意义
  const [submitAttempted, setSubmitAttempted] = useState(false);
  const [serverError, setServerError] = useState("");

  // 成功后工厂会让列表重新取数，这里只管提示与关弹窗
  const saveReceiver = useMutation(saveNotifyReceiverMutation());

  const updateDraft = (changes: Partial<ReceiverFormDraft>) => setDraft((current) => ({ ...current, ...changes }));

  const problems = receiverFormProblems(draft, editing);
  const webhookAlreadySet = editing?.slack?.webhookConfigured === true;

  const handleSubmit = (event: React.FormEvent) => {
    event.preventDefault();
    setSubmitAttempted(true);
    setServerError("");
    if (problems.length > 0) return;

    saveReceiver.mutate(
      {
        receiverId: editing?.id,
        settings: receiverSettingsFromForm(draft),
        // 明文装进一次性的盒子：请求发出即清空，不留在 mutation 缓存里
        takeCredential: sealCredential(receiverCredentialFromForm(draft)),
      },
      {
        onSuccess: () => {
          toast.success(editing ? receiversT.updatedToast : receiversT.createdToast);
          onClose();
        },
        onError: (error) => setServerError(notifyFailureText(error, t.errors)),
      },
    );
  };

  return (
    <Modal isOpen onClose={onClose} title={editing ? receiversT.editTitle : receiversT.createTitle} size="md">
      <form onSubmit={handleSubmit} className="p-6 space-y-4" noValidate>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label htmlFor="notify-receiver-name" className={LABEL_CLASS}>
              {receiversT.name} *
            </label>
            <input
              id="notify-receiver-name"
              type="text"
              value={draft.name}
              onChange={(event) => updateDraft({ name: event.target.value })}
              placeholder={receiversT.namePlaceholder}
              autoComplete="off"
              className={INPUT_CLASS}
            />
          </div>
          <div>
            <label htmlFor="notify-receiver-type" className={LABEL_CLASS}>
              {receiversT.type}
            </label>
            <select
              id="notify-receiver-type"
              value={draft.type}
              onChange={(event) => updateDraft({ type: event.target.value as NotifyReceiverType })}
              // 已有的接收人不换类型：两种类型的凭据互不相通
              disabled={editing !== null}
              className={SELECT_CLASS}
            >
              <option value="slack">{receiversT.typeSlack}</option>
              <option value="email">{receiversT.typeEmail}</option>
            </select>
          </div>
        </div>

        <div className="flex items-center justify-between">
          <span className="text-sm font-medium text-muted">{receiversT.enabled}</span>
          <Switch
            checked={draft.enabled}
            onChange={(enabled) => updateDraft({ enabled })}
            label={draft.enabled ? t.enabledLabel : t.disabledLabel}
          />
        </div>

        {draft.type === "slack" ? (
          <div>
            <label htmlFor="notify-receiver-webhook" className={LABEL_CLASS}>
              {receiversT.webhookUrl} {!webhookAlreadySet && "*"}
            </label>
            <div className="relative">
              <input
                id="notify-receiver-webhook"
                type={showCredential ? "text" : "password"}
                value={draft.webhookUrl}
                onChange={(event) => updateDraft({ webhookUrl: event.target.value })}
                placeholder={webhookAlreadySet ? receiversT.credentialKeepHint : "https://hooks.slack.com/services/..."}
                autoComplete="off"
                spellCheck={false}
                className={`${INPUT_CLASS} pr-10 font-mono`}
              />
              <button
                type="button"
                onClick={() => setShowCredential((shown) => !shown)}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-muted hover:text-default"
              >
                {showCredential ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
              </button>
            </div>
            <p className={HINT_CLASS}>{receiversT.webhookUrlHint}</p>
          </div>
        ) : (
          <div className="space-y-2">
            {/* TagInput 不接受 id，标签无法用 htmlFor 关联，用普通文字 */}
            <span className={LABEL_CLASS}>{receiversT.recipients} *</span>
            <TagInput
              value={draft.recipients}
              onChange={(recipients) => updateDraft({ recipients })}
              placeholder={receiversT.recipientsPlaceholder}
              validator={emailValidator}
              duplicateMessage={receiversT.tagInputDuplicate}
              invalidFormatMessage={receiversT.tagInputInvalidFormat}
            />
            <p className={HINT_CLASS}>{receiversT.emailSenderHint}</p>
            {smtpMissing && (
              <p className="flex items-center gap-1 text-xs text-amber-600 dark:text-amber-400" role="status">
                <AlertTriangle className="w-3.5 h-3.5 flex-shrink-0" />
                {t.smtp.notConfigured}
              </p>
            )}
          </div>
        )}

        {submitAttempted && problems.length > 0 && (
          <ul className="space-y-0.5" role="alert">
            {problems.map((problem) => (
              <li key={problem} className={FIELD_ERROR_CLASS}>
                {receiversT.problems[problem]}
              </li>
            ))}
          </ul>
        )}

        {serverError && (
          <div className="p-3 bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 rounded-lg text-sm" role="alert">
            {serverError}
          </div>
        )}

        <div className="flex justify-end gap-3 pt-2">
          <button type="button" onClick={onClose} className={SECONDARY_BUTTON_CLASS}>
            {commonT.cancel}
          </button>
          <button type="submit" disabled={saveReceiver.isPending} className={PRIMARY_BUTTON_CLASS}>
            {saveReceiver.isPending ? t.saving : commonT.save}
          </button>
        </div>
      </form>
    </Modal>
  );
}
