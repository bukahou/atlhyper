"use client";

import { useState, useRef, useEffect } from "react";
import { Server, ChevronDown, Check } from "lucide-react";
import { useClusterStore } from "@/store/clusterStore";

export function ClusterSelector() {
  const { clusterIds, currentClusterId, setCurrentCluster } = useClusterStore();
  const [isOpen, setIsOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  // 点击外部关闭下拉框
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };

    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  // 只有一个集群时不显示选择器
  if (clusterIds.length <= 1) {
    return (
      <div className="flex items-center gap-2 px-3 py-2 text-sm text-muted">
        <Server className="w-4 h-4" />
        <span className="font-mono">{currentClusterId}</span>
      </div>
    );
  }

  const handleSelect = (id: string) => {
    setCurrentCluster(id);
    setIsOpen(false);
    // 刷新页面以加载新集群的数据
    window.location.reload();
  };

  return (
    <div className="relative" ref={dropdownRef}>
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium text-secondary hover-bg transition-colors"
      >
        <Server className="w-4 h-4 text-primary" />
        <span className="font-mono max-w-[120px] truncate">{currentClusterId}</span>
        <ChevronDown className={`w-4 h-4 transition-transform ${isOpen ? "rotate-180" : ""}`} />
      </button>

      {isOpen && (
        <div className="absolute left-0 mt-2 w-56 dropdown-menu rounded-lg shadow-lg border z-50">
          <div className="px-3 py-2 border-b border-[var(--border-color)]">
            <p className="text-xs font-medium text-muted uppercase">Select Cluster</p>
          </div>
          <div className="py-1 max-h-60 overflow-y-auto">
            {clusterIds.map((id) => (
              <button
                key={id}
                onClick={() => handleSelect(id)}
                className={`w-full px-3 py-2 text-left text-sm flex items-center justify-between hover-bg ${
                  id === currentClusterId ? "text-primary" : "text-secondary"
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
  );
}
