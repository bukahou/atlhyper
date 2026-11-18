"use client";

import { useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { Plus, Server } from "lucide-react";
import type { ClusterRegistryItem } from "@/api/cluster-registry";
import { ConfirmDialog, DemoBanner, LoadingSpinner, PageHeader, QueryView, toast } from "@/components/common";
import { Layout } from "@/components/layout/Layout";
import { useCan } from "@/hooks/useCan";
import { useSessionReady } from "@/hooks/useSessionReady";
import { useI18n } from "@/i18n/context";
import { clusterWriteFailure } from "@/lib/cluster-registry-input";
import { MOCK_CLUSTER_REGISTRY } from "@/mock/cluster-registry";
import {
  clusterRegistryQuery,
  issueClusterTokenMutation,
  unregisterClusterMutation,
} from "@/queries/cluster-registry";
import { REFRESH } from "@/queries/refresh";
import { ClusterProfileModal, type ClusterProfileTarget } from "./components/ClusterProfileModal";
import { ClusterRegistryTable } from "./components/ClusterRegistryTable";
import { ClusterTokenDialog, type IssuedClusterToken } from "./components/ClusterTokenDialog";

export default function ClusterRegistryPage() {
  const { t } = useI18n();
  const registryT = t.clusterRegistry;
  // 集群管理是纯管理功能页：权限加载完才决定显示真实数据还是演示数据；无 clusters.manage 时整页演示数据
  const sessionReady = useSessionReady();
  const canManageClusters = useCan()("clusters.manage");
  const isDemo = sessionReady && !canManageClusters;

  const registryQuery = useQuery(clusterRegistryQuery(canManageClusters));
  // 写操作成功后工厂让列表重新取数，这里只管提示与关弹窗
  const issueToken = useMutation(issueClusterTokenMutation());
  const unregister = useMutation(unregisterClusterMutation());

  const [profileTarget, setProfileTarget] = useState<ClusterProfileTarget | null>(null);
  const [rotateTarget, setRotateTarget] = useState<ClusterRegistryItem | null>(null);
  const [unregisterTarget, setUnregisterTarget] = useState<ClusterRegistryItem | null>(null);
  // 刚签发的令牌明文只放在这里：不进查询缓存、不落 localStorage、不打日志；关掉弹窗即置空丢弃
  const [issuedToken, setIssuedToken] = useState<IssuedClusterToken | null>(null);

  const reportWriteFailure = (error: unknown) => toast.error(registryT.errors[clusterWriteFailure(error)]);

  const issueTokenFor = (cluster: ClusterRegistryItem) => {
    issueToken.mutate(
      {
        clusterId: cluster.clusterId,
        // 明文由工厂直接交到这里，不经过 mutation 的返回值（那会留在缓存里）
        deliverToken: (token) => setIssuedToken({ clusterId: cluster.clusterId, clusterName: cluster.name, token }),
      },
      {
        onSuccess: () => setRotateTarget(null),
        onError: (error) => {
          setRotateTarget(null);
          reportWriteFailure(error);
        },
      },
    );
  };

  // 已有令牌时先确认：轮换会让旧令牌立即失效
  const handleIssueToken = (cluster: ClusterRegistryItem) => {
    if (cluster.tokenIssued) setRotateTarget(cluster);
    else issueTokenFor(cluster);
  };

  const handleUnregisterConfirm = () => {
    if (!unregisterTarget) return;
    unregister.mutate(unregisterTarget.clusterId, {
      onSuccess: () => {
        toast.success(registryT.unregisteredToast);
        setUnregisterTarget(null);
      },
      onError: (error) => {
        setUnregisterTarget(null);
        reportWriteFailure(error);
      },
    });
  };

  const renderRegistry = (clusters: ClusterRegistryItem[]) => (
    <ClusterRegistryTable
      clusters={clusters}
      readOnly={isDemo}
      // 从「未登记」行点进来：集群 ID 预填并锁定，必须与 Agent 上报的逐字一致
      onRegister={(cluster) =>
        setProfileTarget({ mode: "register", clusterId: cluster.clusterId, clusterIdLocked: true })
      }
      onEditProfile={(cluster) =>
        setProfileTarget({
          mode: "edit",
          clusterId: cluster.clusterId,
          name: cluster.name,
          description: cluster.description,
        })
      }
      onIssueToken={handleIssueToken}
      onUnregister={setUnregisterTarget}
      t={registryT}
      commonT={t.common}
    />
  );

  const registerButton = (
    <button
      type="button"
      onClick={() => setProfileTarget({ mode: "register", clusterId: "", clusterIdLocked: false })}
      // 演示模式下写操作不可用
      disabled={!canManageClusters}
      className="flex items-center gap-1.5 sm:gap-2 px-3 sm:px-4 py-2 bg-primary hover:bg-primary-hover text-white rounded-lg transition-colors text-sm disabled:opacity-40 disabled:cursor-not-allowed"
    >
      <Plus className="w-4 h-4" />
      <span className="hidden sm:inline">{registryT.registerCluster}</span>
      <span className="sm:hidden">{registryT.register}</span>
    </button>
  );

  return (
    <Layout>
      <div className="space-y-6">
        <PageHeader
          title={t.nav.clusters}
          description={registryT.pageDescription}
          autoRefreshSeconds={REFRESH.slow / 1000}
          actions={registerButton}
        />

        {isDemo && <DemoBanner hint={registryT.demoHint} />}

        {!sessionReady ? (
          <div className="py-12">
            <LoadingSpinner />
          </div>
        ) : isDemo ? (
          renderRegistry(MOCK_CLUSTER_REGISTRY)
        ) : (
          <QueryView
            query={registryQuery}
            isEmpty={(clusters) => clusters.length === 0}
            empty={
              <div className="bg-card rounded-xl border border-[var(--border-color)] py-12 text-center">
                <Server className="w-12 h-12 mx-auto mb-3 text-muted opacity-50" />
                <p className="text-default font-medium mb-1">{registryT.empty}</p>
                <p className="text-sm text-muted">{registryT.emptyHint}</p>
              </div>
            }
          >
            {renderRegistry}
          </QueryView>
        )}
      </div>

      {/* 登记 / 编辑弹窗：按目标整个重新挂载，表单初值直接取自目标 */}
      {profileTarget && (
        <ClusterProfileModal
          key={`${profileTarget.mode}:${profileTarget.clusterId}`}
          target={profileTarget}
          onClose={() => setProfileTarget(null)}
          t={registryT}
          commonT={t.common}
        />
      )}

      {/* 轮换确认：旧令牌会立即失效 */}
      <ConfirmDialog
        isOpen={rotateTarget !== null}
        onClose={() => setRotateTarget(null)}
        onConfirm={() => rotateTarget && issueTokenFor(rotateTarget)}
        title={registryT.rotateConfirmTitle}
        message={registryT.rotateConfirmMessage.replace("{name}", rotateTarget?.name || rotateTarget?.clusterId || "")}
        confirmText={registryT.rotateToken}
        cancelText={t.common.cancel}
        loading={issueToken.isPending}
        variant="warning"
      />

      {/* 删除登记确认：令牌随之失效，历史数据保留 */}
      <ConfirmDialog
        isOpen={unregisterTarget !== null}
        onClose={() => setUnregisterTarget(null)}
        onConfirm={handleUnregisterConfirm}
        title={registryT.unregisterConfirmTitle}
        message={registryT.unregisterConfirmMessage.replace(
          "{name}",
          unregisterTarget?.name || unregisterTarget?.clusterId || "",
        )}
        confirmText={registryT.unregister}
        cancelText={t.common.cancel}
        loading={unregister.isPending}
        variant="danger"
      />

      {/* 令牌只显示这一次：关闭即从内存丢弃 */}
      {issuedToken && (
        <ClusterTokenDialog issued={issuedToken} onClose={() => setIssuedToken(null)} t={registryT} />
      )}
    </Layout>
  );
}
