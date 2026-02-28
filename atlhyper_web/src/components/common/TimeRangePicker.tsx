"use client";

import { useState, useRef, useEffect } from "react";
import { Calendar, ChevronDown } from "lucide-react";
import type { PresetKey, RelativeUnit, TimeRangeSelection } from "@/types/time-range";
import type { LogTranslations } from "@/types/i18n";
import { toDisplayLabel } from "@/lib/time-range";

const PRESET_KEYS: PresetKey[] = ["15min", "1h", "24h", "7d", "15d", "30d"];
const RELATIVE_UNITS: { value: RelativeUnit; labelKey: "timeRangeMinutes" | "timeRangeHours" | "timeRangeDays" }[] = [
  { value: "m", labelKey: "timeRangeMinutes" },
  { value: "h", labelKey: "timeRangeHours" },
  { value: "d", labelKey: "timeRangeDays" },
];

interface TimeRangePickerProps {
  value: TimeRangeSelection;
  onChange: (value: TimeRangeSelection) => void;
  t: LogTranslations;
}

export function TimeRangePicker({ value, onChange, t }: TimeRangePickerProps) {
  const [isOpen, setIsOpen] = useState(false);
  const panelRef = useRef<HTMLDivElement>(null);

  // 自定义相对时间临时状态
  const [customValue, setCustomValue] = useState(45);
  const [customUnit, setCustomUnit] = useState<RelativeUnit>("m");

  // 绝对时间临时状态
  const [absStart, setAbsStart] = useState("");
  const [absEnd, setAbsEnd] = useState("");
  const [absError, setAbsError] = useState("");

  // 面板外点击关闭
  useEffect(() => {
    if (!isOpen) return;
    const handleClick = (e: MouseEvent) => {
      if (panelRef.current && !panelRef.current.contains(e.target as Node)) {
        setIsOpen(false);
      }
    };
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, [isOpen]);

  const presetLabels: Record<PresetKey, string> = {
    "15min": t.last15min,
    "1h": t.last1h,
    "24h": t.last24h,
    "7d": t.last7d,
    "15d": t.last15d,
    "30d": t.last30d,
  };

  const unitLabels = {
    minutes: t.timeRangeMinutes,
    hours: t.timeRangeHours,
    days: t.timeRangeDays,
  };

  const displayLabel = toDisplayLabel(value, presetLabels, unitLabels);

  const handlePresetClick = (preset: PresetKey) => {
    onChange({ mode: "preset", preset });
    setIsOpen(false);
  };

  const handleCustomApply = () => {
    if (customValue <= 0) return;
    onChange({ mode: "custom", value: customValue, unit: customUnit });
    setIsOpen(false);
  };

  const handleAbsoluteApply = () => {
    if (!absStart || !absEnd) return;
    const startMs = new Date(absStart).getTime();
    const endMs = new Date(absEnd).getTime();
    if (isNaN(startMs) || isNaN(endMs) || endMs <= startMs) {
      setAbsError(t.timeRangeInvalidRange);
      return;
    }
    setAbsError("");
    onChange({ mode: "absolute", start: startMs, end: endMs });
    setIsOpen(false);
  };

  return (
    <div className="relative" ref={panelRef}>
      {/* 触发按钮 */}
      <button
        onClick={() => setIsOpen((v) => !v)}
        className="flex items-center gap-2 px-3 py-1.5 text-sm rounded-lg border border-[var(--border-color)] bg-card hover:bg-[var(--hover-bg)] transition-colors"
      >
        <Calendar className="w-3.5 h-3.5 text-muted" />
        <span className="text-default truncate max-w-[200px]">{displayLabel}</span>
        <ChevronDown className="w-3.5 h-3.5 text-muted" />
      </button>

      {/* 下拉面板 */}
      {isOpen && (
        <div className="absolute right-0 top-full mt-1 z-50 w-[320px] rounded-lg border border-[var(--border-color)] bg-card shadow-lg">
          {/* 快捷预设 */}
          <div className="p-3 pb-2">
            <div className="text-xs font-medium text-muted mb-2">{t.timeRangePresets}</div>
            <div className="grid grid-cols-3 gap-1.5">
              {PRESET_KEYS.map((key) => (
                <button
                  key={key}
                  onClick={() => handlePresetClick(key)}
                  className={`px-2 py-1.5 text-xs rounded-md transition-colors ${
                    value.mode === "preset" && value.preset === key
                      ? "bg-primary/10 text-primary font-medium"
                      : "text-default hover:bg-[var(--hover-bg)]"
                  }`}
                >
                  {presetLabels[key]}
                </button>
              ))}
            </div>
          </div>

          <div className="border-t border-[var(--border-color)]" />

          {/* 自定义相对时间 */}
          <div className="p-3 pb-2">
            <div className="text-xs font-medium text-muted mb-2">{t.timeRangeCustomRelative}</div>
            <div className="flex items-center gap-2">
              <span className="text-xs text-muted whitespace-nowrap">{t.timeRangeLastN}</span>
              <input
                type="number"
                min={1}
                value={customValue}
                onChange={(e) => setCustomValue(Math.max(1, parseInt(e.target.value) || 1))}
                className="w-20 px-2 py-1 text-xs rounded-md border border-[var(--border-color)] bg-card text-default focus:outline-none focus:ring-1 focus:ring-primary"
              />
              <select
                value={customUnit}
                onChange={(e) => setCustomUnit(e.target.value as RelativeUnit)}
                className="px-2 py-1 text-xs rounded-md border border-[var(--border-color)] bg-card text-default focus:outline-none focus:ring-1 focus:ring-primary"
              >
                {RELATIVE_UNITS.map((u) => (
                  <option key={u.value} value={u.value}>{t[u.labelKey]}</option>
                ))}
              </select>
              <button
                onClick={handleCustomApply}
                className="px-2 py-1 text-xs rounded-md bg-primary text-white hover:bg-primary/90 transition-colors whitespace-nowrap"
              >
                {t.timeRangeApply}
              </button>
            </div>
          </div>

          <div className="border-t border-[var(--border-color)]" />

          {/* 绝对时间范围 */}
          <div className="p-3">
            <div className="text-xs font-medium text-muted mb-2">{t.timeRangeAbsolute}</div>
            <div className="space-y-2">
              <div className="flex items-center gap-2">
                <span className="text-xs text-muted w-10">{t.timeRangeStart}</span>
                <input
                  type="datetime-local"
                  value={absStart}
                  onChange={(e) => { setAbsStart(e.target.value); setAbsError(""); }}
                  className="flex-1 px-2 py-1 text-xs rounded-md border border-[var(--border-color)] bg-card text-default focus:outline-none focus:ring-1 focus:ring-primary"
                />
              </div>
              <div className="flex items-center gap-2">
                <span className="text-xs text-muted w-10">{t.timeRangeEnd}</span>
                <input
                  type="datetime-local"
                  value={absEnd}
                  onChange={(e) => { setAbsEnd(e.target.value); setAbsError(""); }}
                  className="flex-1 px-2 py-1 text-xs rounded-md border border-[var(--border-color)] bg-card text-default focus:outline-none focus:ring-1 focus:ring-primary"
                />
              </div>
              {absError && (
                <p className="text-xs text-red-500">{absError}</p>
              )}
              <div className="flex justify-end">
                <button
                  onClick={handleAbsoluteApply}
                  className="px-2 py-1 text-xs rounded-md bg-primary text-white hover:bg-primary/90 transition-colors"
                >
                  {t.timeRangeApply}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
