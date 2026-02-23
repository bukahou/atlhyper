"use client";

import { useMemo } from "react";
import type { TraceSummary, OperationStats } from "@/types/model/apm";
import type { ApmTranslations } from "@/types/i18n";
import {
  getDependencies,
  getSpanTypeBreakdown,
} from "@/datasource/apm";
import { LatencyChart } from "./LatencyChart";
import { ThroughputChart } from "./ThroughputChart";
import { ErrorRateChart } from "./ErrorRateChart";
import { SpanTypeChart } from "./SpanTypeChart";
import { TransactionsTable } from "./TransactionsTable";
import { DependenciesTable } from "./DependenciesTable";

interface ServiceOverviewProps {
  t: ApmTranslations;
  serviceName: string;
  traces: TraceSummary[];
  operations: OperationStats[];
  onSelectTrace: (traceId: string) => void;
  onNavigateToService?: (serviceName: string) => void;
}

const TABS = ["overview", "transactions", "dependencies", "errors"] as const;

export function ServiceOverview({
  t,
  serviceName,
  traces,
  operations,
  onSelectTrace,
  onNavigateToService,
}: ServiceOverviewProps) {
  const serviceTraces = useMemo(
    () => traces.filter((tr) => tr.rootService === serviceName),
    [traces, serviceName]
  );

  const serviceOperations = useMemo(
    () => operations.filter((op) => op.serviceName === serviceName),
    [operations, serviceName]
  );

  const dependencies = useMemo(
    () => getDependencies(serviceName),
    [serviceName]
  );

  const spanBreakdown = useMemo(
    () => getSpanTypeBreakdown(serviceName),
    [serviceName]
  );

  const tabLabels: Record<(typeof TABS)[number], string> = {
    overview: t.overview,
    transactions: t.transactions,
    dependencies: t.dependencies,
    errors: t.errors,
  };

  return (
    <div className="space-y-6">
      {/* Tabs — only overview is active */}
      <div className="flex gap-0 border-b border-[var(--border-color)]">
        {TABS.map((tab) => (
          <button
            key={tab}
            disabled={tab !== "overview"}
            className={`px-4 py-2 text-sm font-medium transition-colors border-b-2 -mb-px ${
              tab === "overview"
                ? "text-primary border-primary"
                : "text-muted/50 border-transparent cursor-not-allowed"
            }`}
          >
            {tabLabels[tab]}
          </button>
        ))}
      </div>

      {/* Overview dashboard */}
      <div className="space-y-6">
        {/* Latency chart — full width */}
        <div className="border border-[var(--border-color)] rounded-xl p-4 bg-card">
          <LatencyChart title={t.latencyChartTitle} traces={serviceTraces} />
        </div>

        {/* Middle section: charts + tables in 2-column layout */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          {/* Left column: Throughput + Error Rate */}
          <div className="space-y-4">
            <div className="border border-[var(--border-color)] rounded-xl p-4 bg-card">
              <ThroughputChart title={t.throughputChartTitle} traces={serviceTraces} />
            </div>
            <div className="border border-[var(--border-color)] rounded-xl p-4 bg-card">
              <ErrorRateChart title={t.errorRateChartTitle} traces={serviceTraces} />
            </div>
          </div>

          {/* Right column: Transactions + Dependencies tables */}
          <div className="space-y-4">
            <div className="border border-[var(--border-color)] rounded-xl p-4 bg-card">
              <TransactionsTable
                t={t}
                operations={serviceOperations}
                onSelectOperation={(op) => {
                  const match = serviceTraces.find((tr) => tr.rootOperation === op);
                  if (match) onSelectTrace(match.traceId);
                }}
              />
            </div>
            <div className="border border-[var(--border-color)] rounded-xl p-4 bg-card">
              <DependenciesTable
                t={t}
                dependencies={dependencies}
                onSelectDependency={onNavigateToService}
              />
            </div>
          </div>
        </div>

        {/* Span type breakdown — full width */}
        <div className="border border-[var(--border-color)] rounded-xl p-4 bg-card">
          <SpanTypeChart title={t.spanTypeChartTitle} breakdown={spanBreakdown} />
        </div>
      </div>
    </div>
  );
}
