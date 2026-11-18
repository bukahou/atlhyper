"use client";

import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { Modal, toast } from "@/components/common";
import { notifyFailureText } from "@/lib/notify-failure";
import {
  newSilenceDraft,
  SILENCE_COMMENT_MAX_CHARS,
  silenceBodyFromDraft,
  silenceDraftProblems,
  type NotifySilenceDraft,
  type NotifySilenceProblem,
} from "@/lib/notify-silence-input";
import { createNotifySilenceMutation } from "@/queries/notify";
import type { CommonTranslations, NotificationsTranslations } from "@/types/i18n";
import {
  FIELD_ERROR_CLASS,
  HINT_CLASS,
  INPUT_CLASS,
  LABEL_CLASS,
  PRIMARY_BUTTON_CLASS,
  SECONDARY_BUTTON_CLASS,
} from "../shared/form-styles";

interface SilenceModalProps {
  /** 可选的集群（已接入的；演示模式下是演示集群） */
  clusterIds: string[];
  onClose: () => void;
  t: NotificationsTranslations;
  commonT: CommonTranslations;
}

const NO_PROBLEMS: NotifySilenceProblem[] = [];

/** 新建静默的表单。静默不能修改：建错了就提前结束，再建一条 */
export function SilenceModal({ clusterIds, onClose, t, commonT }: SilenceModalProps) {
  const silencesT = t.silences;
  const [draft, setDraft] = useState<NotifySilenceDraft>(() => newSilenceDraft(new Date()));
  // 校验要拿「现在」比（结束时间必须晚于现在），所以在点提交的那一刻算，不在渲染时算
  const [problems, setProblems] = useState<NotifySilenceProblem[]>(NO_PROBLEMS);
  const [serverError, setServerError] = useState("");

  // 成功后工厂会让列表重新取数，这里只管提示与关弹窗
  const createSilence = useMutation(createNotifySilenceMutation());
  const updateDraft = (changes: Partial<NotifySilenceDraft>) => setDraft((current) => ({ ...current, ...changes }));

  const silencesEverything =
    draft.clusterId.trim() === "" && draft.namespace.trim() === "" && draft.subject.trim() === "";

  const handleSubmit = (event: React.FormEvent) => {
    event.preventDefault();
    setServerError("");
    const found = silenceDraftProblems(draft, new Date());
    setProblems(found);
    const silence = silenceBodyFromDraft(draft);
    if (found.length > 0 || !silence) return;

    createSilence.mutate(silence, {
      onSuccess: () => {
        toast.success(silencesT.createdToast);
        onClose();
      },
      onError: (error) => setServerError(notifyFailureText(error, t.errors)),
    });
  };

  return (
    <Modal isOpen onClose={onClose} title={silencesT.createTitle} size="md">
      <form onSubmit={handleSubmit} className="p-6 space-y-4" noValidate>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label htmlFor="notify-silence-cluster" className={LABEL_CLASS}>
              {silencesT.cluster}
            </label>
            <input
              id="notify-silence-cluster"
              type="text"
              list="notify-silence-cluster-options"
              value={draft.clusterId}
              onChange={(event) => updateDraft({ clusterId: event.target.value })}
              placeholder={silencesT.allClusters}
              autoComplete="off"
              spellCheck={false}
              className={`${INPUT_CLASS} font-mono`}
            />
            {/* 可以从已接入的集群里选，也可以手填：要静默的往往正是此刻离线的集群 */}
            <datalist id="notify-silence-cluster-options">
              {clusterIds.map((clusterId) => (
                <option key={clusterId} value={clusterId} />
              ))}
            </datalist>
          </div>
          <div>
            <label htmlFor="notify-silence-namespace" className={LABEL_CLASS}>
              {silencesT.namespace}
            </label>
            <input
              id="notify-silence-namespace"
              type="text"
              value={draft.namespace}
              onChange={(event) => updateDraft({ namespace: event.target.value })}
              placeholder={silencesT.namespacePlaceholder}
              autoComplete="off"
              spellCheck={false}
              className={`${INPUT_CLASS} font-mono`}
            />
          </div>
        </div>

        <div>
          <label htmlFor="notify-silence-subject" className={LABEL_CLASS}>
            {silencesT.subject}
          </label>
          <input
            id="notify-silence-subject"
            type="text"
            value={draft.subject}
            onChange={(event) => updateDraft({ subject: event.target.value })}
            placeholder={silencesT.subjectPlaceholder}
            autoComplete="off"
            spellCheck={false}
            className={`${INPUT_CLASS} font-mono`}
          />
          <p className={HINT_CLASS}>{silencesT.subjectHint}</p>
        </div>

        <p className={HINT_CLASS}>{silencesT.heartbeatHint}</p>
        {silencesEverything && (
          <p className="text-xs text-amber-600 dark:text-amber-400" role="status">
            {silencesT.allAlertsWarning}
          </p>
        )}

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label htmlFor="notify-silence-starts-at" className={LABEL_CLASS}>
              {silencesT.startsAt} *
            </label>
            <input
              id="notify-silence-starts-at"
              type="datetime-local"
              value={draft.startsAtLocal}
              onChange={(event) => updateDraft({ startsAtLocal: event.target.value })}
              className={INPUT_CLASS}
            />
          </div>
          <div>
            <label htmlFor="notify-silence-ends-at" className={LABEL_CLASS}>
              {silencesT.endsAt} *
            </label>
            <input
              id="notify-silence-ends-at"
              type="datetime-local"
              value={draft.endsAtLocal}
              onChange={(event) => updateDraft({ endsAtLocal: event.target.value })}
              className={INPUT_CLASS}
            />
          </div>
        </div>

        <div>
          <label htmlFor="notify-silence-comment" className={LABEL_CLASS}>
            {silencesT.comment} *
          </label>
          <textarea
            id="notify-silence-comment"
            value={draft.comment}
            onChange={(event) => updateDraft({ comment: event.target.value })}
            placeholder={silencesT.commentPlaceholder}
            rows={2}
            maxLength={SILENCE_COMMENT_MAX_CHARS}
            className={INPUT_CLASS}
          />
        </div>

        {problems.length > 0 && (
          <ul className="space-y-0.5" role="alert">
            {problems.map((problem) => (
              <li key={problem} className={FIELD_ERROR_CLASS}>
                {silencesT.problems[problem]}
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
          <button type="submit" disabled={createSilence.isPending} className={PRIMARY_BUTTON_CLASS}>
            {createSilence.isPending ? t.saving : commonT.save}
          </button>
        </div>
      </form>
    </Modal>
  );
}
