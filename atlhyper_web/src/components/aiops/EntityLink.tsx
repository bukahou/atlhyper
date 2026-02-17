"use client";

import { Box, Server, Network, Globe } from "lucide-react";
import { useEntityDetail } from "@/components/common/entity-detail";
import type { EntityType } from "@/types/entity-detail";

const TYPE_ICONS: Record<string, typeof Box> = {
  pod: Box,
  node: Server,
  service: Network,
  ingress: Globe,
};

/**
 * 解析 entityKey 格式:
 * - "namespace/type/name" (标准格式)
 * - "cluster/clusterId/node/name" (节点格式)
 */
function parseEntityKey(entityKey: string): {
  namespace: string;
  type: string;
  name: string;
} {
  const parts = entityKey.split("/");
  if (parts.length >= 4 && parts[0] === "cluster") {
    // cluster/{clusterId}/node/{name}
    return { namespace: "", type: parts[2], name: parts[3] };
  }
  if (parts.length >= 3) {
    // namespace/type/name
    return { namespace: parts[0], type: parts[1], name: parts.slice(2).join("/") };
  }
  if (parts.length === 2) {
    return { namespace: "", type: parts[0], name: parts[1] };
  }
  return { namespace: "", type: "unknown", name: entityKey };
}

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
