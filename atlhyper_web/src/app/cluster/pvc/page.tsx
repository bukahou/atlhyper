"use client";

import { useState, useCallback, useMemo } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { getPVCList } from "@/datasource/cluster";
import type { PVCItem } from "@/api/cluster-resources";
import { useAutoRefresh } from "@/hooks/useAutoRefresh";
import { PageHeader, StatsCard, DataTable, StatusBadge, type TableColumn } from "@/components/common";
import { useClusterStore } from "@/store/clusterStore";
import { Eye } from "lucide-react";
import { PVCDetailModal } from "@/components/pvc";
import { PVCFilterBar } from "@/components/pvc/PVCFilterBar";

export default function PVCPage() {
  const { t } = useI18n();
  const { currentClusterId } = useClusterStore();
  const [items, setItems] = useState<PVCItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  // 筛选状态
  const [filters, setFilters] = useState({
    namespace: "",
    search: "",
  });

  // 详情弹窗状态
  const [selectedItem, setSelectedItem] = useState<PVCItem | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);

  const fetchData = useCallback(async () => {
    setError("");
    try {
      const res = await getPVCList({ cluster_id: currentClusterId });
      setItems(res.data.data || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : t.common.loadFailed);
    } finally {
      setLoading(false);
    }
  }, []);

  const { intervalSeconds } = useAutoRefresh(fetchData);

  // 提取唯一的 namespaces
  const namespaces = useMemo(() => {
    const nsSet = new Set<string>();
    items.forEach((pvc) => {
      if (pvc.namespace) nsSet.add(pvc.namespace);
    });
    return Array.from(nsSet).sort();
  }, [items]);

  // StatsCards 统计
  const stats = useMemo(() => {
    const bound = items.filter((pvc) => pvc.phase === "Bound").length;
    const pending = items.filter((pvc) => pvc.phase === "Pending").length;
    const lost = items.filter((pvc) => pvc.phase === "Lost").length;
    return {
      total: items.length,
      bound,
      pending,
      lost,
    };
  }, [items]);

  // 根据筛选条件过滤数据
  const filteredItems = useMemo(() => {
    return items.filter((pvc) => {
      if (filters.search && !pvc.name.toLowerCase().includes(filters.search.toLowerCase())) {
        return false;
      }
      if (filters.namespace && pvc.namespace !== filters.namespace) {
        return false;
      }
      return true;
    });
  }, [items, filters]);

  const handleFilterChange = (key: string, value: string) => {
    setFilters((prev) => ({ ...prev, [key]: value }));
  };

  const handleViewDetail = (item: PVCItem) => {
    setSelectedItem(item);
    setDetailOpen(true);
  };

  const columns: TableColumn<PVCItem>[] = [
    {
      key: "name",
      header: t.common.name,
      mobileTitle: true,
      render: (pvc) => (
        <div>
          <span className="font-medium text-default">{pvc.name || "-"}</span>
          <div className="text-xs text-muted">{pvc.age || "-"}</div>
        </div>
      ),
    },
    { key: "namespace", header: t.common.namespace },
    {
      key: "phase",
      header: t.storagePage.phase,
      render: (pvc) => {
        const typeMap: Record<string, "success" | "warning" | "error"> = {
          Bound: "success",
          Pending: "warning",
          Lost: "error",
        };
        return <StatusBadge status={pvc.phase} type={typeMap[pvc.phase] || "info"} />;
      },
    },
    {
      key: "capacity",
      header: t.storagePage.capacity,
      render: (pvc) => {
        if (pvc.phase === "Bound" && pvc.actualCapacity) {
          return pvc.requestedCapacity !== pvc.actualCapacity
            ? `${pvc.requestedCapacity} → ${pvc.actualCapacity}`
            : pvc.actualCapacity;
        }
        return pvc.requestedCapacity || "-";
      },
    },
    {
      key: "storageClass",
      header: t.storagePage.storageClass,
      mobileVisible: false,
      render: (pvc) => pvc.storageClass || "-",
    },
    {
      key: "volumeName",
      header: t.storagePage.volumeName,
      mobileVisible: false,
      render: (pvc) => pvc.volumeName || "-",
    },
    {
      key: "age",
      header: t.storagePage.age,
      mobileVisible: false,
      render: (pvc) => pvc.age || "-",
    },
    {
      key: "action",
      header: t.common.action,
      mobileVisible: false,
      render: (pvc) => (
        <button
          onClick={(e) => {
            e.stopPropagation();
            handleViewDetail(pvc);
          }}
          className="p-2 hover-bg rounded-lg"
          title={t.common.details}
        >
          <Eye className="w-4 h-4 text-muted" />
        </button>
      ),
    },
  ];

  return (
    <Layout>
      <div className="space-y-4">
        <PageHeader title={t.nav.pvc} description={t.storagePage.pvcDescription} autoRefreshSeconds={intervalSeconds} />

        {items.length > 0 && (
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <StatsCard label={t.common.total} value={stats.total} />
            <StatsCard label={t.storagePage.bound} value={stats.bound} iconColor="text-green-500" />
            <StatsCard label={t.storagePage.pending} value={stats.pending} iconColor="text-yellow-500" />
            <StatsCard label={t.storagePage.lost} value={stats.lost} iconColor="text-red-500" />
          </div>
        )}

        <PVCFilterBar
          namespaces={namespaces}
          filters={filters}
          onFilterChange={handleFilterChange}
        />

        <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
          <DataTable
            columns={columns}
            data={filteredItems}
            loading={loading}
            error={error}
            keyExtractor={(pvc, index) => `${index}-${pvc.namespace}/${pvc.name}`}
            onRowClick={handleViewDetail}
            pageSize={10}
          />
        </div>
      </div>

      {selectedItem && (
        <PVCDetailModal
          isOpen={detailOpen}
          onClose={() => setDetailOpen(false)}
          namespace={selectedItem.namespace}
          name={selectedItem.name}
        />
      )}
    </Layout>
  );
}
