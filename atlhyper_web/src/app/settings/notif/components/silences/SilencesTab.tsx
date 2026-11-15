"use client";

import { useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { BellOff, Plus } from "lucide-react";
import { ConfirmDialog, QueryView, toast } from "@/components/common";
import { useI18n } from "@/i18n/context";
import { notifyFailureText } from "@/lib/notify-failure";
import { MOCK_NOTIFY_SILENCES } from "@/mock/notify";
import { expireNotifySilenceMutation, notifySilencesQuery } from "@/queries/notify";
import type { NotifySilence, NotifySilenceState } from "@/types/notify";
import { EmptyCard } from "../shared/EmptyCard";
import type { NotifyTabProps } from "../shared/format";
import { CARD_CLASS, PRIMARY_BUTTON_CLASS } from "../shared/form-styles";
import { SegmentedFilter } from "../shared/SegmentedFilter";
import { useNotifyClusterIds } from "../shared/useNotifyClusterIds";
import { SilenceModal } from "./SilenceModal";
import { SilencesTable } from "./SilencesTable";

const SILENCE_STATES: NotifySilenceState[] = ["active", "pending", "expired"];

/** 「静默」页签：按状态看静默、新建、提前结束 */
export function SilencesTab({ isDemo }: NotifyTabProps) {
  const { t } = useI18n();
  const notifyT = t.notifications;
  const silencesT = notifyT.silences;
  const clusterIds = useNotifyClusterIds(isDemo);

  const [state, setState] = useState<NotifySilenceState>("active");
  const silencesQuery = useQuery(notifySilencesQuery(state, !isDemo));
  // 成功后工厂让各状态的列表重新取数，这里只管提示与关弹窗
  const expireSilence = useMutation(expireNotifySilenceMutation());

  const [creating, setCreating] = useState(false);
  const [expireTarget, setExpireTarget] = useState<NotifySilence | null>(null);

  const handleExpireConfirm = () => {
    if (!expireTarget) return;
    expireSilence.mutate(expireTarget.id, {
      onSuccess: () => {
        toast.success(silencesT.expiredToast);
        setExpireTarget(null);
      },
      onError: (error) => {
        setExpireTarget(null);
        toast.error(notifyFailureText(error, notifyT.errors));
      },
    });
  };

  const renderSilences = (silences: NotifySilence[]) =>
    silences.length === 0 ? (
      <EmptyCard icon={BellOff} title={silencesT.empty[state]} />
    ) : (
      <div className={CARD_CLASS}>
        <SilencesTable
          silences={silences}
          readOnly={isDemo}
          onExpire={setExpireTarget}
          t={notifyT}
          actionLabel={t.common.action}
        />
      </div>
    );

  return (
    <div className="space-y-4">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <SegmentedFilter
          options={SILENCE_STATES.map((value) => ({ value, label: silencesT.stateLabel[value] }))}
          value={state}
          onChange={setState}
        />
        <button
          type="button"
          onClick={() => setCreating(true)}
          // 演示模式下写操作不可用
          disabled={isDemo}
          className={`${PRIMARY_BUTTON_CLASS} flex-shrink-0 self-end sm:self-auto`}
        >
          <Plus className="w-4 h-4" />
          {silencesT.add}
        </button>
      </div>

      {isDemo ? (
        renderSilences(MOCK_NOTIFY_SILENCES.filter((silence) => silence.state === state))
      ) : (
        <QueryView query={silencesQuery}>{renderSilences}</QueryView>
      )}

      {creating && (
        <SilenceModal clusterIds={clusterIds} onClose={() => setCreating(false)} t={notifyT} commonT={t.common} />
      )}

      <ConfirmDialog
        isOpen={expireTarget !== null}
        onClose={() => setExpireTarget(null)}
        onConfirm={handleExpireConfirm}
        title={silencesT.expireConfirmTitle}
        message={silencesT.expireConfirmMessage}
        confirmText={silencesT.expire}
        cancelText={t.common.cancel}
        loading={expireSilence.isPending}
        variant="warning"
      />
    </div>
  );
}
