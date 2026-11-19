"use client";

/**
 * ECharts 实例的生命周期，各图表组件共用：
 * - 容器挂上时 init、卸下时 dispose（容器可以按条件渲染：用回调 ref 跟着元素走）
 * - 容器尺寸变化时 resize：ResizeObserver，侧栏收起、布局变化也跟得上（⛔ 不只听窗口尺寸）
 * - 主题切换（<html> 的 dark class）时调用 onThemeChange 重新上色
 *
 * 用法：
 *   const { containerRef, chart } = useEChart({ onThemeChange: (c) => c.setOption(themeColors()) });
 *   useEffect(() => { if (chart) chart.setOption(baseOption) }, [chart]);       // 基础配置
 *   useEffect(() => { if (chart) chart.setOption(dataOption) }, [chart, data]); // 随数据更新
 *   return <div ref={containerRef} />;
 */
import { useCallback, useEffect, useRef, useState } from "react";
import * as echarts from "echarts";

export interface UseEChartOptions {
  /** 主题切换后调用（实例已存在时） */
  onThemeChange?: (chart: echarts.ECharts) => void;
}

export function useEChart({ onThemeChange }: UseEChartOptions = {}) {
  const [chart, setChart] = useState<echarts.ECharts | null>(null);

  // 回调存最新的一份：容器的 ref 回调不随它重建（重建会让图表重新 init）
  const onThemeChangeRef = useRef(onThemeChange);
  useEffect(() => {
    onThemeChangeRef.current = onThemeChange;
  });

  const containerRef = useCallback((element: HTMLDivElement | null) => {
    if (!element) return;
    const instance = echarts.init(element);
    setChart(instance);

    const resizeObserver = new ResizeObserver(() => instance.resize());
    resizeObserver.observe(element);
    const themeObserver = new MutationObserver(() => onThemeChangeRef.current?.(instance));
    themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ["class"] });

    return () => {
      resizeObserver.disconnect();
      themeObserver.disconnect();
      instance.dispose();
      setChart(null);
    };
  }, []);

  return { containerRef, chart };
}
