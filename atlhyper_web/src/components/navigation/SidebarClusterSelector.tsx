"use client";

import { Server, ChevronDown, Check } from "lucide-react";
import { useClusterStore } from "@/store/clusterStore";

interface SidebarClusterSelectorProps {
  collapsed: boolean;
  clusterMenuOpen: boolean;
  onSetClusterMenuOpen: (open: boolean) => void;
}

export function SidebarClusterSelector({
  collapsed,
  clusterMenuOpen,
  onSetClusterMenuOpen,
}: SidebarClusterSelectorProps) {
  const { clusterIds, currentClusterId, setCurrentCluster } = useClusterStore();

  if (clusterIds.length === 0) return null;

  return (
    <div className={`relative border-b border-[var(--border-color)]/20 ${collapsed ? "py-2 flex justify-center" : "px-3 py-2"}`}>
      {collapsed ? (
        <div
          className="relative"
          onMouseEnter={() => clusterIds.length > 1 && onSetClusterMenuOpen(true)}
          onMouseLeave={() => onSetClusterMenuOpen(false)}
        >
          <button className="p-2 rounded-xl hover:bg-white/5 transition-colors" title={currentClusterId}>
            <Server className="w-5 h-5 text-primary" />
          </button>
          {clusterIds.length > 1 && clusterMenuOpen && (
            <div className="absolute left-full top-0 ml-2 z-50">
              <div className="py-2 px-1 min-w-[180px] rounded-2xl border border-white/10 dark:border-white/5 bg-card/95 backdrop-blur-xl shadow-[0_8px_30px_rgb(0,0,0,0.12)] dark:shadow-[0_8px_30px_rgb(0,0,0,0.4)] ring-1 ring-black/5 dark:ring-white/10">
                <div className="px-3 py-2 text-[11px] font-semibold text-muted uppercase tracking-wider border-b border-[var(--border-color)]/30 mb-1">
                  Select Cluster
                </div>
                {clusterIds.map((id) => (
                  <button
                    key={id}
                    onClick={() => { setCurrentCluster(id); window.location.reload(); }}
                    className={`w-full flex items-center justify-between gap-2 px-3 py-2 rounded-xl text-sm transition-all ${
                      id === currentClusterId ? "text-primary bg-primary/10" : "text-secondary hover:bg-white/5"
                    }`}
                  >
                    <span className="font-mono truncate">{id}</span>
                    {id === currentClusterId && <Check className="w-4 h-4 flex-shrink-0" />}
                  </button>
                ))}
              </div>
            </div>
          )}
        </div>
      ) : (
        <div className="relative">
          <button
            onClick={() => clusterIds.length > 1 && onSetClusterMenuOpen(!clusterMenuOpen)}
            className={`w-full flex items-center gap-2.5 px-3 py-2 rounded-xl text-sm transition-all hover:bg-white/5 ${
              clusterIds.length <= 1 ? "cursor-default" : ""
            }`}
          >
            <Server className="w-4 h-4 text-primary flex-shrink-0" />
            <span className="font-mono text-secondary truncate flex-1 text-left">{currentClusterId}</span>
            {clusterIds.length > 1 && (
              <ChevronDown className={`w-4 h-4 text-muted transition-transform ${clusterMenuOpen ? "rotate-180" : ""}`} />
            )}
          </button>
          {clusterIds.length > 1 && clusterMenuOpen && (
            <div className="absolute left-0 right-0 top-full mt-1 z-50 py-1 rounded-xl border border-white/10 dark:border-white/5 bg-card/95 backdrop-blur-xl shadow-lg">
              {clusterIds.map((id) => (
                <button
                  key={id}
                  onClick={() => { setCurrentCluster(id); onSetClusterMenuOpen(false); window.location.reload(); }}
                  className={`w-full flex items-center justify-between gap-2 px-3 py-2 text-sm transition-all ${
                    id === currentClusterId ? "text-primary" : "text-secondary hover:bg-white/5"
                  }`}
                >
                  <span className="font-mono truncate">{id}</span>
                  {id === currentClusterId && <Check className="w-4 h-4 flex-shrink-0" />}
                </button>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
