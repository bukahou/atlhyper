"use client";

import { useState, useCallback, useMemo } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { getEventOverview } from "@/datasource/cluster";
import { PageHeader, StatsCard, DataTable, StatusBadge, type TableColumn } from "@/components/common";
import { useClusterStore } from "@/store/clusterStore";
import { useAutoRefresh } from "@/hooks/useAutoRefresh";
import { Filter, X } from "lucide-react";
import type { EventLog, EventOverview } from "@/types/cluster";
import { EventDetailDrawer } from "@/components/event";

function FilterInput({
  value,
  onChange,
  onClear,
  placeholder,
}: {
  value: string;
  onChange: (value: string) => void;
  onClear: () => void;
  placeholder: string;
}) {
  return (
    <div className="relative">
      <input
        type="text"
        placeholder={placeholder}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="w-full px-3 py-2.5 sm:py-2 pr-8 bg-[var(--background)] border border-[var(--border-color)] rounded-lg text-sm text-default placeholder:text-muted focus:outline-none focus:ring-1 focus:ring-primary"
      />
      {value && (
        <button
          onClick={onClear}
          className="absolute right-2 top-1/2 -translate-y-1/2 p-1.5 text-muted hover:text-default transition-colors"
        >
          <X className="w-4 h-4 sm:w-3 sm:h-3" />
        </button>
      )}
    </div>
  );
}

function FilterSelect({
  value,
  onChange,
  onClear,
  placeholder,
  options,
}: {
  value: string;
  onChange: (value: string) => void;
  onClear: () => void;
  placeholder: string;
  options: { value: string; label: string }[];
}) {
  return (
    <div className="relative">
      <select
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="w-full px-3 py-2.5 sm:py-2 pr-8 bg-[var(--background)] border border-[var(--border-color)] rounded-lg text-sm text-default focus:outline-none focus:ring-1 focus:ring-primary appearance-none"
      >
        <option value="">{placeholder}</option>
        {options.map((opt) => (
          <option key={opt.value} value={opt.value}>
            {opt.label}
          </option>
        ))}
      </select>
      {value ? (
        <button
          onClick={(e) => {
            e.preventDefault();
            onClear();
          }}
          className="absolute right-2 top-1/2 -translate-y-1/2 p-1.5 text-muted hover:text-default transition-colors z-10"
        >
          <X className="w-4 h-4 sm:w-3 sm:h-3" />
        </button>
      ) : (
        <div className="absolute right-2 top-1/2 -translate-y-1/2 pointer-events-none">
          <svg className="w-4 h-4 text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
          </svg>
        </div>
      )}
    </div>
  );
}

function getSeverityLabel(severity: string): string {
  if (severity === "Warning") {
    return "Warning";
  }
  if (severity === "Critical" || severity === "Error") {
    return "Critical";
  }
  return "Normal";
}

function isCriticalEvent(event: EventLog): boolean {
  const criticalReasons = [
    "OOMKilling", "CrashLoopBackOff", "FailedScheduling",
    "FailedMount", "NodeNotReady", "FailedBinding",
  ];
  return event.severity === "Warning" && criticalReasons.includes(event.reason);
}

