"use client";

import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import type { ServiceAccountView } from "@/api/service-accounts";
import { Modal, toast } from "@/components/common";
import type { RoleInfo } from "@/app/admin/roles/roles";
import {
  SERVICE_ACCOUNT_DISPLAY_NAME_MAX_CHARS,
  SERVICE_ACCOUNT_NAME_MAX_LENGTH,
  SERVICE_ACCOUNT_NAME_MIN_LENGTH,
  displayNameProblem,
  serviceAccountFailure,
  serviceAccountNameProblem,
} from "@/lib/service-account-input";
import { createServiceAccountMutation, updateServiceAccountMutation } from "@/queries/service-accounts";
import type { CommonTranslations, ServiceAccountsTranslations } from "@/types/i18n";

/** 弹窗要做的事：新建一个服务账号，或修改已有账号的显示名、等级与启停 */
export type ServiceAccountModalTarget = { mode: "create" } | { mode: "edit"; account: ServiceAccountView };

interface ServiceAccountModalProps {
  target: ServiceAccountModalTarget;
  /** 可选的等级（后端下发的业务参数），从低到高；名称与说明沿用角色权限页的文案 */
  roles: RoleInfo[];
  onClose: () => void;
  t: ServiceAccountsTranslations;
  commonT: CommonTranslations;
}

const INPUT_CLASS =
  "w-full px-3 py-2 rounded-lg border border-[var(--border-color)] bg-[var(--background)] text-default text-sm focus:ring-2 focus:ring-primary outline-none disabled:opacity-60";

/** 文案里的数字占位换成前端校验用的同一组常量 */
function fillLimits(text: string, min: number, max: number): string {
  return text.replace("{min}", String(min)).replace("{max}", String(max));
}

/**
 * 新建 / 编辑服务账号的表单。
 *
 * 由外层按 target 整个重新挂载（key），所以表单初值直接取自 target，不需要在 effect 里同步。
 */
