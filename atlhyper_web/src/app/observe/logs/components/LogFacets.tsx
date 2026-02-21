"use client";

import { useState } from "react";
import { ChevronDown, ChevronRight } from "lucide-react";
import type { LogFacet } from "@/types/model/log";
import type { LogTranslations } from "@/types/i18n";
import { shortScopeName } from "@/types/model/log";

interface LogFacetsProps {
  services: LogFacet[];
  severities: LogFacet[];
  scopes: LogFacet[];
  selectedServices: string[];
  selectedSeverities: string[];
  selectedScopes: string[];
  onServicesChange: (services: string[]) => void;
  onSeveritiesChange: (severities: string[]) => void;
  onScopesChange: (scopes: string[]) => void;
  t: LogTranslations;
}

interface FacetGroupProps {
  title: string;
  items: LogFacet[];
  selected: string[];
  onChange: (selected: string[]) => void;
  formatLabel?: (value: string) => string;
  tooltipLabel?: (value: string) => string;
}

function FacetGroup({ title, items, selected, onChange, formatLabel, tooltipLabel }: FacetGroupProps) {
  const [collapsed, setCollapsed] = useState(false);

  const toggle = (value: string) => {
    if (selected.includes(value)) {
      onChange(selected.filter((v) => v !== value));
    } else {
      onChange([...selected, value]);
    }
  };

  return (
    <div className="mb-4">
      <button
        onClick={() => setCollapsed(!collapsed)}
        className="flex items-center gap-1 w-full text-left mb-1.5"
      >
        {collapsed
          ? <ChevronRight className="w-3.5 h-3.5 text-muted" />
          : <ChevronDown className="w-3.5 h-3.5 text-muted" />
        }
        <span className="text-xs font-semibold text-default">{title}</span>
      </button>

      {!collapsed && (
        <div className="space-y-0.5 ml-1">
          {items.map((item) => {
            const isSelected = selected.includes(item.value);
            const label = formatLabel ? formatLabel(item.value) : item.value;
            const tooltip = tooltipLabel ? tooltipLabel(item.value) : undefined;

            return (
              <label
                key={item.value}
                className="flex items-center gap-2 py-1 px-1 rounded cursor-pointer hover:bg-[var(--hover-bg)] transition-colors"
                title={tooltip}
              >
                <input
                  type="checkbox"
                  checked={isSelected}
                  onChange={() => toggle(item.value)}
                  className="w-3.5 h-3.5 rounded border-[var(--border-color)] accent-primary"
                />
                <span className="text-xs text-default truncate flex-1">{label}</span>
                <span className="text-[10px] text-muted tabular-nums">{item.count}</span>
              </label>
            );
          })}
        </div>
      )}
    </div>
  );
}

export function LogFacets({
  services, severities, scopes,
  selectedServices, selectedSeverities, selectedScopes,
  onServicesChange, onSeveritiesChange, onScopesChange,
  t,
}: LogFacetsProps) {
  return (
    <div className="w-56 flex-shrink-0">
      <div className="sticky top-4 space-y-1">
        <FacetGroup
          title={t.facetServices}
          items={services}
          selected={selectedServices}
          onChange={onServicesChange}
        />

        <FacetGroup
          title={t.facetSeverities}
          items={severities}
          selected={selectedSeverities}
          onChange={onSeveritiesChange}
        />

        <FacetGroup
          title={t.facetScopes}
          items={scopes}
          selected={selectedScopes}
          onChange={onScopesChange}
          formatLabel={shortScopeName}
          tooltipLabel={(v) => v}
        />
      </div>
    </div>
  );
}
