"use client";

import { useState, useMemo } from "react";
import { useI18n } from "@/i18n/context";
import { BookMarked, Lock, Globe, Search } from "lucide-react";
import { Pagination, paginate } from "@/app/admin/deploy/components/Pagination";
import type { MockAuthorizedRepo } from "@/mock/github/data";

const PAGE_SIZE = 8;

interface ReposCardProps {
  repos: MockAuthorizedRepo[];
  connected: boolean;
}

export function ReposCard({ repos, connected }: ReposCardProps) {
  const { t } = useI18n();
  const gt = t.githubPage;
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(0);

  const filtered = useMemo(() => {
    if (!search.trim()) return repos;
    const q = search.toLowerCase();
    return repos.filter((r) => r.fullName.toLowerCase().includes(q));
  }, [repos, search]);

  // 搜索变化时重置分页
  const handleSearch = (value: string) => {
    setSearch(value);
    setPage(0);
  };

  const paged = paginate(filtered, page, PAGE_SIZE);

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
      <div className="flex items-center justify-between px-6 py-4 border-b border-[var(--border-color)]">
        <div className="flex items-center gap-2">
          <BookMarked className="w-5 h-5 text-muted" />
          <h3 className="text-lg font-medium text-default">{gt.reposSection}</h3>
        </div>
        <span className="text-sm text-muted">
          {filtered.length}{search && filtered.length !== repos.length ? ` / ${repos.length}` : ""} {gt.reposCount}
        </span>
      </div>

      {/* 搜索栏 */}
      {connected && repos.length > 0 && (
        <div className="px-6 pt-4">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted" />
            <input
              type="text"
              value={search}
              onChange={(e) => handleSearch(e.target.value)}
              placeholder={t.common.search}
              className="w-full pl-9 pr-3 py-2 text-sm rounded-lg border border-[var(--border-color)] bg-white dark:bg-gray-800 text-default placeholder:text-muted"
            />
          </div>
        </div>
      )}

      {!connected || repos.length === 0 ? (
        <div className="p-12 text-center">
          <BookMarked className="w-12 h-12 mx-auto mb-3 text-muted opacity-30" />
          <p className="text-muted">{gt.noRepos}</p>
          <p className="text-xs text-muted mt-1">{gt.noReposHint}</p>
        </div>
      ) : filtered.length === 0 ? (
        <div className="p-8 text-center">
          <Search className="w-8 h-8 mx-auto mb-2 text-muted opacity-30" />
          <p className="text-sm text-muted">{t.table.noData}</p>
        </div>
      ) : (
        <>
          <div className="divide-y divide-[var(--border-color)]">
            {paged.map((repo) => (
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
              </div>
            ))}
          </div>
          <Pagination
            page={page}
            pageSize={PAGE_SIZE}
            total={filtered.length}
            onPageChange={setPage}
            labels={t.table}
          />
        </>
      )}
    </div>
  );
}
