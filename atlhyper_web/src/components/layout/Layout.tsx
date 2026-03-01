"use client";

import { useState, useEffect } from "react";
import { Menu } from "lucide-react";
import { Sidebar } from "@/components/navigation/Sidebar";
import { MobileMenu } from "@/components/navigation/MobileMenu";
import { LoginDialog } from "@/components/auth/LoginDialog";
import { ToastContainer } from "@/components/common";
import { useAuthError } from "@/hooks/useAuthError";
import { useClusterStore } from "@/store/clusterStore";
import { getClusterList } from "@/api/cluster";

interface LayoutProps {
  children: React.ReactNode;
}

export function Layout({ children }: LayoutProps) {
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const { setClusterIds } = useClusterStore();

  // 全局监听权限错误，自动触发登录对话框
  useAuthError();

  // 初始化集群列表
  useEffect(() => {
    const fetchClusters = async () => {
      try {
        const res = await getClusterList();
        const clusters = res.data?.clusters || [];
        if (clusters.length > 0) {
          const ids = clusters.map((c: { cluster_id: string }) => c.cluster_id);
          setClusterIds(ids);
        }
      } catch (err) {
        console.warn("[Layout] Failed to fetch clusters:", err);
      }
    };
    fetchClusters();
  }, [setClusterIds]);

  return (
    <div className="h-screen flex bg-[var(--background)] overflow-hidden">
      {/* Desktop Sidebar */}
      <div className="hidden lg:flex h-screen sticky top-0 overflow-visible z-50">
        <Sidebar collapsed={sidebarCollapsed} onToggle={() => setSidebarCollapsed((v) => !v)} />
      </div>

      {/* Mobile Menu */}
      <MobileMenu open={mobileMenuOpen} onClose={() => setMobileMenuOpen(false)} />

      {/* Main Content */}
      <div className="flex-1 min-w-0 flex flex-col h-screen relative z-0 py-3 pr-3 sm:py-4 sm:pr-4 md:py-6 md:pr-6">
        {/* Mobile Header - only shows on mobile */}
        <div className="lg:hidden h-14 flex items-center px-4 border-b border-[var(--border-color)]/30 flex-shrink-0 -mt-3 -mr-3 mb-3 sm:-mt-4 sm:-mr-4 sm:mb-4 md:-mt-6 md:-mr-6 md:mb-6 bg-card rounded-t-2xl">
          <button
            onClick={() => setMobileMenuOpen(true)}
            className="p-2 rounded-lg hover:bg-[var(--hover-bg)]"
            aria-label="Toggle menu"
          >
            <Menu className="w-5 h-5 text-secondary" />
          </button>
        </div>
        {/* 主内容卡片 - 圆角风格与 Sidebar 一致 */}
        <main className="flex-1 p-4 sm:p-5 md:p-6 overflow-y-auto overflow-x-hidden min-h-0 rounded-2xl bg-card border border-[var(--border-color)]/50 shadow-[0_10px_40px_rgb(0,0,0,0.08),0_0_20px_rgb(0,0,0,0.05)] dark:shadow-[0_10px_40px_rgb(0,0,0,0.3),0_0_20px_rgb(0,0,0,0.2)]">
          {children}
        </main>
      </div>

      {/* Login Dialog */}
      <LoginDialog />

      {/* Toast Notifications */}
      <ToastContainer />
    </div>
  );
}
