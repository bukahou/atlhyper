"use client";

import { useState, useCallback } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { getNodeOverview, cordonNode, uncordonNode } from "@/api/node";
import { useAutoRefresh } from "@/hooks/useAutoRefresh";
import { PageHeader, StatsCard, StatusBadge, LoadingSpinner, ConfirmDialog } from "@/components/common";
import { getCurrentClusterId } from "@/config/cluster";
import { Shield, ShieldOff, Cpu, HardDrive, Eye } from "lucide-react";
import { useRequireAuth } from "@/hooks/useRequireAuth";
import type { NodeItem, NodeOverview } from "@/types/cluster";
import { NodeDetailModal } from "@/components/node";

// Node 卡片组件
function NodeCard({
  node,
  onViewDetail,
  onToggleSchedulable,
}: {
  node: NodeItem;
  onViewDetail: () => void;
  onToggleSchedulable: () => void;
}) {
  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-6">
      <div className="flex items-start justify-between mb-4">
        <div>
          <h3 className="text-lg font-semibold text-default">{node.name}</h3>
          <div className="flex items-center gap-2 mt-1">
            <StatusBadge status={node.ready ? "Ready" : "NotReady"} />
            <StatusBadge status={node.architecture} type="info" />
            <StatusBadge
              status={node.schedulable ? "Schedulable" : "Unschedulable"}
              type={node.schedulable ? "success" : "warning"}
            />
          </div>
        </div>
        <div className="flex gap-1">
          <button onClick={onViewDetail} className="p-2 hover-bg rounded-lg" title="查看详情">
            <Eye className="w-4 h-4 text-muted" />
          </button>
          <button
            onClick={onToggleSchedulable}
            className="p-2 hover-bg rounded-lg"
            title={node.schedulable ? "封锁节点" : "解封节点"}
          >
            {node.schedulable ? (
              <Shield className="w-4 h-4 text-muted" />
            ) : (
              <ShieldOff className="w-4 h-4 text-yellow-500" />
            )}
          </button>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-4 mb-4">
        <div className="flex items-center gap-2">
          <Cpu className="w-4 h-4 text-muted" />
          <div>
            <p className="text-sm text-muted">CPU Cores</p>
            <p className="font-medium">{node.cpuCores}</p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <HardDrive className="w-4 h-4 text-muted" />
          <div>
            <p className="text-sm text-muted">Memory</p>
            <p className="font-medium">{node.memoryGiB.toFixed(1)} GiB</p>
          </div>
        </div>
      </div>

      <div className="space-y-2 text-sm">
        <div className="flex justify-between">
          <span className="text-muted">IP</span>
          <span className="font-mono text-xs">{node.internalIP}</span>
        </div>
        <div className="flex justify-between">
          <span className="text-muted">OS</span>
          <span>{node.osImage}</span>
        </div>
      </div>
    </div>
  );
}

export default function NodePage() {
  const { t } = useI18n();
  const requireAuth = useRequireAuth();
  const [loading, setLoading] = useState(true);
  const [data, setData] = useState<NodeOverview | null>(null);
  const [error, setError] = useState("");

  // 详情弹窗状态
  const [selectedNode, setSelectedNode] = useState<NodeItem | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);

  // 封锁/解封确认状态
  const [blockTarget, setBlockTarget] = useState<NodeItem | null>(null);
  const [blockLoading, setBlockLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setError("");
    try {
      const res = await getNodeOverview({ ClusterID: getCurrentClusterId() });
      setData(res.data.data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "加载失败");
    } finally {
      setLoading(false);
    }
  }, []);

  const { intervalSeconds } = useAutoRefresh(fetchData);

  // 查看详情
  const handleViewDetail = (node: NodeItem) => {
    setSelectedNode(node);
    setDetailOpen(true);
  };

  // 封锁/解封确认（需要先登录）
  const handleToggleSchedulable = (node: NodeItem) => {
    requireAuth(() => setBlockTarget(node));
  };

  // 执行封锁/解封
  const handleBlockConfirm = async () => {
    if (!blockTarget) return;
    setBlockLoading(true);
    try {
      if (blockTarget.schedulable) {
        await cordonNode({ ClusterID: getCurrentClusterId(), Node: blockTarget.name });
      } else {
        await uncordonNode({ ClusterID: getCurrentClusterId(), Node: blockTarget.name });
      }
      setBlockTarget(null);
      // 延迟2秒后刷新，给后端处理时间
      setTimeout(() => fetchData(), 2000);
    } catch (err) {
      console.error("Block/Unblock failed:", err);
    } finally {
      setBlockLoading(false);
    }
  };

  return (
    <Layout>
      <div className="space-y-6">
        <PageHeader title={t.nav.node} description="Node 资源监控与管理" autoRefreshSeconds={intervalSeconds} />

        {data && (
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <StatsCard label={t.common.total} value={data.cards.totalNodes} />
            <StatsCard label="Ready" value={data.cards.readyNodes} iconColor="text-green-500" />
            <StatsCard label="Total CPU" value={data.cards.totalCPU} iconColor="text-blue-500" />
            <StatsCard label="Total Memory" value={`${data.cards.totalMemoryGiB.toFixed(1)} GiB`} iconColor="text-purple-500" />
          </div>
        )}

        {loading ? (
          <LoadingSpinner />
        ) : error ? (
          <div className="text-center py-12 text-red-500">{error}</div>
        ) : !data?.rows?.length ? (
          <div className="text-center py-12 text-gray-500">{t.common.noData}</div>
        ) : (
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
            {data.rows.map((node) => (
              <NodeCard
                key={node.name}
                node={node}
                onViewDetail={() => handleViewDetail(node)}
                onToggleSchedulable={() => handleToggleSchedulable(node)}
              />
            ))}
          </div>
        )}
      </div>

      {/* Node 详情弹窗 */}
      {selectedNode && (
        <NodeDetailModal
          isOpen={detailOpen}
          onClose={() => setDetailOpen(false)}
          nodeName={selectedNode.name}
        />
      )}

      {/* 封锁/解封确认对话框 */}
      <ConfirmDialog
        isOpen={!!blockTarget}
        onClose={() => setBlockTarget(null)}
        onConfirm={handleBlockConfirm}
        title={blockTarget?.schedulable ? "确认封锁节点" : "确认解封节点"}
        message={
          blockTarget?.schedulable
            ? `确定要封锁节点 "${blockTarget?.name}" 吗？封锁后新的 Pod 将不会调度到此节点。`
            : `确定要解封节点 "${blockTarget?.name}" 吗？解封后 Pod 可以调度到此节点。`
        }
        confirmText={blockTarget?.schedulable ? "封锁" : "解封"}
        cancelText="取消"
        loading={blockLoading}
        variant={blockTarget?.schedulable ? "warning" : "info"}
      />
    </Layout>
  );
}
