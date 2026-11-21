/**
 * 图表里与主题有关的颜色（随 <html> 的 dark class 变化）：所有 ECharts 图表共用这一份。
 * 主题切换后由 useEChart 的 onThemeChange 重新取色上色。
 */
export function getThemeColors() {
  const isDark = document.documentElement.classList.contains("dark");
  return {
    textColor: isDark ? "#9ca3af" : "#6b7280",
    lineColor: isDark ? "#374151" : "#e5e7eb",
    splitLineColor: isDark ? "#1f2937" : "#f3f4f6",
    tooltipBg: isDark ? "#1f2937" : "#fff",
    tooltipBorder: isDark ? "#374151" : "#e5e7eb",
    tooltipText: isDark ? "#e5e7eb" : "#111827",
  };
}

export type ChartThemeColors = ReturnType<typeof getThemeColors>;
