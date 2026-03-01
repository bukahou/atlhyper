import type { MeshTopologyResponse } from "@/types/mesh";

export const NODE_RADIUS = 32;
export const SVG_HEIGHT = 500;
export const MIN_ZOOM = 0.2;
export const MAX_ZOOM = 4;

// Topological sort for layered layout
export function computeNodeLayers(topology: MeshTopologyResponse): string[][] {
  const nodeIds = topology.nodes.map(n => n.id);
  const inDegree: Record<string, number> = {};
  const outEdges: Record<string, string[]> = {};
  nodeIds.forEach(id => { inDegree[id] = 0; outEdges[id] = []; });
  topology.edges.forEach(edge => {
    if (nodeIds.includes(edge.source) && nodeIds.includes(edge.target)) {
      inDegree[edge.target]++;
      outEdges[edge.source].push(edge.target);
    }
  });
  const level: Record<string, number> = {};
  const queue: string[] = [];
  nodeIds.forEach(id => { if (inDegree[id] === 0) { level[id] = 0; queue.push(id); } });
  while (queue.length > 0) {
    const current = queue.shift()!;
    outEdges[current].forEach(next => {
      const newLevel = level[current] + 1;
      if (level[next] === undefined || level[next] < newLevel) level[next] = newLevel;
      inDegree[next]--;
      if (inDegree[next] === 0) queue.push(next);
    });
  }
  nodeIds.forEach(id => { if (level[id] === undefined) level[id] = 0; });
  const maxLevel = Math.max(...Object.values(level), 0);
  const layers: string[][] = Array.from({ length: maxLevel + 1 }, () => []);
  nodeIds.forEach(id => layers[level[id]].push(id));
  return layers;
}

// Group nodes by namespace
export function groupNodesByNamespace(topology: MeshTopologyResponse): Record<string, string[]> {
  const groups: Record<string, string[]> = {};
  topology.nodes.forEach(n => {
    if (!groups[n.namespace]) groups[n.namespace] = [];
    groups[n.namespace].push(n.id);
  });
  return groups;
}

export interface LayoutResult {
  positions: Record<string, { x: number; y: number }>;
  bounds: Record<string, { minX: number; minY: number; maxX: number; maxY: number }>;
}

// Namespace swim-lane layout: each namespace gets a horizontal band,
// nodes within each band are placed by topological layer (left->right)
export function computeSwimLaneLayout(
  containerWidth: number,
  nodeLayers: string[][],
  nsGroups: Record<string, string[]>,
  sortedNamespaces: string[],
): LayoutResult {
  const paddingX = 80, paddingY = 50, lanePadding = 35, laneGap = 20, nodeGapY = 85;
  const usableWidth = containerWidth - paddingX * 2;
  const layerCount = nodeLayers.length;
  const layerGapX = layerCount > 1 ? usableWidth / (layerCount - 1) : 0;

  const positions: Record<string, { x: number; y: number }> = {};
  const bounds: Record<string, { minX: number; minY: number; maxX: number; maxY: number }> = {};
  let currentY = paddingY;

  sortedNamespaces.forEach(ns => {
    const nsNodeIds = nsGroups[ns];
    const nodesPerLayer: Record<number, string[]> = {};
    nsNodeIds.forEach(nodeId => {
      const layerIdx = nodeLayers.findIndex(layer => layer.includes(nodeId));
      if (layerIdx >= 0) {
        if (!nodesPerLayer[layerIdx]) nodesPerLayer[layerIdx] = [];
        nodesPerLayer[layerIdx].push(nodeId);
      }
    });
    const maxNodesInAnyLayer = Math.max(...Object.values(nodesPerLayer).map(a => a.length), 1);
    const laneHeight = (maxNodesInAnyLayer - 1) * nodeGapY + lanePadding * 2;
    const laneCenterY = currentY + laneHeight / 2;

    Object.entries(nodesPerLayer).forEach(([li, nodeIds]) => {
      const layerIdx = parseInt(li);
      const x = layerCount > 1 ? paddingX + layerIdx * layerGapX : containerWidth / 2;
      const startY = laneCenterY - ((nodeIds.length - 1) * nodeGapY) / 2;
      nodeIds.forEach((nodeId, idx) => { positions[nodeId] = { x, y: startY + idx * nodeGapY }; });
    });

    bounds[ns] = { minX: paddingX - lanePadding, minY: currentY, maxX: containerWidth - paddingX + lanePadding, maxY: currentY + laneHeight };
    currentY += laneHeight + laneGap;
  });

  return { positions, bounds };
}

// Auto-fit zoom to show all content
export function computeFitZoom(
  positions: Record<string, { x: number; y: number }>,
  containerWidth: number,
): { zoom: number; viewOrigin: { x: number; y: number } } | null {
  const posValues = Object.values(positions);
  if (posValues.length === 0) return null;
  const pad = NODE_RADIUS + 40;
  const minX = Math.min(...posValues.map(p => p.x)) - pad;
  const maxX = Math.max(...posValues.map(p => p.x)) + pad;
  const minY = Math.min(...posValues.map(p => p.y)) - pad;
  const maxY = Math.max(...posValues.map(p => p.y)) + pad;
  const contentW = maxX - minX;
  const contentH = maxY - minY;
  const fitZoom = Math.min(containerWidth / contentW, SVG_HEIGHT / contentH, MAX_ZOOM);
  return {
    zoom: fitZoom,
    viewOrigin: {
      x: minX - (containerWidth / fitZoom - contentW) / 2,
      y: minY - (SVG_HEIGHT / fitZoom - contentH) / 2,
    },
  };
}

// Edge path (cubic bezier)
export function getEdgePath(
  sourceId: string,
  targetId: string,
  positions: Record<string, { x: number; y: number }>,
): string {
  const source = positions[sourceId], target = positions[targetId];
  if (!source || !target) return "";
  const startX = source.x + NODE_RADIUS, startY = source.y, endX = target.x - NODE_RADIUS, endY = target.y;
  const midX = (startX + endX) / 2;
  return `M ${startX} ${startY} C ${midX} ${startY}, ${midX} ${endY}, ${endX} ${endY}`;
}

// Edge label position (midpoint)
export function getEdgeLabelPos(
  sourceId: string,
  targetId: string,
  positions: Record<string, { x: number; y: number }>,
): { x: number; y: number } {
  const source = positions[sourceId], target = positions[targetId];
  if (!source || !target) return { x: 0, y: 0 };
  return { x: (source.x + target.x) / 2, y: (source.y + target.y) / 2 };
}
