"use client";

import { useState } from "react";
import { ChevronDown, ChevronRight, Wrench, MessageSquare } from "lucide-react";
import { useI18n } from "@/i18n/context";

interface InvestigationStep {
  round: number;
  thinking: string;
  toolCalls: { name: string; args: string; result: string }[];
}

interface InvestigationTimelineProps {
  stepsJson: string;
}

export function InvestigationTimeline({ stepsJson }: InvestigationTimelineProps) {
  const { t } = useI18n();
  const aiT = t.aiops.ai;
  const [expanded, setExpanded] = useState(false);

  let steps: InvestigationStep[] = [];
  try {
    steps = JSON.parse(stepsJson);
  } catch {
    // JSON 解析失败，保持 steps 为空数组
  }

  if (steps.length === 0) {
    return (
      <div className="mt-3 text-xs text-muted text-center py-2">
        {aiT.investigationSteps}
      </div>
    );
  }

  return (
    <div className="mt-3">
      <button
        onClick={() => setExpanded(!expanded)}
        className="flex items-center gap-1 text-xs text-blue-500 hover:text-blue-600 transition-colors"
      >
        {expanded ? <ChevronDown className="w-3 h-3" /> : <ChevronRight className="w-3 h-3" />}
        {expanded ? aiT.collapseSteps : aiT.expandSteps}
        <span className="text-muted ml-1">({steps.length} {aiT.round})</span>
      </button>

      {expanded && (
        <div className="mt-2 space-y-3 ml-2 border-l-2 border-blue-200 dark:border-blue-800 pl-3">
          {steps.map((step) => (
            <div key={step.round} className="text-xs">
              <div className="flex items-center gap-2 mb-1">
                <span className="font-mono font-bold text-blue-500">R{step.round}</span>
                <span className="text-muted">{aiT.round} {step.round}</span>
              </div>

              {step.thinking && (
                <div className="flex items-start gap-1.5 mb-1 text-muted">
                  <MessageSquare className="w-3 h-3 mt-0.5 shrink-0" />
                  <p className="line-clamp-2">{step.thinking}</p>
                </div>
              )}

              {step.toolCalls.map((tc, i) => (
                <div key={i} className="flex items-start gap-1.5 ml-4 text-muted">
                  <Wrench className="w-3 h-3 mt-0.5 shrink-0 text-violet-500" />
                  <div>
                    <span className="font-mono text-default">{tc.name}</span>
                    {tc.result && (
                      <p className="text-[10px] text-muted line-clamp-1 mt-0.5">{tc.result}</p>
                    )}
                  </div>
                </div>
              ))}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
