"use client";

import { Pause, Minus, Plus, Save, X, Edit2, CheckCircle, XCircle, AlertTriangle } from "lucide-react";
import { StatusBadge } from "@/components/common";
import type { DeploymentDetail } from "@/types/cluster";
import type { useI18n } from "@/i18n/context";
import { RolloutBadge } from "./RolloutBadge";
import { InfoCard } from "./InfoCard";

interface OverviewTabProps {
  detail: DeploymentDetail;
  editingReplicas: number | null;
  onStartEdit: () => void;
  onReplicasChange: (n: number) => void;
  onCancelEdit: () => void;
  onSave: () => void;
  t: ReturnType<typeof useI18n>["t"];
}

export function OverviewTab({
  detail,
  editingReplicas,
  onStartEdit,
  onReplicasChange,
  onCancelEdit,
  onSave,
  t,
}: OverviewTabProps) {
  const isEditing = editingReplicas !== null;
  const replicas = editingReplicas ?? detail.replicas;

  return (
    <div className="space-y-6">
      {/* Rollout 状态徽标 */}
      {detail.rollout && (
        <div className="flex items-center gap-2 flex-wrap">
          {detail.rollout.badges?.map((badge, i) => (
            <RolloutBadge key={i} badge={badge} />
          ))}
          {detail.paused && (
            <span className="inline-flex items-center gap-1 px-2 py-1 bg-yellow-100 dark:bg-yellow-900/30 text-yellow-600 dark:text-yellow-400 text-xs rounded">
              <Pause className="w-3 h-3" /> {t.deployment.paused}
            </span>
          )}
        </div>
      )}

      {/* 副本状态卡片 */}
      <div className="grid grid-cols-4 gap-4">
        <div className="bg-[var(--background)] rounded-lg p-4 text-center">
          <div className="text-2xl font-bold text-default">{detail.replicas}</div>
          <div className="text-xs text-muted mt-1">{t.deployment.desired}</div>
        </div>
        <div className="bg-[var(--background)] rounded-lg p-4 text-center">
          <div className="text-2xl font-bold text-green-500">{detail.ready}</div>
          <div className="text-xs text-muted mt-1">{t.deployment.ready}</div>
        </div>
        <div className="bg-[var(--background)] rounded-lg p-4 text-center">
          <div className="text-2xl font-bold text-blue-500">{detail.available}</div>
          <div className="text-xs text-muted mt-1">{t.deployment.available}</div>
        </div>
        <div className="bg-[var(--background)] rounded-lg p-4 text-center">
          <div className="text-2xl font-bold text-orange-500">{detail.updated}</div>
          <div className="text-xs text-muted mt-1">{t.deployment.updated}</div>
        </div>
      </div>

      {/* 扩缩容 */}
      <div className="bg-[var(--background)] rounded-lg p-4">
        <h3 className="text-sm font-semibold text-default mb-3">{t.deployment.adjustReplicas}</h3>
        {isEditing ? (
          <div className="flex items-center gap-4">
            <button
              onClick={() => onReplicasChange(Math.max(0, replicas - 1))}
              className="p-2 bg-card hover:bg-[var(--border-color)] rounded-lg"
            >
              <Minus className="w-5 h-5" />
            </button>
            <input
              type="number"
              min="0"
              value={replicas}
              onChange={(e) => onReplicasChange(Math.max(0, parseInt(e.target.value) || 0))}
              className="w-20 px-3 py-2 text-center text-xl font-bold bg-card border border-[var(--border-color)] rounded-lg"
            />
            <button
              onClick={() => onReplicasChange(replicas + 1)}
              className="p-2 bg-card hover:bg-[var(--border-color)] rounded-lg"
            >
              <Plus className="w-5 h-5" />
            </button>
            <button
              onClick={onSave}
              disabled={replicas === detail.replicas}
              className="px-3 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 disabled:opacity-50"
            >
              <Save className="w-4 h-4" />
            </button>
            <button onClick={onCancelEdit} className="px-3 py-2 hover-bg rounded-lg">
              <X className="w-4 h-4 text-muted" />
            </button>
          </div>
        ) : (
          <button
            onClick={onStartEdit}
            className="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 flex items-center gap-2"
          >
            <Edit2 className="w-4 h-4" />
            {t.deployment.adjustReplicas}
          </button>
        )}
      </div>

      {/* 基本信息 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">{t.deployment.basicInfo}</h3>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
          <InfoCard label={t.common.name} value={detail.name} />
          <InfoCard label={t.common.namespace} value={detail.namespace} />
          <InfoCard label={t.deployment.strategy} value={detail.strategy || "-"} />
          <InfoCard label={t.deployment.age} value={detail.age || "-"} />
          <InfoCard label={t.deployment.selector} value={detail.selector || "-"} />
          <InfoCard label={t.common.createdAt} value={detail.createdAt ? new Date(detail.createdAt).toLocaleString() : "-"} />
        </div>
      </div>

      {/* Conditions */}
      {detail.conditions && detail.conditions.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">{t.deployment.conditions}</h3>
          <div className="space-y-2">
            {detail.conditions.map((cond, i) => (
              <div key={i} className="bg-[var(--background)] rounded-lg p-3 flex items-center justify-between">
                <div className="flex items-center gap-2">
                  {cond.status === "True" ? (
                    <CheckCircle className="w-4 h-4 text-green-500" />
                  ) : cond.status === "False" ? (
                    <XCircle className="w-4 h-4 text-gray-400" />
                  ) : (
                    <AlertTriangle className="w-4 h-4 text-yellow-500" />
                  )}
                  <span className="font-medium text-default">{cond.type}</span>
                  {cond.reason && <span className="text-sm text-muted">({cond.reason})</span>}
                </div>
                <StatusBadge status={cond.status} type={cond.status === "True" ? "success" : "warning"} />
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
