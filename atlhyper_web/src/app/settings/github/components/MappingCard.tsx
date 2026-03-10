"use client";

import { useState } from "react";
import { useI18n } from "@/i18n/context";
import { GitBranch, Check, CheckCheck, Pencil, Plus, Trash2, X } from "lucide-react";
import type { MockRepoMapping } from "@/mock/github/data";

interface MappingCardProps {
  mappings: MockRepoMapping[];
  connected: boolean;
  namespaces: string[];
  deployments: { name: string; namespace: string; image: string }[];
  repoDirs: Record<string, string[]>;
  repoNamespaces: Record<string, string[]>;
  onUpdateMapping: (id: number, field: string, value: string) => void;
  onConfirmMapping: (id: number) => void;
  onConfirmAll: () => void;
  onAddMapping: (repo: string) => void;
  onDeleteMapping: (id: number) => void;
  onAddRepoNamespace: (repo: string, ns: string) => void;
  onRemoveRepoNamespace: (repo: string, ns: string) => void;
}

const selectClass =
  "w-full px-2 py-1.5 text-sm rounded-lg border border-[var(--border-color)] bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100";

// ---- 单行映射 ----
function MappingRow({
  mapping,
  repoNs,
  deployments,
  dirs,
  onUpdate,
  onConfirm,
  onDelete,
}: {
  mapping: MockRepoMapping;
  repoNs: string[];
  deployments: { name: string; namespace: string; image: string }[];
  dirs: string[];
  onUpdate: (id: number, field: string, value: string) => void;
  onConfirm: (id: number) => void;
  onDelete: (id: number) => void;
}) {
  const { t } = useI18n();
  const gt = t.githubPage;
  const [editing, setEditing] = useState(false);

  const isEditable = !mapping.confirmed || editing;

  // 只显示该仓库配置的 NS（已通过头部缩小范围）
  const filteredDeployments = deployments.filter(
    (d) => d.namespace === mapping.namespace
  );

  const matchedDeploy = deployments.find(
    (d) => d.name === mapping.deployment && d.namespace === mapping.namespace
  );
  const currentImage = matchedDeploy
    ? matchedDeploy.image.split(":").pop() || ""
    : "";

  const handleConfirm = () => {
    onConfirm(mapping.id);
    setEditing(false);
  };

  return (
    <div className="flex items-center gap-3 py-3 border-b border-[var(--border-color)] last:border-0">
      {/* Namespace — 只从仓库配置的 NS 中选择 */}
      <div className="flex-1 min-w-0">
        <label className="block text-xs text-muted mb-1">
          {gt.namespaceCol}
        </label>
        {isEditable ? (
          <select
            value={mapping.namespace}
            onChange={(e) => onUpdate(mapping.id, "namespace", e.target.value)}
            className={selectClass}
          >
            <option value="">{t.common.select}</option>
            {repoNs.map((ns) => (
              <option key={ns} value={ns}>
                {ns}
              </option>
            ))}
          </select>
        ) : (
          <div className="px-2 py-1.5 text-sm text-default">{mapping.namespace}</div>
        )}
      </div>

      {/* Deployment */}
      <div className="flex-1 min-w-0">
        <label className="block text-xs text-muted mb-1">
          {gt.deploymentCol}
        </label>
        {isEditable ? (
          <select
            value={mapping.deployment}
            onChange={(e) => onUpdate(mapping.id, "deployment", e.target.value)}
            className={selectClass}
          >
            <option value="">{t.common.select}</option>
            {filteredDeployments.map((d) => (
              <option key={d.name} value={d.name}>
                {d.name}
              </option>
            ))}
          </select>
        ) : (
          <div className="px-2 py-1.5 text-sm font-medium text-default">
            {mapping.deployment}
          </div>
        )}
      </div>

      {/* 当前镜像 tag（只读） */}
      <div className="flex-1 min-w-0">
        <label className="block text-xs text-muted mb-1">
          {gt.imageTagCol}
        </label>
        <div className="px-2 py-1.5">
          {currentImage ? (
            <code className="text-xs bg-[var(--bg-secondary)] px-1.5 py-0.5 rounded text-muted">
              {currentImage}
            </code>
          ) : (
            <span className="text-xs text-muted">—</span>
          )}
        </div>
      </div>

      {/* 源码目录 */}
      <div className="flex-1 min-w-0">
        <label className="block text-xs text-muted mb-1">
          {gt.sourcePathCol}
        </label>
        {isEditable ? (
          <select
            value={mapping.sourcePath}
            onChange={(e) => onUpdate(mapping.id, "sourcePath", e.target.value)}
            className={selectClass}
          >
            <option value="">{t.common.select}</option>
            {dirs.map((dir) => (
              <option key={dir} value={dir}>
                {dir}
              </option>
            ))}
          </select>
        ) : (
          <code className="block px-2 py-1.5 text-sm text-muted">
            {mapping.sourcePath}
          </code>
        )}
      </div>

      {/* 操作按钮 */}
      <div className="flex items-end pt-5 gap-2 flex-shrink-0">
        {mapping.confirmed && !editing ? (
          <>
            <Check className="w-5 h-5 text-emerald-500" />
            <button
              onClick={() => setEditing(true)}
              className="p-1.5 rounded-lg text-muted hover:text-default hover:bg-[var(--bg-secondary)] transition-colors"
              title={t.common.edit}
            >
              <Pencil className="w-4 h-4" />
            </button>
          </>
        ) : (
          <>
            <button
              onClick={handleConfirm}
              disabled={!mapping.namespace || !mapping.deployment || !mapping.sourcePath}
              className="px-3 py-1.5 text-xs rounded-lg bg-violet-600 text-white hover:bg-violet-700 transition-colors whitespace-nowrap disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {gt.confirmMapping}
            </button>
            <button
              onClick={() => onDelete(mapping.id)}
              className="p-1.5 rounded-lg text-muted hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"
              title={gt.deleteMapping}
            >
              <Trash2 className="w-4 h-4" />
            </button>
          </>
        )}
      </div>
    </div>
  );
}

