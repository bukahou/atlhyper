"use client";

import { Calendar } from "lucide-react";
import { CompareMetric } from "./common";

interface CompareTabTranslations {
  currentVsPrevious: string;
  previousPeriod: string;
  availability: string;
  p95Latency: string;
  errorRate: string;
}

export function CompareTab({ current, previous, t }: {
  current: { availability: number; p95Latency: number; errorRate: number };
  previous: { availability: number; p95Latency: number; errorRate: number };
  t: CompareTabTranslations;
}) {
  return (
    <div className="p-3 sm:p-4 space-y-3 sm:space-y-4">
      <div className="flex items-center gap-2 text-xs text-muted">
        <Calendar className="w-4 h-4" />
        <span>{t.currentVsPrevious}</span>
      </div>
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 sm:gap-4">
        <CompareMetric
          label={t.availability}
          current={current.availability}
          previous={previous.availability}
          unit="%"
          inverse={false}
          previousPeriodLabel={t.previousPeriod}
        />
        <CompareMetric
          label={t.p95Latency}
          current={current.p95Latency}
          previous={previous.p95Latency}
          unit="ms"
          inverse={true}
          previousPeriodLabel={t.previousPeriod}
        />
        <CompareMetric
          label={t.errorRate}
          current={current.errorRate}
          previous={previous.errorRate}
          unit="%"
          inverse={true}
          previousPeriodLabel={t.previousPeriod}
        />
      </div>
    </div>
  );
}
