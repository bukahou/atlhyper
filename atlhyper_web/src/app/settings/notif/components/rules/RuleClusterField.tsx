"use client";

import { useState } from "react";
import { Plus } from "lucide-react";
import type { NotifyRulesTabTranslations } from "@/types/i18n-notify";
import { ChipMultiSelect } from "../shared/ChipMultiSelect";
import { HINT_CLASS, INPUT_CLASS, LABEL_CLASS } from "../shared/form-styles";

interface RuleClusterFieldProps {
  /** 已接入的集群 */
  knownClusterIds: string[];
  selected: string[];
  onChange: (selected: string[]) => void;
  t: NotifyRulesTabTranslations;
}

/**
 * 规则的集群条件：从已接入的集群里点选，也可以手填一个 ID。
 *
 * 手填是为了给此刻不在线的集群配规则 —— 下拉里只有正在上报的集群。不选 = 全部集群。
 */
export function RuleClusterField({ knownClusterIds, selected, onChange, t }: RuleClusterFieldProps) {
  const [typedClusterId, setTypedClusterId] = useState("");
  // 已选但不在已接入列表里的（手填的、已离线的）也要显示出来，否则没法取消
  const clusterOptions = [...new Set([...knownClusterIds, ...selected])];

  const addTypedCluster = () => {
    const clusterId = typedClusterId.trim();
    if (clusterId === "") return;
    if (!selected.includes(clusterId)) onChange([...selected, clusterId]);
    setTypedClusterId("");
  };

  return (
    <div>
      <span className={LABEL_CLASS}>{t.clusters}</span>
      <ChipMultiSelect
        options={clusterOptions.map((clusterId) => ({ value: clusterId, label: clusterId }))}
        selected={selected}
        onChange={onChange}
        emptyText={t.allClusters}
      />
      <div className="flex gap-2 mt-2">
        <input
          type="text"
          value={typedClusterId}
          onChange={(event) => setTypedClusterId(event.target.value)}
          onKeyDown={(event) => {
            // 回车是「添加这个集群」，不是提交整个表单
            if (event.key === "Enter") {
              event.preventDefault();
              addTypedCluster();
            }
          }}
          placeholder={t.addClusterPlaceholder}
          aria-label={t.addClusterPlaceholder}
          autoComplete="off"
          spellCheck={false}
          className={`${INPUT_CLASS} font-mono`}
        />
        <button
          type="button"
          onClick={addTypedCluster}
          disabled={typedClusterId.trim() === ""}
          className="px-3 py-2 rounded-lg border border-[var(--border-color)] text-default hover-bg disabled:opacity-50 disabled:cursor-not-allowed"
          aria-label={t.addClusterPlaceholder}
        >
          <Plus className="w-4 h-4" />
        </button>
      </div>
      <p className={HINT_CLASS}>{t.clustersHint}</p>
    </div>
  );
}
