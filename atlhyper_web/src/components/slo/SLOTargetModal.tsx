"use client";

import { useState } from "react";
import { Target, X } from "lucide-react";
import { upsertSLOTarget } from "@/api/slo";

type TimeRange = "1d" | "7d" | "30d";

interface SLOTargetModalTranslations {
  configSloTarget: string;
  targetDomain: string;
  selectPeriod: string;
  day: string;
  week: string;
  month: string;
  targetAvailability: string;
  targetAvailabilityHint: string;
  targetP95: string;
  targetP95Hint: string;
  errorRateThreshold: string;
  errorRateAutoCalc: string;
  cancel: string;
  save: string;
  saving: string;
}

export function SLOTargetModal({
  isOpen,
  onClose,
  domain,
  clusterId,
  timeRange,
  onSaved,
  t,
}: {
  isOpen: boolean;
  onClose: () => void;
  domain: string;
  clusterId: string;
  timeRange: TimeRange;
  onSaved: () => void;
  t: SLOTargetModalTranslations;
}) {
  const [selectedRange, setSelectedRange] = useState<TimeRange>(timeRange);
  const [availability, setAvailability] = useState(95);
  const [p95Latency, setP95Latency] = useState(300);
  const [saving, setSaving] = useState(false);

  const errorRateThreshold = (100 - availability).toFixed(2);

  const handleSave = async () => {
    setSaving(true);
    try {
      await upsertSLOTarget({
        clusterId,
        host: domain,
        timeRange: selectedRange,
        availabilityTarget: availability,
        p95LatencyTarget: p95Latency,
      });
      onSaved();
      onClose();
    } catch (err) {
      console.error("Save SLO target failed:", err);
    } finally {
      setSaving(false);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />
      <div className="relative bg-card border border-[var(--border-color)] rounded-xl shadow-xl w-full max-w-md mx-4">
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-[var(--border-color)]">
          <div className="flex items-center gap-2">
            <Target className="w-5 h-5 text-primary" />
            <h3 className="font-semibold text-default">{t.configSloTarget}</h3>
          </div>
          <button onClick={onClose} className="p-1 rounded-lg hover:bg-[var(--hover-bg)] text-muted hover:text-default">
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Content */}
        <div className="p-4 space-y-4">
          <div className="p-3 rounded-lg bg-[var(--hover-bg)]">
            <div className="text-xs text-muted mb-1">{t.targetDomain}</div>
            <div className="font-medium text-default">{domain}</div>
          </div>

          <div>
            <label className="block text-sm font-medium text-default mb-2">{t.selectPeriod}</label>
            <div className="flex gap-2">
              {([
                { value: "1d", label: t.day },
                { value: "7d", label: t.week },
                { value: "30d", label: t.month },
              ] as const).map((range) => (
                <button
                  key={range.value}
                  onClick={() => setSelectedRange(range.value)}
                  className={`flex-1 px-3 py-2 text-sm rounded-lg border transition-colors ${
                    selectedRange === range.value
                      ? "border-primary bg-primary/10 text-primary"
                      : "border-[var(--border-color)] text-muted hover:text-default hover:bg-[var(--hover-bg)]"
                  }`}
                >
                  {range.label}
                </button>
              ))}
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-default mb-2">{t.targetAvailability}</label>
            <input
              type="number"
              value={availability}
              onChange={(e) => setAvailability(Math.min(100, Math.max(0, Number(e.target.value))))}
              min={0} max={100} step={0.1}
              className="w-full px-3 py-2 rounded-lg border border-[var(--border-color)] bg-[var(--background)] text-default focus:outline-none focus:ring-2 focus:ring-primary/50"
            />
            <p className="text-xs text-muted mt-1">{t.targetAvailabilityHint}</p>
          </div>

          <div>
            <label className="block text-sm font-medium text-default mb-2">{t.targetP95}</label>
            <input
              type="number"
              value={p95Latency}
              onChange={(e) => setP95Latency(Math.max(0, Number(e.target.value)))}
              min={0} step={10}
              className="w-full px-3 py-2 rounded-lg border border-[var(--border-color)] bg-[var(--background)] text-default focus:outline-none focus:ring-2 focus:ring-primary/50"
            />
            <p className="text-xs text-muted mt-1">{t.targetP95Hint}</p>
          </div>

          <div className="p-3 rounded-lg bg-[var(--hover-bg)]">
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted">{t.errorRateThreshold}</span>
              <span className="text-sm font-medium text-default">{errorRateThreshold}%</span>
            </div>
            <p className="text-xs text-muted mt-1">{t.errorRateAutoCalc}</p>
          </div>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-3 px-4 py-3 border-t border-[var(--border-color)]">
          <button onClick={onClose} className="px-4 py-2 text-sm rounded-lg border border-[var(--border-color)] text-muted hover:text-default hover:bg-[var(--hover-bg)]">
            {t.cancel}
          </button>
          <button onClick={handleSave} disabled={saving} className="px-4 py-2 text-sm rounded-lg bg-primary text-white hover:bg-primary/90 disabled:opacity-50">
            {saving ? t.saving : t.save}
          </button>
        </div>
      </div>
    </div>
  );
}
