"use client";

import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { FlaskConical } from "lucide-react";
import { StatusBadge } from "@/components/common";
import { notifyFailureText } from "@/lib/notify-failure";
import { matchNotifyRulesMutation } from "@/queries/notify-routing";
import type { NotificationsTranslations } from "@/types/i18n";
import {
  NOTIFY_ALERT_SOURCES,
  NOTIFY_SEVERITIES,
  type NotifyAlertSource,
  type NotifyRuleMatchResult,
  type NotifySeverity,
} from "@/types/notify";
import { INPUT_CLASS, LABEL_CLASS, PRIMARY_BUTTON_CLASS, SELECT_CLASS } from "../shared/form-styles";

interface RuleMatchPanelProps {
  /** 可选的集群（也可以手填） */
  clusterIds: string[];
  /** 演示模式：试匹配要问后端，不可用 */
  readOnly: boolean;
  t: NotificationsTranslations;
}

function MatchResult({ result, t }: { result: NotifyRuleMatchResult; t: NotificationsTranslations }) {
  const rulesT = t.rules;
  const receivers = result.receivers ?? [];
  return (
    <div className="mt-4 p-4 rounded-lg bg-[var(--background)] border border-[var(--border-color)] space-y-3" role="status">
      <div className="text-sm">
        <span className="text-muted">{rulesT.matchedRule}: </span>
        {result.matchedRule ? (
          <span className="font-medium text-default">{result.matchedRule.name}</span>
        ) : (
          <span className="text-default">{rulesT.matchedDefault}</span>
        )}
      </div>
      {result.belowMinSeverity && (
        <p className="text-sm text-amber-600 dark:text-amber-400">{rulesT.matchBelowMinSeverity}</p>
      )}
      <div className="text-sm">
        <span className="text-muted">{rulesT.matchReceivers}: </span>
        {receivers.length === 0 ? (
          <span className="text-amber-600 dark:text-amber-400">{rulesT.matchNoReceivers}</span>
        ) : (
          <span className="inline-flex flex-wrap gap-2 align-middle">
            {receivers.map((receiver) => (
              <span key={receiver.id} className="inline-flex items-center gap-1.5">
                <span className="text-default">{receiver.name}</span>
                <span className="text-xs text-muted">
                  {receiver.type === "slack" ? t.receivers.typeSlack : t.receivers.typeEmail}
                </span>
                {!receiver.enabled && <StatusBadge status={rulesT.matchReceiverDisabled} type="warning" />}
              </span>
            ))}
          </span>
        )}
      </div>
    </div>
  );
}

/** 试匹配：假想一条告警（集群 / 来源 / 严重度），问后端它会命中哪条规则、发给谁 */
export function RuleMatchPanel({ clusterIds, readOnly, t }: RuleMatchPanelProps) {
  const rulesT = t.rules;
  const [clusterId, setClusterId] = useState("");
  const [source, setSource] = useState<NotifyAlertSource>("incident");
  const [severity, setSeverity] = useState<NotifySeverity>("high");
  const matchRules = useMutation(matchNotifyRulesMutation());

  const handleSubmit = (event: React.FormEvent) => {
    event.preventDefault();
    matchRules.mutate({ clusterId: clusterId.trim(), source, severity });
  };

  return (
    <section className="bg-card rounded-xl border border-[var(--border-color)] p-4 sm:p-6">
      <h3 className="flex items-center gap-2 text-sm font-semibold text-default">
        <FlaskConical className="w-4 h-4 text-muted" />
        {rulesT.matchTitle}
      </h3>
      <p className="text-xs text-muted mt-1 mb-4">{rulesT.matchHint}</p>

      <form onSubmit={handleSubmit} className="grid grid-cols-1 sm:grid-cols-4 gap-3 items-end">
        <div>
          <label htmlFor="notify-match-cluster" className={LABEL_CLASS}>
            {rulesT.matchCluster}
          </label>
          <input
            id="notify-match-cluster"
            type="text"
            list="notify-match-cluster-options"
            value={clusterId}
            onChange={(event) => setClusterId(event.target.value)}
            placeholder={rulesT.matchClusterPlaceholder}
            autoComplete="off"
            spellCheck={false}
            disabled={readOnly}
            className={`${INPUT_CLASS} font-mono`}
          />
          <datalist id="notify-match-cluster-options">
            {clusterIds.map((knownClusterId) => (
              <option key={knownClusterId} value={knownClusterId} />
            ))}
          </datalist>
        </div>
        <div>
          <label htmlFor="notify-match-source" className={LABEL_CLASS}>
            {rulesT.matchSource}
          </label>
          <select
            id="notify-match-source"
            value={source}
            onChange={(event) => setSource(event.target.value as NotifyAlertSource)}
            disabled={readOnly}
            className={SELECT_CLASS}
          >
            {NOTIFY_ALERT_SOURCES.map((option) => (
              <option key={option} value={option}>
                {t.source[option]}
              </option>
            ))}
          </select>
        </div>
        <div>
          <label htmlFor="notify-match-severity" className={LABEL_CLASS}>
            {rulesT.matchSeverity}
          </label>
          <select
            id="notify-match-severity"
            value={severity}
            onChange={(event) => setSeverity(event.target.value as NotifySeverity)}
            disabled={readOnly}
            className={SELECT_CLASS}
          >
            {NOTIFY_SEVERITIES.map((option) => (
              <option key={option} value={option}>
                {t.severity[option]}
              </option>
            ))}
          </select>
        </div>
        <button
          type="submit"
          disabled={readOnly || matchRules.isPending || clusterId.trim() === ""}
          className={`${PRIMARY_BUTTON_CLASS} justify-center`}
        >
          {rulesT.matchRun}
        </button>
      </form>

      {matchRules.isError && (
        <p className="mt-4 text-sm text-red-600 dark:text-red-400" role="alert">
          {notifyFailureText(matchRules.error, t.errors)}
        </p>
      )}
      {matchRules.data && !matchRules.isPending && <MatchResult result={matchRules.data} t={t} />}
    </section>
  );
}
