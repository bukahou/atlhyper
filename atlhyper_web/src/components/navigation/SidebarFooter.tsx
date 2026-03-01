"use client";

import {
  User,
  LogIn,
  LogOut,
  Settings,
  Languages,
  Sun,
  ChevronsLeft,
  ChevronsRight,
} from "lucide-react";
import { useState } from "react";
import { useI18n } from "@/i18n/context";
import { useAuthStore } from "@/store/authStore";
import { useTheme } from "@/theme/context";
import { languages, themeOptions, getRoleName } from "./SidebarTypes";

interface SidebarFooterProps {
  collapsed: boolean;
  onToggle: () => void;
}

export function SidebarFooter({ collapsed, onToggle }: SidebarFooterProps) {
  const { t, language, setLanguage } = useI18n();
  const { isAuthenticated, user, openLoginDialog, logout } = useAuthStore();
  const { theme, setTheme } = useTheme();
  const [userMenuOpen, setUserMenuOpen] = useState(false);
  const [settingsMenuOpen, setSettingsMenuOpen] = useState(false);

  return (
    <div className={`border-t border-[var(--border-color)]/20 ${collapsed ? "py-2" : "p-3"}`}>
      {/* 用户区域 */}
      <div
        className="relative"
        onMouseEnter={() => setUserMenuOpen(true)}
        onMouseLeave={() => setUserMenuOpen(false)}
      >
        {isAuthenticated ? (
          <>
            <button
              className={`flex items-center gap-3 rounded-xl transition-all duration-150 hover:bg-white/5 ${
                collapsed ? "p-2 mx-auto" : "w-full px-3 py-2.5"
              }`}
            >
              <div className="w-8 h-8 rounded-full bg-primary/20 flex items-center justify-center flex-shrink-0">
                <User className="w-4 h-4 text-primary" />
              </div>
              {!collapsed && (
                <div className="flex-1 text-left min-w-0">
                  <p className="text-sm font-medium text-default truncate">
                    {user?.displayName || user?.username || "User"}
                  </p>
                  <p className="text-xs text-muted">
                    {user ? getRoleName(user.role) : ""}
                  </p>
                </div>
              )}
            </button>

            {/* 用户悬浮卡片 */}
            {userMenuOpen && (
              <div className={`absolute z-50 ${collapsed ? "left-full bottom-0 pl-2" : "left-0 right-0 bottom-full pb-2"}`}>
                <div className="py-2 rounded-2xl border border-[var(--border-color)] bg-card shadow-[0_8px_30px_rgb(0,0,0,0.15)] dark:shadow-[0_8px_30px_rgb(0,0,0,0.5)] min-w-[180px]">
                  {/* 用户信息 */}
                  <div className="px-4 py-3 border-b border-[var(--border-color)]/30">
                    <p className="text-sm font-medium text-default">
                      {user?.displayName || user?.username}
                    </p>
                    {user?.email && (
                      <p className="text-xs text-secondary truncate">
                        {user.email}
                      </p>
                    )}
                    <p className="text-xs text-muted">
                      {user ? getRoleName(user.role) : ""}
                    </p>
                  </div>

                  {/* 退出登录 */}
                  <div className="px-2 py-2">
                    <button
                      onClick={logout}
                      className="w-full flex items-center gap-2 px-3 py-2 rounded-xl text-sm text-red-500 hover:bg-red-500/10 transition-all"
                    >
                      <LogOut className="w-4 h-4" />
                      {t.common.logout}
                    </button>
                  </div>
                </div>
              </div>
            )}
          </>
        ) : (
          <button
            onClick={() => openLoginDialog()}
            className={`flex items-center gap-2 rounded-xl text-primary hover:bg-primary/10 transition-all ${
              collapsed ? "p-2 mx-auto" : "w-full px-3 py-2.5"
            }`}
          >
            <LogIn className="w-5 h-5" />
            {!collapsed && <span className="text-sm font-medium">{t.common.login}</span>}
          </button>
        )}
      </div>

      {/* 设置 + 折叠 并列 */}
      <div className={`flex items-center gap-1 mt-2 ${collapsed ? "flex-col" : ""}`}>
        {/* 设置按钮（语言+主题） */}
        <div
          className="relative flex-1"
          onMouseEnter={() => setSettingsMenuOpen(true)}
          onMouseLeave={() => setSettingsMenuOpen(false)}
        >
          <button
            className={`flex items-center gap-2 rounded-xl hover:bg-white/5 transition-all ${
              collapsed ? "p-2 mx-auto" : "w-full px-3 py-2"
            }`}
            title="Settings"
          >
            <Settings className="w-4 h-4 text-muted" />
            {!collapsed && <span className="text-xs text-muted font-medium">{t.common.settings}</span>}
          </button>

          {/* 设置弹出面板 */}
          {settingsMenuOpen && (
            <div className={`absolute z-50 ${collapsed ? "left-full bottom-0 pl-2" : "left-0 bottom-full pb-2"}`}>
              <div className="py-2 rounded-2xl border border-[var(--border-color)] bg-card shadow-[0_8px_30px_rgb(0,0,0,0.15)] dark:shadow-[0_8px_30px_rgb(0,0,0,0.5)] min-w-[180px]">
                {/* 语言切换 */}
                <div className="px-2 py-2 border-b border-[var(--border-color)]/30">
                  <div className="px-2 py-1 text-[11px] font-semibold text-muted uppercase tracking-wider flex items-center gap-2">
                    <Languages className="w-3.5 h-3.5" />
                    Language
                  </div>
                  <div className="flex gap-1 mt-1">
                    {languages.map((lang) => (
                      <button
                        key={lang.code}
                        onClick={() => setLanguage(lang.code)}
                        className={`flex-1 px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${
                          language === lang.code
                            ? "bg-primary/15 text-primary"
                            : "text-secondary hover:bg-white/5"
                        }`}
                      >
                        {lang.label}
                      </button>
                    ))}
                  </div>
                </div>

                {/* 主题切换 */}
                <div className="px-2 py-2">
                  <div className="px-2 py-1 text-[11px] font-semibold text-muted uppercase tracking-wider flex items-center gap-2">
                    <Sun className="w-3.5 h-3.5" />
                    Theme
                  </div>
                  <div className="flex gap-1 mt-1">
                    {themeOptions.map((th) => {
                      const ThemeIcon = th.icon;
                      const themeLabel = th.value === "light" ? t.common.themeLight : th.value === "dark" ? t.common.themeDark : t.common.themeSystem;
                      return (
                        <button
                          key={th.value}
                          onClick={() => setTheme(th.value)}
                          className={`flex-1 px-2 py-1.5 rounded-lg text-xs font-medium transition-all flex items-center justify-center gap-1 ${
                            theme === th.value
                              ? "bg-primary/15 text-primary"
                              : "text-secondary hover:bg-white/5"
                          }`}
                        >
                          <ThemeIcon className="w-3.5 h-3.5" />
                          {themeLabel}
                        </button>
                      );
                    })}
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* 折叠按钮 */}
        <button
          onClick={onToggle}
          className={`flex items-center gap-2 rounded-xl hover:bg-white/5 transition-all duration-150 ${
            collapsed ? "p-2" : "flex-1 px-3 py-2"
          }`}
          title={collapsed ? t.common.expandSidebar : t.common.collapseSidebar}
        >
          {collapsed ? (
            <ChevronsRight className="w-4 h-4 text-muted" />
          ) : (
            <>
              <ChevronsLeft className="w-4 h-4 text-muted" />
              <span className="text-xs text-muted font-medium">{t.common.collapseSidebar}</span>
            </>
          )}
        </button>
      </div>
    </div>
  );
}