// ---- NS 标签管理区（仓库头部） ----
function NamespaceTags({
  repo,
  repoNs,
  allNamespaces,
  onAdd,
  onRemove,
}: {
  repo: string;
  repoNs: string[];
  allNamespaces: string[];
  onAdd: (repo: string, ns: string) => void;
  onRemove: (repo: string, ns: string) => void;
}) {
  const { t } = useI18n();
  const gt = t.githubPage;
  const [adding, setAdding] = useState(false);

  // 可添加的 NS = 全部 NS - 已添加的
  const availableNs = allNamespaces.filter((ns) => !repoNs.includes(ns));

  return (
    <div className="flex items-center gap-2 flex-wrap">
      {/* 已配置的 NS 标签 */}
      {repoNs.map((ns) => (
        <span
          key={ns}
          className="inline-flex items-center gap-1 px-2 py-0.5 text-xs rounded-full bg-violet-100 dark:bg-violet-900/30 text-violet-700 dark:text-violet-300"
        >
          {ns}
          <button
            onClick={() => onRemove(repo, ns)}
            className="hover:text-red-500 transition-colors"
          >
            <X className="w-3 h-3" />
          </button>
        </span>
      ))}

      {/* 添加 NS */}
      {adding ? (
        <select
          autoFocus
          className="px-2 py-0.5 text-xs rounded-lg border border-violet-400 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100"
          defaultValue=""
          onChange={(e) => {
            if (e.target.value) {
              onAdd(repo, e.target.value);
            }
            setAdding(false);
          }}
          onBlur={() => setAdding(false)}
        >
          <option value="">{t.common.select}</option>
          {availableNs.map((ns) => (
            <option key={ns} value={ns}>
              {ns}
            </option>
          ))}
        </select>
      ) : (
        availableNs.length > 0 && (
          <button
            onClick={() => setAdding(true)}
            className="inline-flex items-center gap-1 px-2 py-0.5 text-xs rounded-full border border-dashed border-[var(--border-color)] text-muted hover:text-violet-600 hover:border-violet-400 transition-colors"
          >
            <Plus className="w-3 h-3" />
            {gt.addNamespace}
          </button>
        )
      )}
    </div>
  );
}

