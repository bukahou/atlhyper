"use client";

import { Save, X, Edit2 } from "lucide-react";
import { StatusBadge } from "@/components/common";
import type { DeploymentContainer } from "@/types/cluster";
import type { useI18n } from "@/i18n/context";
import { ProbesDisplay } from "./ProbesDisplay";

interface ContainersTabProps {
  containers: DeploymentContainer[];
  editingImage: { containerName: string; oldImage: string; newImage: string } | null;
  onEditImage: (containerName: string, oldImage: string) => void;
  onImageChange: (newImage: string) => void;
  onCancelEdit: () => void;
  onSaveImage: () => void;
  t: ReturnType<typeof useI18n>["t"];
}

export function ContainersTab({
  containers,
  editingImage,
  onEditImage,
  onImageChange,
  onCancelEdit,
  onSaveImage,
  t,
}: ContainersTabProps) {
  if (!containers || containers.length === 0) {
    return <div className="text-center py-8 text-muted">{t.deployment.noContainers}</div>;
  }

  return (
    <div className="space-y-4">
      {containers.map((container) => (
        <div key={container.name} className="bg-[var(--background)] rounded-lg p-4">
          <div className="flex items-center justify-between mb-3">
            <h4 className="font-medium text-default">{container.name}</h4>
            {container.imagePullPolicy && <StatusBadge status={container.imagePullPolicy} type="info" />}
          </div>

          {/* 镜像 */}
          <div className="mb-4">
            <div className="text-xs text-muted mb-1">{t.deployment.image}</div>
            {editingImage?.containerName === container.name ? (
              <div className="flex items-center gap-2">
                <input
                  type="text"
                  value={editingImage.newImage}
                  onChange={(e) => onImageChange(e.target.value)}
                  className="flex-1 px-3 py-2 bg-card border border-[var(--border-color)] rounded-lg text-sm"
                />
                <button
                  onClick={onSaveImage}
                  disabled={editingImage.newImage === editingImage.oldImage}
                  className="p-2 bg-primary text-white rounded-lg disabled:opacity-50"
                >
                  <Save className="w-4 h-4" />
                </button>
                <button onClick={onCancelEdit} className="p-2 hover-bg rounded-lg">
                  <X className="w-4 h-4 text-muted" />
                </button>
              </div>
            ) : (
              <div className="flex items-center gap-2">
                <span className="text-sm font-mono text-default break-all">{container.image}</span>
                <button onClick={() => onEditImage(container.name, container.image)} className="p-1.5 hover-bg rounded-lg shrink-0">
                  <Edit2 className="w-3.5 h-3.5 text-muted" />
                </button>
              </div>
            )}
          </div>

          {/* 端口 */}
          {container.ports && container.ports.length > 0 && (
            <div className="mb-3">
              <div className="text-xs text-muted mb-1">{t.deployment.ports}</div>
              <div className="flex flex-wrap gap-2">
                {container.ports.map((port, i) => (
                  <span key={i} className="px-2 py-1 bg-card rounded text-xs font-mono">
                    {port.containerPort}/{port.protocol || "TCP"}
                    {port.name && ` (${port.name})`}
                  </span>
                ))}
              </div>
            </div>
          )}

          {/* 资源限制 */}
          {(container.requests || container.limits) && (
            <div className="grid grid-cols-2 gap-4 mb-3">
              {container.requests && Object.keys(container.requests).length > 0 && (
                <div>
                  <div className="text-xs text-muted mb-1">Requests</div>
                  <div className="space-y-1">
                    {Object.entries(container.requests).map(([k, v]) => (
                      <div key={k} className="text-xs">
                        <span className="text-muted">{k}:</span> <span className="font-mono">{v}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
              {container.limits && Object.keys(container.limits).length > 0 && (
                <div>
                  <div className="text-xs text-muted mb-1">Limits</div>
                  <div className="space-y-1">
                    {Object.entries(container.limits).map(([k, v]) => (
                      <div key={k} className="text-xs">
                        <span className="text-muted">{k}:</span> <span className="font-mono">{v}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}

          {/* 探针 */}
          <ProbesDisplay
            liveness={container.livenessProbe}
            readiness={container.readinessProbe}
            startup={container.startupProbe}
            t={t}
          />

          {/* 环境变量 */}
          {container.envs && container.envs.length > 0 && (
            <details className="mt-3">
              <summary className="text-xs text-muted cursor-pointer">{t.deployment.envVars} ({container.envs.length})</summary>
              <div className="mt-2 space-y-1">
                {container.envs.map((env, i) => (
                  <div key={i} className="text-xs font-mono bg-card px-2 py-1 rounded">
                    <span className="text-primary">{env.name}</span>
                    <span className="text-muted">=</span>
                    <span className="text-default">{env.value || env.valueFrom || '""'}</span>
                  </div>
                ))}
              </div>
            </details>
          )}

          {/* 挂载 */}
          {container.volumeMounts && container.volumeMounts.length > 0 && (
            <details className="mt-3">
              <summary className="text-xs text-muted cursor-pointer">{t.deployment.volumeMounts} ({container.volumeMounts.length})</summary>
              <div className="mt-2 space-y-1">
                {container.volumeMounts.map((vm, i) => (
                  <div key={i} className="text-xs font-mono bg-card px-2 py-1 rounded flex justify-between">
                    <span className="text-primary">{vm.name}</span>
                    <span className="text-default">{vm.mountPath}</span>
                    {vm.readOnly && <span className="text-muted">(ro)</span>}
                  </div>
                ))}
              </div>
            </details>
          )}
        </div>
      ))}
    </div>
  );
}
