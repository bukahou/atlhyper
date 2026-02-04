"use client";

import { CheckCircle, XCircle, AlertTriangle, Pause, Activity } from "lucide-react";

interface RolloutBadgeProps {
  badge: string;
}

export function RolloutBadge({ badge }: RolloutBadgeProps) {
  const config: Record<string, { bg: string; text: string; icon: React.ReactNode }> = {
    Available: { bg: "bg-green-100 dark:bg-green-900/30", text: "text-green-600 dark:text-green-400", icon: <CheckCircle className="w-3 h-3" /> },
    Progressing: { bg: "bg-blue-100 dark:bg-blue-900/30", text: "text-blue-600 dark:text-blue-400", icon: <Activity className="w-3 h-3" /> },
    Failed: { bg: "bg-red-100 dark:bg-red-900/30", text: "text-red-600 dark:text-red-400", icon: <XCircle className="w-3 h-3" /> },
    ReplicaFailure: { bg: "bg-red-100 dark:bg-red-900/30", text: "text-red-600 dark:text-red-400", icon: <AlertTriangle className="w-3 h-3" /> },
    Paused: { bg: "bg-yellow-100 dark:bg-yellow-900/30", text: "text-yellow-600 dark:text-yellow-400", icon: <Pause className="w-3 h-3" /> },
  };
  const c = config[badge] || { bg: "bg-gray-100 dark:bg-gray-800", text: "text-gray-600 dark:text-gray-400", icon: null };
  return (
    <span className={`inline-flex items-center gap-1 px-2 py-1 ${c.bg} ${c.text} text-xs rounded`}>
      {c.icon} {badge}
    </span>
  );
}
