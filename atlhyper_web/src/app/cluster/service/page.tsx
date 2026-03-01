"use client";

import { useState, useCallback, useMemo } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { getServiceOverview } from "@/datasource/cluster";
import { useAutoRefresh } from "@/hooks/useAutoRefresh";
import { PageHeader, StatsCard, DataTable, StatusBadge, type TableColumn } from "@/components/common";
import { getCurrentClusterId } from "@/config/cluster";
import { Eye } from "lucide-react";
import type { ServiceItem, ServiceOverview } from "@/types/cluster";
import { ServiceDetailModal } from "@/components/service";
import { ServiceFilterBar } from "./components/ServiceFilterBar";

export default function ServicePage() {
  const { t } = useI18n();
  const [loading, setLoading] = useState(true);
  const [data, setData] = useState<ServiceOverview | null>(null);
  const [error, setError] = useState("");

  // Filter state
  const [filters, setFilters] = useState({
    namespace: "",
    type: "",
    search: "",
  });

  // Detail modal state
  const [selectedService, setSelectedService] = useState<ServiceItem | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);

  const fetchData = useCallback(async () => {
    setError("");
    try {
      const res = await getServiceOverview({ ClusterID: getCurrentClusterId() });
      setData(res.data.data);
    } catch (err) {
      setError(err instanceof Error ? err.message : t.common.loadFailed);
    } finally {
      setLoading(false);
    }
  }, []);

  const { intervalSeconds } = useAutoRefresh(fetchData);

  // Extract unique namespaces and types
  const { namespaces, types } = useMemo(() => {
    const rows = data?.rows || [];
    const nsSet = new Set<string>();
    const typeSet = new Set<string>();

    rows.forEach((s) => {
      if (s.namespace) nsSet.add(s.namespace);
      if (s.type) typeSet.add(s.type);
    });

    return {
      namespaces: Array.from(nsSet).sort(),
      types: Array.from(typeSet).sort(),
    };
  }, [data?.rows]);

  // Filter items
  const filteredServices = useMemo(() => {
    const rows = data?.rows || [];
    return rows.filter((s) => {
      if (filters.search && !s.name.toLowerCase().includes(filters.search.toLowerCase())) {
        return false;
      }
      if (filters.namespace && s.namespace !== filters.namespace) {
        return false;
      }
      if (filters.type && s.type !== filters.type) {
        return false;
      }
      return true;
    });
  }, [data?.rows, filters]);

  const handleFilterChange = (key: string, value: string) => {
    setFilters((prev) => ({ ...prev, [key]: value }));
  };

  const handleViewDetail = (service: ServiceItem) => {
    setSelectedService(service);
    setDetailOpen(true);
  };

  const getTypeStatus = (type: string): "success" | "info" | "default" => {
    if (type === "LoadBalancer") return "success";
    if (type === "NodePort") return "info";
    return "default";
  };

  const columns: TableColumn<ServiceItem>[] = [
    {
      key: "name",
      header: t.common.name,
      mobileTitle: true,
      render: (s) => <span className="font-medium text-default">{s.name || "-"}</span>,
    },
    { key: "namespace", header: t.common.namespace },
    {
      key: "type",
      header: t.service.serviceType,
      render: (s) => <StatusBadge status={s.type || "-"} type={getTypeStatus(s.type || "")} />,
    },
    {
      key: "clusterIP",
      header: t.service.clusterIP,
      mobileVisible: false,
      render: (s) => <span className="font-mono text-sm">{s.clusterIP || "-"}</span>,
    },
    {
      key: "ports",
      header: t.service.ports,
      render: (s) => <span className="text-sm">{s.ports || "-"}</span>,
    },
    {
      key: "selector",
      header: t.service.selector,
      mobileVisible: false,
      render: (s) => (
        <span className="text-xs text-muted truncate max-w-[150px] block" title={s.selector || ""}>
          {s.selector || "-"}
        </span>
      ),
    },
    {
      key: "action",
      header: t.common.action,
      mobileVisible: false,
      render: (s) => (
        <button
          onClick={(e) => {
            e.stopPropagation();
            handleViewDetail(s);
          }}
          className="p-2 hover-bg rounded-lg"
          title={t.service.viewDetails}
        >
          <Eye className="w-4 h-4 text-muted" />
        </button>
      ),
    },
  ];

  return (
    <Layout>
      <div className="space-y-4">
        <PageHeader
          title={t.nav.service}
          description={t.service.pageDescription}
          autoRefreshSeconds={intervalSeconds}
        />

        {data && (
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <StatsCard label={t.common.total} value={data.cards.totalServices} />
            <StatsCard label="External" value={data.cards.externalServices} iconColor="text-green-500" />
            <StatsCard label="Internal" value={data.cards.internalServices} iconColor="text-blue-500" />
            <StatsCard label="Headless" value={data.cards.headlessServices} iconColor="text-gray-500" />
          </div>
        )}

        <ServiceFilterBar
          namespaces={namespaces}
          types={types}
          filters={filters}
          onFilterChange={handleFilterChange}
        />

        <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
          <DataTable
            columns={columns}
            data={filteredServices}
            loading={loading}
            error={error}
            keyExtractor={(s, index) => `${index}-${s.namespace}/${s.name}`}
            onRowClick={handleViewDetail}
            pageSize={10}
          />
        </div>
      </div>

      {selectedService && (
        <ServiceDetailModal
          isOpen={detailOpen}
          onClose={() => setDetailOpen(false)}
          namespace={selectedService.namespace}
          serviceName={selectedService.name}
        />
      )}
    </Layout>
  );
}
