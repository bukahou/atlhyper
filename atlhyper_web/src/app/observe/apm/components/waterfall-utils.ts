import type { Span } from "@/types/model/apm";

export const SERVICE_COLORS = [
  "#60a5fa", "#34d399", "#fbbf24", "#a78bfa",
  "#f87171", "#22d3ee", "#fb923c", "#818cf8",
];

export interface SpanNode {
  span: Span;
  children: SpanNode[];
  depth: number;
}

export function buildSpanTree(spans: Span[]): SpanNode[] {
  const spanMap = new Map<string, SpanNode>();
  const roots: SpanNode[] = [];

  for (const span of spans) {
    spanMap.set(span.spanId, { span, children: [], depth: 0 });
  }

  for (const span of spans) {
    const node = spanMap.get(span.spanId)!;
    if (span.parentSpanId && spanMap.has(span.parentSpanId)) {
      spanMap.get(span.parentSpanId)!.children.push(node);
    } else {
      roots.push(node);
    }
  }

  function setDepth(node: SpanNode, depth: number) {
    node.depth = depth;
    node.children.forEach((c) => setDepth(c, depth + 1));
  }
  roots.forEach((r) => setDepth(r, 0));
  return roots;
}

export function flattenTree(nodes: SpanNode[], collapsed: Set<string>): SpanNode[] {
  const result: SpanNode[] = [];
  function walk(node: SpanNode) {
    result.push(node);
    if (!collapsed.has(node.span.spanId)) {
      node.children.forEach(walk);
    }
  }
  nodes.forEach(walk);
  return result;
}

export function countDescendants(node: SpanNode): number {
  let count = node.children.length;
  for (const child of node.children) count += countDescendants(child);
  return count;
}