export default function EventPage() {
  const { t } = useI18n();
  const { currentClusterId } = useClusterStore();
  const [loading, setLoading] = useState(true);
  const [data, setData] = useState<EventOverview | null>(null);
  const [error, setError] = useState("");

  const [filters, setFilters] = useState({
    search: "",
    severity: "",
    kind: "",
    namespace: "",
  });

  const [selectedEvent, setSelectedEvent] = useState<EventLog | null>(null);
  const [drawerOpen, setDrawerOpen] = useState(false);

  const fetchData = useCallback(async () => {
    if (!currentClusterId) return;
    setError("");
    try {
      const res = await getEventOverview({ ClusterID: currentClusterId });
      setData(res.data.data);
    } catch (err) {
      setError(err instanceof Error ? err.message : t.event.loadFailed);
    } finally {
      setLoading(false);
    }
  }, [currentClusterId]);

  const { intervalSeconds } = useAutoRefresh(fetchData);

  const handleRowClick = (event: EventLog) => {
    setSelectedEvent(event);
    setDrawerOpen(true);
  };

  // Derive unique kinds and namespaces
  const { kinds, namespaces } = useMemo(() => {
    const rows = data?.rows || [];
    const kindSet = new Set<string>();
    const nsSet = new Set<string>();
    rows.forEach((e) => {
      if (e.kind) kindSet.add(e.kind);
      if (e.namespace) nsSet.add(e.namespace);
    });
    return {
      kinds: Array.from(kindSet).sort(),
      namespaces: Array.from(nsSet).sort(),
    };
  }, [data?.rows]);

  // Compute stats
  const stats = useMemo(() => {
    const rows = data?.rows || [];
    let normal = 0;
    let warning = 0;
    let critical = 0;
    rows.forEach((e) => {
      if (isCriticalEvent(e)) {
        critical++;
      } else if (e.severity === "Warning") {
        warning++;
      } else {
        normal++;
      }
    });
    return { total: rows.length, normal, warning, critical };
  }, [data?.rows]);

  // Filter events
  const filteredEvents = useMemo(() => {
    const rows = data?.rows || [];
    return rows.filter((e) => {
      if (filters.search) {
        const q = filters.search.toLowerCase();
        if (!e.name.toLowerCase().includes(q) && !e.message.toLowerCase().includes(q)) {
          return false;
        }
      }
      if (filters.kind && e.kind !== filters.kind) return false;
      if (filters.namespace && e.namespace !== filters.namespace) return false;
      if (filters.severity) {
        if (filters.severity === "Critical") {
          if (!isCriticalEvent(e)) return false;
        } else if (filters.severity === "Warning") {
          if (e.severity !== "Warning" || isCriticalEvent(e)) return false;
        } else if (filters.severity === "Normal") {
          if (e.severity !== "Normal") return false;
        }
      }
      return true;
    });
  }, [data?.rows, filters]);

  const handleFilterChange = (key: string, value: string) => {
    setFilters((prev) => ({ ...prev, [key]: value }));
  };

  const hasFilters = filters.search || filters.severity || filters.kind || filters.namespace;
  const activeCount = [filters.search, filters.severity, filters.kind, filters.namespace].filter(Boolean).length;

  const columns: TableColumn<EventLog>[] = [
    {
      key: "time",
      header: t.common.time,
      render: (e) => {
        const d = new Date(e.eventTime || e.time);
        return (
          <span className="text-xs text-muted whitespace-nowrap font-mono">
            {d.toLocaleString()}
          </span>
        );
      },
    },
    {
      key: "severity",
      header: t.common.status,
      render: (e) => {
        const label = isCriticalEvent(e) ? "Critical" : getSeverityLabel(e.severity);
        return <StatusBadge status={label} />;
      },
    },
    {
      key: "kind",
      header: t.common.type,
      render: (e) => <span className="text-sm font-medium">{e.kind}</span>,
    },
    {
      key: "name",
      header: t.common.name,
      mobileTitle: true,
      render: (e) => (
        <div className="max-w-[200px]">
          <span className="font-medium text-default truncate block">{e.name}</span>
        </div>
      ),
    },
    {
      key: "namespace",
      header: t.common.namespace,
      mobileVisible: false,
      render: (e) => <span className="text-sm">{e.namespace || "-"}</span>,
    },
    {
      key: "reason",
      header: t.event.reason,
      render: (e) => <span className="text-sm">{e.reason}</span>,
    },
    {
      key: "source",
      header: t.event.source,
      mobileVisible: false,
      render: (e) => <span className="text-xs text-muted">{e.source || "-"}</span>,
    },
    {
      key: "message",
      header: t.alert.message,
      mobileVisible: false,
      render: (e) => (
        <div className="max-w-[300px]">
          <span className="text-sm text-muted truncate block">{e.message}</span>
        </div>
      ),
    },
    {
      key: "count",
      header: t.event.count,
      mobileVisible: false,
      render: (e) => <span className="text-sm font-mono">{e.count ?? 1}</span>,
    },
  ];

  // Sort by time descending
  const sortedEvents = useMemo(() => {
    return [...filteredEvents].sort((a, b) => {
      const ta = new Date(a.eventTime || a.time).getTime();
      const tb = new Date(b.eventTime || b.time).getTime();
      return tb - ta;
    });
  }, [filteredEvents]);

  return (
    <Layout>
      <div className="space-y-4">
        <PageHeader
          title={t.nav.event}
          description={t.event.pageDescription}
          autoRefreshSeconds={intervalSeconds}
        />

        {/* Stats Cards */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <StatsCard label={t.event.totalEvents} value={stats.total} />
          <StatsCard label={t.event.normalEvents} value={stats.normal} iconColor="text-green-500" />
          <StatsCard label={t.event.warningEvents} value={stats.warning} iconColor="text-yellow-500" />
          <StatsCard label={t.event.criticalEvents} value={stats.critical} iconColor="text-red-500" />
        </div>

        {/* Filter Bar */}
        <div className="bg-card rounded-xl border border-[var(--border-color)] p-4">
          <div className="flex items-center gap-2 mb-3">
            <Filter className="w-4 h-4 text-muted" />
            <span className="text-sm font-medium text-default">{t.common.filter}</span>
            {activeCount > 0 && (
              <span className="px-1.5 py-0.5 text-xs bg-primary/10 text-primary rounded">
                {activeCount}
              </span>
            )}
            {hasFilters && (
              <button
                onClick={() => setFilters({ search: "", severity: "", kind: "", namespace: "" })}
                className="ml-auto flex items-center gap-1 text-xs text-muted hover:text-default transition-colors"
              >
                <X className="w-3 h-3" />
                {t.common.clearAll}
              </button>
            )}
          </div>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
            <FilterInput
              value={filters.search}
              onChange={(v) => handleFilterChange("search", v)}
              onClear={() => handleFilterChange("search", "")}
              placeholder={t.event.searchPlaceholder}
            />
            <FilterSelect
              value={filters.severity}
              onChange={(v) => handleFilterChange("severity", v)}
              onClear={() => handleFilterChange("severity", "")}
              placeholder={t.event.allSeverities}
              options={[
                { value: "Normal", label: "Normal" },
                { value: "Warning", label: "Warning" },
                { value: "Critical", label: "Critical" },
              ]}
            />
            <FilterSelect
              value={filters.kind}
              onChange={(v) => handleFilterChange("kind", v)}
              onClear={() => handleFilterChange("kind", "")}
              placeholder={t.event.allKinds}
              options={kinds.map((k) => ({ value: k, label: k }))}
            />
            <FilterSelect
              value={filters.namespace}
              onChange={(v) => handleFilterChange("namespace", v)}
              onClear={() => handleFilterChange("namespace", "")}
              placeholder={t.event.allNamespaces}
              options={namespaces.map((ns) => ({ value: ns, label: ns }))}
            />
          </div>
        </div>

        {/* Data Table */}
        <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
          <DataTable
            columns={columns}
            data={sortedEvents}
            loading={loading}
            error={error}
            keyExtractor={(e, i) => `${i}-${e.kind}/${e.namespace}/${e.name}/${e.eventTime}`}
            onRowClick={handleRowClick}
            pageSize={15}
          />
        </div>
      </div>

      {/* Event Detail Drawer */}
      {selectedEvent && (
        <EventDetailDrawer
          isOpen={drawerOpen}
          onClose={() => setDrawerOpen(false)}
          event={selectedEvent}
        />
      )}
    </Layout>
  );
}
