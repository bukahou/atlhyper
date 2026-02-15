"use client";

import { Loader2 } from "lucide-react";
import { useI18n } from "@/i18n/context";

interface GlobalSettingsCardProps {
  enabled: boolean;
  timeout: number;
  isAdmin: boolean;
  saving: boolean;
  onToggleEnabled: () => void;
  onTimeoutChange: (value: number) => void;
  onSaveTimeout: () => void;
}

export function GlobalSettingsCard({
  enabled,
  timeout,
  isAdmin,
  saving,
  onToggleEnabled,
  onTimeoutChange,
  onSaveTimeout,
}: GlobalSettingsCardProps) {
  const { t } = useI18n();
  const aiT = t.aiSettingsPage;

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-6">
      <h3 className="text-lg font-medium text-default mb-4">{aiT.globalSettings}</h3>
      <div className="flex flex-wrap items-center gap-6">
        {/* Enable Toggle */}
        <div className="flex items-center gap-3">
          <span className="text-sm text-muted">{aiT.aiFeature}:</span>
          <button
            onClick={onToggleEnabled}
            disabled={!isAdmin || saving}
            className={`relative w-12 h-6 rounded-full transition-colors ${
              enabled ? "bg-green-500" : "bg-gray-300 dark:bg-gray-600"
            } ${!isAdmin ? "opacity-50 cursor-not-allowed" : "cursor-pointer"}`}
          >
            <span
              className={`absolute top-1 left-1 w-4 h-4 rounded-full bg-white transition-transform ${
                enabled ? "translate-x-6" : "translate-x-0"
              }`}
            />
          </button>
          <span className={`text-sm ${enabled ? "text-green-600" : "text-muted"}`}>
            {enabled ? aiT.enabled : aiT.disabled}
          </span>
        </div>

        {/* Tool Timeout */}
        <div className="flex items-center gap-3">
          <span className="text-sm text-muted">{aiT.toolTimeout}:</span>
          <input
            type="number"
            value={timeout}
            onChange={(e) => onTimeoutChange(parseInt(e.target.value) || 30)}
            min={5}
            max={300}
            disabled={!isAdmin}
            className="w-20 px-2 py-1 rounded border border-[var(--border-color)] bg-[var(--bg-primary)] text-default text-sm"
          />
          <span className="text-sm text-muted">{aiT.seconds}</span>
          {isAdmin && (
            <button
              onClick={onSaveTimeout}
              disabled={saving}
              className="px-3 py-1 text-sm rounded bg-violet-600 text-white hover:bg-violet-700 disabled:opacity-50"
            >
              {saving ? <Loader2 className="w-4 h-4 animate-spin" /> : aiT.save}
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
