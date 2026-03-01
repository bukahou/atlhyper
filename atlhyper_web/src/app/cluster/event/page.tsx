"use client";

import { useState, useCallback, useMemo } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { getEventOverview } from "@/datasource/cluster";
import { PageHeader } from "@/components/common";
import { useClusterStore } from "@/store/clusterStore";
import { useAutoRefresh } from "@/hooks/useAutoRefresh";
import type { EventLog, EventOverview } from "@/types/cluster";
import { EventDetailDrawer } from "@/components/event";
import { EventFilterBar, type EventFilters } from "./components/EventFilterBar";
import { EventTable, isCriticalEvent } from "./components/EventTable";
import { EventStatsCards } from "./components/EventStatsCards";

export default function EventPage() {
  const { t } = useI18n();
  const { currentClusterId } = useClusterStore();
  const [loading, setLoading] = useState(true);
  const [data, setData] = useState<EventOverview | null>(null);
  const [error, setError] = useState("");

  const [filters, setFilters] = useState<EventFilters>({
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

  // Derive unique kinds and namespaces for filter options
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

  // Filter events based on current filters
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

  const handleClearAll = () => {
    setFilters({ search: "", severity: "", kind: "", namespace: "" });
  };

  return (
    <Layout>
      <div className="space-y-4">
        <PageHeader
          title={t.nav.event}
          description={t.event.pageDescription}
          autoRefreshSeconds={intervalSeconds}
        />

        <EventStatsCards events={data?.rows || []} />

        <EventFilterBar
          filters={filters}
          onFilterChange={handleFilterChange}
          onClearAll={handleClearAll}
          kinds={kinds}
          namespaces={namespaces}
        />

        <EventTable
          events={filteredEvents}
          loading={loading}
          error={error}
          onRowClick={handleRowClick}
        />
      </div>

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
