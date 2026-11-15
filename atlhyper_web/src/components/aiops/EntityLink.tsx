"use client";

import { Box, CalendarClock, Database, Globe, Layers, LayoutGrid, Network, Play, Server } from "lucide-react";
import { useEntityDetail } from "@/components/common/entity-detail";
import { parseEntityKey } from "@/lib/entity-key";
import type { EntityType } from "@/types/entity-detail";

// 实体（pod / node / service / ingress）与工作负载主体（事件按主体收敛后，主体键也会出现在这里）
const TYPE_ICONS: Record<string, typeof Box> = {
  pod: Box,
  node: Server,
  service: Network,
  ingress: Globe,
  deployment: Layers,
  statefulset: Database,
  daemonset: LayoutGrid,
  job: Play,
  cronjob: CalendarClock,
};

interface EntityLinkProps {
  entityKey: string;
  showType?: boolean;
}

export function EntityLink({ entityKey, showType = true }: EntityLinkProps) {
  const { namespace, type, name } = parseEntityKey(entityKey);
  const Icon = TYPE_ICONS[type] ?? Box;
  const { openEntityDetail } = useEntityDetail();

  const handleClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    openEntityDetail({ type: type as EntityType, name, namespace });
  };

  return (
    <span
      role="button"
      tabIndex={0}
      onClick={handleClick}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          handleClick(e as unknown as React.MouseEvent);
        }
      }}
      className="inline-flex items-center gap-1.5 text-sm text-blue-600 dark:text-blue-400 hover:underline cursor-pointer"
      title={entityKey}
    >
      {showType && <Icon className="w-3.5 h-3.5 flex-shrink-0" />}
      <span className="truncate max-w-[200px]">{name}</span>
    </span>
  );
}
