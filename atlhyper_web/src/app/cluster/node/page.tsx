"use client";

import { useState, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { PageHeader, QueryView, StatsCard, DataTable, StatusBadge, type TableColumn } from "@/components/common";
import { useCurrentCluster } from "@/hooks/useCurrentCluster";
import { nodeOverviewQuery } from "@/queries/cluster";
import { REFRESH } from "@/queries/refresh";
import type { NodeItem } from "@/types/cluster";
import { NodeDetailModal } from "@/components/node";
import { NodeFilterBar, NO_NODE_FILTERS, type NodeFilters } from "./components/NodeFilterBar";

export default function NodePage() {
  const { t } = useI18n();
  const { clusterId } = useCurrentCluster();
  const query = useQuery(nodeOverviewQuery(clusterId));
  const data = query.data;

  // 筛选状态
  const [filters, setFilters] = useState<NodeFilters>(NO_NODE_FILTERS);

  // 详情弹窗状态
  const [selectedNode, setSelectedNode] = useState<NodeItem | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);

  // 查看详情
  const handleRowClick = (node: NodeItem) => {
    setSelectedNode(node);
    setDetailOpen(true);
  };

  // 动态提取架构选项
  const architectureOptions = useMemo(() => {
    const rows = data?.rows || [];
    const archSet = new Set<string>();
    rows.forEach((node) => {
      if (node.architecture) archSet.add(node.architecture);
    });
    return Array.from(archSet).sort().map((a) => ({ value: a, label: a }));
  }, [data?.rows]);

  // 客户端筛选
  const filteredNodes = useMemo(() => {
    const rows = data?.rows || [];
    return rows.filter((node) => {
      if (filters.search && !node.name.toLowerCase().includes(filters.search.toLowerCase())) {
        return false;
      }
      if (filters.status) {
        const isReady = node.ready ? "Ready" : "NotReady";
        if (isReady !== filters.status) return false;
      }
      if (filters.architecture && node.architecture !== filters.architecture) {
        return false;
      }
      if (filters.schedulable) {
        const sched = node.schedulable ? "Schedulable" : "Unschedulable";
        if (sched !== filters.schedulable) return false;
      }
      return true;
    });
  }, [data?.rows, filters]);

  const handleFilterChange = (key: string, value: string) => {
    setFilters((prev) => ({ ...prev, [key]: value }));
  };

  const columns: TableColumn<NodeItem>[] = [
    {
      key: "name",
      header: t.common.name,
      mobileTitle: true,
      render: (node) => <span className="font-medium text-default">{node.name}</span>,
    },
    {
      key: "status",
      header: t.common.status,
      render: (node) => (
        <div className="flex items-center gap-1.5">
          <StatusBadge status={node.ready ? "Ready" : "NotReady"} />
          <StatusBadge
            status={node.schedulable !== false ? "Schedulable" : "Unschedulable"}
            type={node.schedulable !== false ? "success" : "warning"}
          />
        </div>
      ),
    },
    {
      key: "architecture",
      header: t.node.architecture,
      mobileVisible: false,
      render: (node) => node.architecture ? <StatusBadge status={node.architecture} type="info" /> : <span className="text-muted">-</span>,
    },
    {
      key: "cpu",
      header: "CPU",
      render: (node) => <span className="text-sm">{node.cpuCores != null ? `${node.cpuCores} cores` : "-"}</span>,
    },
    {
      key: "memory",
      header: "Memory",
      render: (node) => <span className="text-sm">{node.memoryGiB != null ? `${node.memoryGiB.toFixed(1)} GiB` : "-"}</span>,
    },
    {
      key: "ip",
      header: "IP",
      mobileVisible: false,
      render: (node) => <span className="font-mono text-xs">{node.internalIP || "-"}</span>,
    },
    {
      key: "os",
      header: "OS",
      mobileVisible: false,
      render: (node) => <span className="truncate max-w-[180px] block text-sm" title={node.osImage}>{node.osImage || "-"}</span>,
    },
  ];

  return (
    <Layout>
      <div className="space-y-4">
        <PageHeader title={t.nav.node} description={t.node.pageDescription} autoRefreshSeconds={REFRESH.standard / 1000} />

        {data && (
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <StatsCard label={t.common.total} value={data.cards.totalNodes ?? 0} />
            <StatsCard label={t.status.ready} value={data.cards.readyNodes ?? 0} iconColor="text-green-500" />
            <StatsCard label="Total CPU" value={data.cards.totalCPU ?? 0} iconColor="text-blue-500" />
            <StatsCard label="Total Memory" value={data.cards.totalMemoryGiB != null ? `${data.cards.totalMemoryGiB.toFixed(1)} GiB` : "-"} iconColor="text-purple-500" />
          </div>
        )}

        <NodeFilterBar
          filters={filters}
          architectureOptions={architectureOptions}
          onFilterChange={handleFilterChange}
          onClearAll={() => setFilters(NO_NODE_FILTERS)}
        />

        {/* 数据表格 */}
        <QueryView query={query} requireCluster>
          {() => (
            <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
              <DataTable
                columns={columns}
                data={filteredNodes}
                loading={false}
                error=""
                keyExtractor={(node) => node.name}
                onRowClick={handleRowClick}
                pageSize={10}
              />
            </div>
          )}
        </QueryView>
      </div>

      {/* Node 详情弹窗 */}
      {selectedNode && (
        <NodeDetailModal
          isOpen={detailOpen}
          onClose={() => setDetailOpen(false)}
          nodeName={selectedNode.name}
        />
      )}
    </Layout>
  );
}