// ---- 主组件 ----
export function MappingCard({
  mappings,
  connected,
  namespaces,
  deployments,
  repoDirs,
  repoNamespaces,
  onUpdateMapping,
  onConfirmMapping,
  onConfirmAll,
  onAddMapping,
  onDeleteMapping,
  onAddRepoNamespace,
  onRemoveRepoNamespace,
}: MappingCardProps) {
  const { t } = useI18n();
  const gt = t.githubPage;

  const confirmedCount = mappings.filter((m) => m.confirmed).length;
  const totalCount = mappings.length;

  // 按仓库分组
  const repoGroups = mappings.reduce<Record<string, MockRepoMapping[]>>(
    (acc, m) => {
      if (!acc[m.repo]) acc[m.repo] = [];
      acc[m.repo].push(m);
      return acc;
    },
    {}
  );

  const enabledRepos = Object.keys(repoDirs);

  if (!connected) {
    return (
      <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
        <div className="flex items-center gap-2 px-6 py-4 border-b border-[var(--border-color)]">
          <GitBranch className="w-5 h-5 text-muted" />
          <h3 className="text-lg font-medium text-default">
            {gt.mappingSection}
          </h3>
        </div>
        <div className="p-12 text-center">
          <GitBranch className="w-12 h-12 mx-auto mb-3 text-muted opacity-30" />
          <p className="text-muted">{gt.noRepos}</p>
          <p className="text-xs text-muted mt-1">{gt.noReposHint}</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* 标题栏 */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <GitBranch className="w-5 h-5 text-muted" />
          <h3 className="text-lg font-medium text-default">
            {gt.mappingSection}
          </h3>
          {totalCount > 0 && (
            <span className="text-sm text-muted">
              {confirmedCount}/{totalCount} {gt.confirmedCol}
            </span>
          )}
        </div>
        {totalCount > 0 && confirmedCount < totalCount && (
          <button
            onClick={onConfirmAll}
            className="flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-lg bg-violet-600 text-white hover:bg-violet-700 transition-colors"
          >
            <CheckCheck className="w-4 h-4" />
            {gt.confirmMapping}
          </button>
        )}
      </div>

      <p className="text-sm text-muted">{gt.mappingSectionHint}</p>

      {/* 每个仓库一个卡片 */}
      {enabledRepos.map((repo) => {
        const items = repoGroups[repo] || [];
        const dirs = repoDirs[repo] || [];
        const repoNs = repoNamespaces[repo] || [];
        const repoConfirmed = items.filter((m) => m.confirmed).length;
        const hasNs = repoNs.length > 0;

        return (
          <div
            key={repo}
            className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden"
          >
            {/* 仓库标题 + NS 标签 */}
            <div className="px-6 py-3 border-b border-[var(--border-color)] bg-[var(--bg-secondary)]">
              <div className="flex items-center justify-between mb-2">
                <div className="flex items-center gap-2">
                  <GitBranch className="w-4 h-4 text-violet-500" />
                  <span className="text-sm font-semibold text-default">
                    {repo}
                  </span>
                </div>
                {items.length > 0 && (
                  <span className="text-xs text-muted">
                    {repoConfirmed}/{items.length} {gt.confirmedCol}
                  </span>
                )}
              </div>
              {/* NS 标签管理 */}
              <NamespaceTags
                repo={repo}
                repoNs={repoNs}
                allNamespaces={namespaces}
                onAdd={onAddRepoNamespace}
                onRemove={onRemoveRepoNamespace}
              />
            </div>

            {/* 映射行 */}
            <div className="px-6">
              {!hasNs ? (
                <div className="py-6 text-center text-sm text-muted">
                  {gt.noNamespaceHint}
                </div>
              ) : items.length === 0 ? (
                <div className="py-6 text-center text-sm text-muted">
                  {gt.addMapping}
                </div>
              ) : (
                items.map((mapping) => (
                  <MappingRow
                    key={mapping.id}
                    mapping={mapping}
                    repoNs={repoNs}
                    deployments={deployments}
                    dirs={dirs}
                    onUpdate={onUpdateMapping}
                    onConfirm={onConfirmMapping}
                    onDelete={onDeleteMapping}
                  />
                ))
              )}

              {/* 添加映射按钮（仅在有 NS 时显示） */}
              {hasNs && (
                <div className="py-3">
                  <button
                    onClick={() => onAddMapping(repo)}
                    className="flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-lg border border-dashed border-[var(--border-color)] text-muted hover:text-default hover:border-violet-400 hover:bg-[var(--bg-secondary)] transition-colors w-full justify-center"
                  >
                    <Plus className="w-4 h-4" />
                    {gt.addMapping}
                  </button>
                </div>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
}
