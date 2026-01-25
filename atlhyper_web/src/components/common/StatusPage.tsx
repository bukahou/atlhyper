import { type LucideIcon } from "lucide-react";

interface StatusPageProps {
  icon: LucideIcon;
  title: string;
  description?: string;
  code?: string;
  action?: {
    label: string;
    onClick: () => void;
  };
}

export function StatusPage({ icon: Icon, title, description, code, action }: StatusPageProps) {
  return (
    <div className="relative flex flex-col items-center justify-center h-full select-none overflow-hidden">
      {/* 点阵网格背景 */}
      <div
        className="absolute inset-0 opacity-[0.4] dark:opacity-[0.15]"
        style={{
          backgroundImage: "radial-gradient(circle, currentColor 1px, transparent 1px)",
          backgroundSize: "24px 24px",
          color: "var(--text-muted, #94a3b8)",
        }}
      />

      {/* 中心渐变光晕 */}
      <div
        className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[500px] h-[500px] rounded-full opacity-30 dark:opacity-20 blur-3xl pointer-events-none"
        style={{
          background: "radial-gradient(circle, var(--color-primary, #3b82f6) 0%, transparent 70%)",
        }}
      />

      {/* 内容 */}
      <div className="relative z-10 flex flex-col items-center gap-4 max-w-sm text-center">
        {/* Icon */}
        <div className="w-20 h-20 rounded-2xl bg-card backdrop-blur-sm border border-[var(--border-color)] flex items-center justify-center shadow-lg">
          <Icon className="w-10 h-10 text-muted" />
        </div>

        {/* Code (e.g. 404) */}
        {code && (
          <span className="text-6xl font-extrabold opacity-20 text-muted">{code}</span>
        )}

        {/* Title */}
        <h2 className="text-xl font-semibold text-default">{title}</h2>

        {/* Description */}
        {description && (
          <p className="text-sm text-muted leading-relaxed">{description}</p>
        )}

        {/* Action button */}
        {action && (
          <button
            onClick={action.onClick}
            className="mt-2 px-5 py-2.5 bg-primary hover:bg-primary-hover text-white text-sm font-medium rounded-lg transition-colors shadow-md"
          >
            {action.label}
          </button>
        )}
      </div>
    </div>
  );
}
