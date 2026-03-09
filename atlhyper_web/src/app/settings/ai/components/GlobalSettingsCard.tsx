"use client";

import { Loader2 } from "lucide-react";
import { useI18n } from "@/i18n/context";

interface GlobalSettingsCardProps {
  timeout: number;
  isAdmin: boolean;
  saving: boolean;
  onTimeoutChange: (value: number) => void;
  onSaveTimeout: () => void;
}

export function GlobalSettingsCard({
  timeout,
  isAdmin,
  saving,
  onTimeoutChange,
  onSaveTimeout,
}: GlobalSettingsCardProps) {
  const { t } = useI18n();
  const aiT = t.aiSettingsPage;

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-6">
      <h3 className="text-lg font-medium text-default mb-4">{aiT.globalSettings}</h3>
      <div className="flex flex-wrap items-center gap-6">
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
