"use client";

import { Wrench } from "lucide-react";
import { StreamSegment } from "./types";

interface ToolCallBlockProps {
  segment: StreamSegment;
  tokens?: number;
}

export function ToolCallBlock({ segment, tokens }: ToolCallBlockProps) {
  // 解析参数用于显示
  let paramsDisplay = "";
  if (segment.params) {
    try {
      const p = JSON.parse(segment.params);
      paramsDisplay = Object.entries(p)
        .map(([k, v]) => `${k}=${v}`)
        .join(", ");
    } catch {
      paramsDisplay = segment.params;
    }
  }

  return (
    <div className="my-2 rounded-lg border border-[var(--border-color)] bg-[var(--background)] text-sm overflow-hidden">
      <div className="flex items-center gap-2 px-3 py-2 bg-blue-50 dark:bg-blue-900/20">
        <Wrench className="w-3.5 h-3.5 text-blue-600 dark:text-blue-400" />
        <span className="font-mono text-blue-700 dark:text-blue-300">
          {segment.tool}
        </span>
        {paramsDisplay && (
          <span className="text-muted text-xs truncate">({paramsDisplay})</span>
        )}
        {tokens !== undefined && (
          <span className="ml-auto text-[11px] text-amber-600 dark:text-amber-400 font-medium whitespace-nowrap">
            {tokens.toLocaleString()} tokens
          </span>
        )}
      </div>
    </div>
  );
}
