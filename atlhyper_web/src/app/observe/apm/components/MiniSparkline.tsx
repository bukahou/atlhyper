"use client";

interface MiniSparklineProps {
  data: number[];
  type: "line" | "bar";
  color: string;
  width?: number;
  height?: number;
}

export function MiniSparkline({
  data,
  type,
  color,
  width = 80,
  height = 24,
}: MiniSparklineProps) {
  if (data.length === 0) return null;

  const padding = 1;
  const innerW = width - padding * 2;
  const innerH = height - padding * 2;
  const max = Math.max(...data, 1);

  if (type === "bar") {
    const barGap = 1;
    const barWidth = Math.max(1, (innerW - barGap * (data.length - 1)) / data.length);
    return (
      <svg width={width} height={height} className="inline-block align-middle">
        {data.map((v, i) => {
          const barH = Math.max(1, (v / max) * innerH);
          const x = padding + i * (barWidth + barGap);
          const y = padding + innerH - barH;
          return (
            <rect key={i} x={x} y={y} width={barWidth} height={barH} fill={color} rx={0.5} opacity={0.8} />
          );
        })}
      </svg>
    );
  }

  // Line type with area fill
  const points = data.map((v, i) => ({
    x: padding + (i / Math.max(data.length - 1, 1)) * innerW,
    y: padding + innerH - (v / max) * innerH,
  }));

  const linePath = points.map((p, i) => `${i === 0 ? "M" : "L"} ${p.x} ${p.y}`).join(" ");
  const areaPath = `${linePath} L ${points[points.length - 1].x} ${padding + innerH} L ${points[0].x} ${padding + innerH} Z`;

  return (
    <svg width={width} height={height} className="inline-block align-middle">
      <path d={areaPath} fill={color} opacity={0.15} />
      <path d={linePath} fill="none" stroke={color} strokeWidth={1.5} />
    </svg>
  );
}
