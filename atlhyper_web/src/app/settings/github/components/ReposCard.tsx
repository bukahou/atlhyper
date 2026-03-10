"use client";

import { useI18n } from "@/i18n/context";
import { BookMarked, Lock, Globe, GitBranch } from "lucide-react";
import type { MockAuthorizedRepo } from "@/mock/github/data";

interface ReposCardProps {
  repos: MockAuthorizedRepo[];
  connected: boolean;
  onToggleMapping: (fullName: string) => void;
}

export function ReposCard({ repos, connected, onToggleMapping }: ReposCardProps) {
  const { t } = useI18n();
  const gt = t.githubPage;

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
      <div className="flex items-center justify-between px-6 py-4 border-b border-[var(--border-color)]">
        <div className="flex items-center gap-2">
          <BookMarked className="w-5 h-5 text-muted" />
          <h3 className="text-lg font-medium text-default">{gt.reposSection}</h3>
        </div>
        {repos.length > 0 && (
          <span className="text-sm text-muted">
            {repos.length} {gt.reposCount}
          </span>
        )}
      </div>

      {!connected || repos.length === 0 ? (
        <div className="p-12 text-center">
          <BookMarked className="w-12 h-12 mx-auto mb-3 text-muted opacity-30" />
          <p className="text-muted">{gt.noRepos}</p>
          <p className="text-xs text-muted mt-1">{gt.noReposHint}</p>
        </div>
      ) : (
        <div className="divide-y divide-[var(--border-color)]">
          {repos.map((repo) => (
            <div
              key={repo.fullName}
              className="flex items-center justify-between px-6 py-3 hover:bg-[var(--bg-secondary)] transition-colors"
            >
              <div className="flex items-center gap-3">
                {repo.private ? (
                  <Lock className="w-4 h-4 text-amber-500" />
                ) : (
                  <Globe className="w-4 h-4 text-emerald-500" />
                )}
                <div>
                  <span className="text-sm font-medium text-default">
                    {repo.fullName}
                  </span>
                  <span className="text-xs text-muted ml-2">
                    {repo.defaultBranch}
                  </span>
                </div>
              </div>

              {/* 启用映射开关 */}
              <div className="flex items-center gap-2">
                <span className="text-xs text-muted flex items-center gap-1">
                  <GitBranch className="w-3 h-3" />
                  {gt.enableMapping}
                </span>
                <button
                  onClick={() => onToggleMapping(repo.fullName)}
                  className={`relative w-9 h-5 rounded-full transition-colors ${
                    repo.mappingEnabled
                      ? "bg-violet-600"
                      : "bg-gray-300 dark:bg-gray-600"
                  }`}
                >
                  <span
                    className={`absolute top-0.5 left-0.5 w-4 h-4 rounded-full bg-white transition-transform ${
                      repo.mappingEnabled ? "translate-x-4" : "translate-x-0"
                    }`}
                  />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
