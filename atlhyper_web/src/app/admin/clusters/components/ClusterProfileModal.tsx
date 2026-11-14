"use client";

import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { Modal, toast } from "@/components/common";
import {
  CLUSTER_DESCRIPTION_MAX_CHARS,
  CLUSTER_ID_MAX_LENGTH,
  CLUSTER_NAME_MAX_CHARS,
  clusterIdProblem,
  clusterProfileProblem,
  clusterWriteFailure,
} from "@/lib/cluster-registry-input";
import { registerClusterMutation, updateClusterProfileMutation } from "@/queries/cluster-registry";
import type { ClusterRegistryTranslations, CommonTranslations } from "@/types/i18n";

/** 弹窗要做的事：登记一个新集群，或修改已登记集群的显示名与描述 */
export type ClusterProfileTarget =
  | {
      mode: "register";
      /** 预填的集群 ID；从「未登记」行点进来时带上该行的 ID */
      clusterId: string;
      /** 预填的 ID 来自正在上报的集群：必须逐字一致，不让改 */
      clusterIdLocked: boolean;
    }
  | { mode: "edit"; clusterId: string; name: string; description: string };

interface ClusterProfileModalProps {
  target: ClusterProfileTarget;
  onClose: () => void;
  t: ClusterRegistryTranslations;
  commonT: CommonTranslations;
}

const INPUT_CLASS =
  "w-full px-3 py-2 rounded-lg border border-[var(--border-color)] bg-[var(--background)] text-default text-sm focus:ring-2 focus:ring-primary outline-none disabled:opacity-60";

/**
 * 登记 / 编辑集群的表单。
 *
 * 由外层按 target 整个重新挂载（key），所以表单初值直接取自 target，不需要在 effect 里同步。
 */
export function ClusterProfileModal({ target, onClose, t, commonT }: ClusterProfileModalProps) {
  const isRegistering = target.mode === "register";
  const [clusterId, setClusterId] = useState(target.clusterId);
  const [name, setName] = useState(target.mode === "edit" ? target.name : "");
  const [description, setDescription] = useState(target.mode === "edit" ? target.description : "");
  // 没点过提交就不报错：一打开就满屏红字没有意义
  const [submitAttempted, setSubmitAttempted] = useState(false);
  const [serverError, setServerError] = useState("");

  // 成功后工厂会让列表重新取数，这里只管提示与关弹窗
  const register = useMutation(registerClusterMutation());
  const updateProfile = useMutation(updateClusterProfileMutation());
  const submitting = register.isPending || updateProfile.isPending;

  const idProblem = isRegistering ? clusterIdProblem(clusterId) : null;
  const profileProblem = clusterProfileProblem(name, description);
  const idError = idProblem === "length" ? t.errors.idLength : idProblem === "charset" ? t.errors.idCharset : "";
  const profileError = profileProblem ? t.errors[profileProblem] : "";

  const handleSubmit = (event: React.FormEvent) => {
    event.preventDefault();
    setSubmitAttempted(true);
    setServerError("");
    if (idProblem || profileProblem) return;

    // 后端会去掉首尾空白；这里先去掉，返回的那一行与输入一致
    const profile = { name: name.trim(), description: description.trim() };
    const onError = (error: unknown) => setServerError(t.errors[clusterWriteFailure(error)]);
    if (isRegistering) {
      register.mutate(
        { clusterId, ...profile },
        {
          onSuccess: () => {
            toast.success(t.registeredToast);
            onClose();
          },
          onError,
        },
      );
    } else {
      updateProfile.mutate(
        { clusterId: target.clusterId, ...profile },
        {
          onSuccess: () => {
            toast.success(t.updatedToast);
            onClose();
          },
          onError,
        },
      );
    }
  };

  return (
    <Modal isOpen onClose={onClose} title={isRegistering ? t.registerTitle : t.editTitle} size="sm">
      <form onSubmit={handleSubmit} className="p-6 space-y-4" noValidate>
        <div>
          <label htmlFor="cluster-registry-id" className="block text-sm font-medium text-muted mb-1">
            {t.clusterId} *
          </label>
          <input
            id="cluster-registry-id"
            type="text"
            value={clusterId}
            onChange={(event) => setClusterId(event.target.value)}
            disabled={!isRegistering || target.clusterIdLocked}
            maxLength={CLUSTER_ID_MAX_LENGTH}
            autoComplete="off"
            spellCheck={false}
            className={`${INPUT_CLASS} font-mono`}
          />
          {isRegistering && (
            <p className="text-xs text-muted mt-1">
              {t.clusterIdFormatHint} {t.clusterIdMatchHint}
            </p>
          )}
          {submitAttempted && idError && <p className="text-xs text-red-500 mt-1">{idError}</p>}
        </div>

        <div>
          <label htmlFor="cluster-registry-name" className="block text-sm font-medium text-muted mb-1">
            {t.displayName} *
          </label>
          <input
            id="cluster-registry-name"
            type="text"
            value={name}
            onChange={(event) => setName(event.target.value)}
            maxLength={CLUSTER_NAME_MAX_CHARS}
            placeholder={t.displayNameHint}
            autoComplete="off"
            className={INPUT_CLASS}
          />
        </div>

        <div>
          <label htmlFor="cluster-registry-description" className="block text-sm font-medium text-muted mb-1">
            {t.description}
          </label>
          <input
            id="cluster-registry-description"
            type="text"
            value={description}
            onChange={(event) => setDescription(event.target.value)}
            maxLength={CLUSTER_DESCRIPTION_MAX_CHARS}
            placeholder={t.descriptionHint}
            autoComplete="off"
            className={INPUT_CLASS}
          />
        </div>

        {submitAttempted && profileError && <p className="text-xs text-red-500">{profileError}</p>}

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
            {isRegistering ? (submitting ? t.registering : t.register) : submitting ? t.saving : commonT.save}
          </button>
        </div>
      </form>
    </Modal>
  );
}
