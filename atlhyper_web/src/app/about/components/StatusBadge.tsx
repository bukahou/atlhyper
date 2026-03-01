"use client";

import { CheckCircle2, Clock } from "lucide-react";

export function StatusBadge({ status, label }: { status: "done" | "planned"; label: string }) {
  if (status === "done") {
    return (
      <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-emerald-500/15 text-emerald-500 flex-shrink-0">
        <CheckCircle2 className="w-3 h-3" />
        {label}
      </span>
    );
  }
  return (
    <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-blue-500/15 text-blue-500 flex-shrink-0">
      <Clock className="w-3 h-3" />
      {label}
    </span>
  );
}
