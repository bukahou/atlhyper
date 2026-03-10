"use client";

import { useI18n } from "@/i18n/context";
import { Settings, Plug, Pencil, Plus, Trash2, FolderGit2 } from "lucide-react";
import { useState } from "react";
import type { MockDeployConfig } from "@/mock/deploy/data";
import { Pagination, paginate } from "./Pagination";

interface ConfigCardProps {
  config: MockDeployConfig | null;
  editing: boolean;
  editConfig: MockDeployConfig | null;
  repos: { fullName: string; defaultBranch: string; private: boolean }[];
  kustomizePaths: string[];
  saving: boolean;
  onStartEdit: () => void;
  onCancelEdit: () => void;
  onSave: () => void;
  onUpdateConfig: (config: MockDeployConfig) => void;
  onTestConnection: () => Promise<boolean>;
}

const selectClass =
  "w-full px-3 py-2 rounded-lg border border-[var(--border-color)] bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 text-sm";

export function ConfigCard({
  config,
  editing,
  editConfig,
  repos,
  kustomizePaths,
  saving,
  onStartEdit,
  onCancelEdit,
  onSave,
  onUpdateConfig,
  onTestConnection,
}: ConfigCardProps) {
  const { t } = useI18n();
  const dt = t.deployPage;
  const [testing, setTesting] = useState(false);
  const [testResult, setTestResult] = useState<boolean | null>(null);
  const [pathPage, setPathPage] = useState(0);
  const PATH_PAGE_SIZE = 5;

  const handleTest = async () => {
    setTesting(true);
    setTestResult(null);
    const ok = await onTestConnection();
    setTestResult(ok);
    setTesting(false);
  };

  // --- 只读模式 ---
  if (!editing && config) {
    return (
      <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
        <div className="flex items-center justify-between px-6 py-4 border-b border-[var(--border-color)]">
          <div className="flex items-center gap-2">
            <Settings className="w-5 h-5 text-muted" />
            <h3 className="text-lg font-medium text-default">{dt.configSection}</h3>
          </div>
          <button
            onClick={onStartEdit}
            className="flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-lg border border-[var(--border-color)] text-muted hover:text-default hover:bg-[var(--bg-secondary)] transition-colors"
          >
            <Pencil className="w-4 h-4" />
            {dt.editConfig}
          </button>
        </div>
        <div className="p-6 space-y-4">
          {/* Config 仓库 */}
          <div className="flex items-center gap-3">
            <span className="text-sm text-muted w-28 flex-shrink-0">{dt.configRepo}</span>
            <span className="text-sm font-medium text-default">{config.repoUrl}</span>
          </div>

          {/* 部署路径列表 */}
          <div>
            <div className="flex items-center gap-2 mb-2">
              <FolderGit2 className="w-4 h-4 text-muted" />
              <span className="text-sm text-muted">{dt.pathsSection}</span>
              <span className="text-xs text-muted">({config.paths.length})</span>
            </div>
            {config.paths.length === 0 ? (
              <p className="text-sm text-muted pl-6">{dt.noPathsHint}</p>
            ) : (
              <>
                <div className="space-y-1.5 pl-6">
                  {paginate(config.paths, pathPage, PATH_PAGE_SIZE).map((p) => (
                    <code
                      key={p}
                      className="block text-xs bg-[var(--bg-secondary)] px-2 py-1 rounded text-default w-fit"
                    >
                      {p}
                    </code>
                  ))}
                </div>
                {config.paths.length > PATH_PAGE_SIZE && (
                  <div className="pl-6 mt-2">
                    <Pagination
                      page={pathPage}
                      pageSize={PATH_PAGE_SIZE}
                      total={config.paths.length}
                      onPageChange={setPathPage}
                      labels={t.table}
                    />
                  </div>
                )}
              </>
            )}
          </div>

          {/* 调谐设置 */}
          <div className="flex items-center gap-6">
            <div className="flex items-center gap-2">
              <span className="text-sm text-muted">{dt.pollInterval}:</span>
              <span className="text-sm text-default">{config.intervalSec}{dt.pollIntervalUnit}</span>
            </div>
            <div className="flex items-center gap-2">
              <span className="text-sm text-muted">{dt.autoDeploy}:</span>
              <span className={`text-sm font-medium ${config.autoDeploy ? "text-emerald-600 dark:text-emerald-400" : "text-gray-500"}`}>
                {config.autoDeploy ? dt.autoDeployOn : dt.autoDeployOff}
              </span>
            </div>
          </div>
        </div>
      </div>
    );
  }

  // --- 编辑模式 ---
  if (!editConfig) return null;

  // 已选路径（用于过滤下拉可选项）
  const selectedPaths = new Set(editConfig.paths);
  const availablePaths = kustomizePaths.filter((p) => !selectedPaths.has(p));

  const handleAddPath = () => {
    if (availablePaths.length === 0) return;
    // 添加空路径占位，用户通过下拉选择
    onUpdateConfig({
      ...editConfig,
      paths: [...editConfig.paths, ""],
    });
  };

  const handleUpdatePath = (index: number, value: string) => {
    const newPaths = [...editConfig.paths];
    newPaths[index] = value;
    onUpdateConfig({ ...editConfig, paths: newPaths });
  };

  const handleDeletePath = (index: number) => {
    onUpdateConfig({
      ...editConfig,
      paths: editConfig.paths.filter((_, i) => i !== index),
    });
  };

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
      <div className="flex items-center gap-2 px-6 py-4 border-b border-[var(--border-color)]">
        <Settings className="w-5 h-5 text-muted" />
        <h3 className="text-lg font-medium text-default">{dt.configSection}</h3>
      </div>

      <div className="p-6 space-y-5">
        {/* Config 仓库选择 */}
        <div>
          <label className="block text-sm font-medium text-default mb-1">
            {dt.configRepo}
          </label>
          <select
            value={editConfig.repoUrl}
            onChange={(e) => onUpdateConfig({ ...editConfig, repoUrl: e.target.value, paths: [] })}
            className={selectClass}
          >
            <option value="">{dt.configRepoPlaceholder}</option>
            {repos.map((repo) => (
              <option key={repo.fullName} value={repo.fullName}>
                {repo.fullName}{repo.private ? " 🔒" : ""}
              </option>
            ))}
          </select>
        </div>

        {/* 部署路径列表（仅在选择了仓库后显示） */}
        {editConfig.repoUrl && (
          <div>
            <div className="mb-2">
              <label className="block text-sm font-medium text-default">
                {dt.pathsSection}
              </label>
              <p className="text-xs text-muted mt-0.5">{dt.pathsSectionHint}</p>
            </div>

            <div className="space-y-2">
              {editConfig.paths.map((path, index) => {
                // 当前行可选 = 未被其他行选中的 + 自己当前选中的
                const rowAvailable = kustomizePaths.filter(
                  (kp) => !selectedPaths.has(kp) || kp === path
                );

                return (
                  <div key={index} className="flex items-center gap-2">
                    <select
                      value={path}
                      onChange={(e) => handleUpdatePath(index, e.target.value)}
                      className={selectClass}
                    >
                      <option value="">{t.common.select}</option>
                      {rowAvailable.map((kp) => (
                        <option key={kp} value={kp}>{kp}</option>
                      ))}
                    </select>
                    <button
                      onClick={() => handleDeletePath(index)}
                      className="p-2 rounded-lg text-muted hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors flex-shrink-0"
                      title={dt.deletePath}
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>
                );
              })}

              {availablePaths.length > 0 && (
                <button
                  onClick={handleAddPath}
                  className="flex items-center gap-1.5 px-3 py-2 text-sm rounded-lg border border-dashed border-[var(--border-color)] text-muted hover:text-default hover:border-violet-400 hover:bg-[var(--bg-secondary)] transition-colors w-full justify-center"
                >
                  <Plus className="w-4 h-4" />
                  {dt.addPath}
                </button>
              )}

              {kustomizePaths.length === 0 && (
                <p className="text-sm text-muted text-center py-3">
                  {dt.noPathsHint}
                </p>
              )}
            </div>
          </div>
        )}

        {/* 调谐设置 */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-default mb-1">
              {dt.pollInterval}
            </label>
            <div className="flex items-center gap-2">
              <input
                type="number"
                min={10}
                max={600}
                value={editConfig.intervalSec}
                onChange={(e) =>
                  onUpdateConfig({ ...editConfig, intervalSec: parseInt(e.target.value) || 60 })
                }
                className="w-24 px-3 py-2 rounded-lg border border-[var(--border-color)] bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 text-sm"
              />
              <span className="text-sm text-muted">{dt.pollIntervalUnit}</span>
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-default mb-1">
              {dt.autoDeploy}
            </label>
            <button
              onClick={() => onUpdateConfig({ ...editConfig, autoDeploy: !editConfig.autoDeploy })}
              className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                editConfig.autoDeploy
                  ? "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400"
                  : "bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400"
              }`}
            >
              {editConfig.autoDeploy ? dt.autoDeployOn : dt.autoDeployOff}
            </button>
            <p className="text-xs text-muted mt-1">{dt.autoDeployHint}</p>
          </div>
        </div>

        {/* 操作按钮 */}
        <div className="flex items-center justify-end gap-3 pt-2">
          {testResult !== null && (
            <span className={`text-sm ${testResult ? "text-emerald-600" : "text-red-600"}`}>
              {testResult ? dt.testConnectionSuccess : dt.testConnectionFailed}
            </span>
          )}
          <button
            onClick={handleTest}
            disabled={testing || !editConfig.repoUrl}
            className="flex items-center gap-2 px-4 py-2 text-sm rounded-lg border border-[var(--border-color)] text-default hover:bg-[var(--bg-secondary)] transition-colors disabled:opacity-50"
          >
            <Plug className="w-4 h-4" />
            {testing ? "..." : dt.testConnection}
          </button>
          {config && (
            <button
              onClick={onCancelEdit}
              className="px-4 py-2 text-sm rounded-lg border border-[var(--border-color)] text-default hover:bg-[var(--bg-secondary)] transition-colors"
            >
              {t.common.cancel}
            </button>
          )}
          <button
            onClick={onSave}
            disabled={saving || !editConfig.repoUrl}
            className="px-4 py-2 text-sm rounded-lg bg-violet-600 text-white hover:bg-violet-700 transition-colors disabled:opacity-50"
          >
            {saving ? "..." : t.common.save}
          </button>
        </div>
      </div>
    </div>
  );
}