export function ServiceAccountModal({ target, roles, onClose, t, commonT }: ServiceAccountModalProps) {
  const editing = target.mode === "edit" ? target.account : null;
  const [name, setName] = useState(editing?.name ?? "");
  const [displayName, setDisplayName] = useState(editing?.displayName ?? "");
  const [role, setRole] = useState<number | null>(editing ? editing.role : (roles[0]?.level ?? null));
  const [enabled, setEnabled] = useState(editing?.enabled ?? true);
  // 没点过提交就不报错：一打开就满屏红字没有意义
  const [submitAttempted, setSubmitAttempted] = useState(false);
  const [serverError, setServerError] = useState("");

  // 成功后工厂会让列表重新取数，这里只管提示与关弹窗
  const create = useMutation(createServiceAccountMutation());
  const update = useMutation(updateServiceAccountMutation());
  const submitting = create.isPending || update.isPending;

  const nameProblem = editing ? null : serviceAccountNameProblem(name);
  const profileProblem = displayNameProblem(displayName);
  const nameError = nameProblem
    ? fillLimits(t.errors[nameProblem], SERVICE_ACCOUNT_NAME_MIN_LENGTH, SERVICE_ACCOUNT_NAME_MAX_LENGTH)
    : "";
  const profileError = profileProblem
    ? fillLimits(t.errors[profileProblem], 0, SERVICE_ACCOUNT_DISPLAY_NAME_MAX_CHARS)
    : "";

  const handleSubmit = (event: React.FormEvent) => {
    event.preventDefault();
    setSubmitAttempted(true);
    setServerError("");
    if (nameProblem || profileProblem || role === null) return;

    // 后端会去掉首尾空白；这里先去掉，返回的那一行与输入一致
    const trimmedDisplayName = displayName.trim();
    if (editing) {
      update.mutate(
        { accountId: editing.id, request: { displayName: trimmedDisplayName, role, enabled } },
        {
          onSuccess: () => {
            toast.success(t.updatedToast);
            onClose();
          },
          onError: (error) => setServerError(t.errors[serviceAccountFailure(error, "update")]),
        },
      );
      return;
    }
    create.mutate(
      { name, displayName: trimmedDisplayName, role },
      {
        onSuccess: () => {
          toast.success(t.createdToast);
          onClose();
        },
        onError: (error) => setServerError(t.errors[serviceAccountFailure(error, "create")]),
      },
    );
  };

  return (
    <Modal isOpen onClose={onClose} title={editing ? t.editTitle : t.createTitle} size="sm">
      <form onSubmit={handleSubmit} className="p-6 space-y-4" noValidate>
        <div>
          <label htmlFor="service-account-name" className="block text-sm font-medium text-muted mb-1">
            {t.name} *
          </label>
          <input
            id="service-account-name"
            type="text"
            value={name}
            onChange={(event) => setName(event.target.value)}
            disabled={editing !== null}
            maxLength={SERVICE_ACCOUNT_NAME_MAX_LENGTH}
            autoComplete="off"
            spellCheck={false}
            className={`${INPUT_CLASS} font-mono`}
          />
          {!editing && (
            <p className="text-xs text-muted mt-1">
              {fillLimits(t.nameHint, SERVICE_ACCOUNT_NAME_MIN_LENGTH, SERVICE_ACCOUNT_NAME_MAX_LENGTH)}
            </p>
          )}
          {submitAttempted && nameError && <p className="text-xs text-red-500 mt-1">{nameError}</p>}
        </div>

        <div>
          <label htmlFor="service-account-display-name" className="block text-sm font-medium text-muted mb-1">
            {t.displayName}
          </label>
          <input
            id="service-account-display-name"
            type="text"
            value={displayName}
            onChange={(event) => setDisplayName(event.target.value)}
            maxLength={SERVICE_ACCOUNT_DISPLAY_NAME_MAX_CHARS}
            placeholder={fillLimits(t.displayNameHint, 0, SERVICE_ACCOUNT_DISPLAY_NAME_MAX_CHARS)}
            autoComplete="off"
            className={INPUT_CLASS}
          />
          {submitAttempted && profileError && <p className="text-xs text-red-500 mt-1">{profileError}</p>}
        </div>

        <fieldset>
          <legend className="block text-sm font-medium text-muted mb-2">{t.role} *</legend>
          <div className="space-y-2">
            {roles.map((option) => (
              <label
                key={option.level}
                className={`flex items-start gap-3 p-3 rounded-lg border cursor-pointer transition-colors ${
                  role === option.level ? "border-primary bg-primary/5" : "border-[var(--border-color)] hover:bg-[var(--background)]"
                }`}
              >
                <input
                  type="radio"
                  name="service-account-role"
                  checked={role === option.level}
                  onChange={() => setRole(option.level)}
                  className="mt-1"
                />
                <span>
                  <span className={`block text-sm font-medium ${option.color}`}>{option.name}</span>
                  <span className="block text-xs text-muted">{option.description}</span>
                </span>
              </label>
            ))}
          </div>
          <p className="text-xs text-muted mt-2">{t.roleHint}</p>
        </fieldset>

        {editing && (
          <label className="flex items-start gap-3">
            <input type="checkbox" checked={enabled} onChange={(event) => setEnabled(event.target.checked)} className="mt-1" />
            <span>
              <span className="block text-sm text-default">{t.enabledLabel}</span>
              <span className="block text-xs text-muted">{t.enabledHint}</span>
            </span>
          </label>
        )}

        {serverError && (
          <div className="p-3 bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 rounded-lg text-sm" role="alert">
            {serverError}
          </div>
        )}

        <div className="flex gap-3 pt-2">
          <button
            type="button"
            onClick={onClose}
            className="flex-1 px-4 py-2 border border-[var(--border-color)] rounded-lg hover-bg text-sm text-default"
          >
            {commonT.cancel}
          </button>
          <button
            type="submit"
            disabled={submitting}
            className="flex-1 px-4 py-2 bg-primary hover:bg-primary-hover text-white rounded-lg disabled:opacity-50 text-sm"
          >
            {editing ? (submitting ? t.saving : commonT.save) : submitting ? t.creating : t.create}
          </button>
        </div>
      </form>
    </Modal>
  );
}
