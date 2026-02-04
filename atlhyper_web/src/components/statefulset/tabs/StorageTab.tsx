"use client";

import type { StatefulSetDetail } from "@/api/workload";

interface StorageTabProps {
  detail: StatefulSetDetail;
}

export function StorageTab({ detail }: StorageTabProps) {
  const vcts = detail.spec.volumeClaimTemplates || [];
  const volumes = detail.template?.volumes || [];

  return (
    <div className="space-y-6">
      {/* Volume Claim Templates */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">
          Volume Claim Templates ({vcts.length})
        </h3>
        {vcts.length === 0 ? (
          <div className="text-center py-4 text-muted bg-[var(--background)] rounded-lg">
            无 PVC 模板
          </div>
        ) : (
          <div className="space-y-3">
            {vcts.map((vct, i) => (
              <div key={i} className="bg-[var(--background)] rounded-lg p-4">
                <div className="flex items-center justify-between mb-2">
                  <h4 className="font-medium text-default">{vct.name}</h4>
                  {vct.storage && (
                    <span className="px-2 py-1 bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400 text-xs rounded">
                      {vct.storage}
                    </span>
                  )}
                </div>
                <div className="grid grid-cols-2 gap-3 text-sm">
                  {vct.storageClass && (
                    <div>
                      <span className="text-muted">StorageClass: </span>
                      <span className="text-default">{vct.storageClass}</span>
                    </div>
                  )}
                  {vct.accessModes && vct.accessModes.length > 0 && (
                    <div>
                      <span className="text-muted">Access Modes: </span>
                      <span className="text-default">{vct.accessModes.join(", ")}</span>
                    </div>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Volumes */}
      {volumes.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">
            Volumes ({volumes.length})
          </h3>
          <div className="space-y-2">
            {volumes.map((v, i) => (
              <div key={i} className="flex items-center justify-between p-3 bg-[var(--background)] rounded-lg">
                <div>
                  <span className="font-medium text-default">{v.name}</span>
                  <span className="text-xs text-muted ml-2">({v.type})</span>
                </div>
                {v.source && (
                  <span className="text-sm text-muted truncate max-w-[200px]">{v.source}</span>
                )}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
