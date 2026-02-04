"use client";

interface InfoCardProps {
  label: string;
  value: React.ReactNode;
}

export function InfoCard({ label, value }: InfoCardProps) {
  return (
    <div className="bg-[var(--background)] rounded-lg p-3">
      <div className="text-xs text-muted mb-1">{label}</div>
      <div className="text-sm text-default font-medium break-all">{value}</div>
    </div>
  );
}
