"use client";

import { useState, useEffect } from "react";
import { Sidebar } from "@/components/navigation/Sidebar";
import { Navbar } from "@/components/navigation/Navbar";
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
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
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
    <div className="min-h-screen flex bg-[var(--background)]">
      {/* Desktop Sidebar */}
      <div className="hidden lg:block">
        <Sidebar collapsed={sidebarCollapsed} />
      </div>

      {/* Mobile Menu */}
      <MobileMenu open={mobileMenuOpen} onClose={() => setMobileMenuOpen(false)} />

      {/* Main Content */}
      <div className="flex-1 flex flex-col min-h-screen">
        <Navbar onMenuClick={() => setMobileMenuOpen(true)} />
        <main className="flex-1 p-6 overflow-y-auto overflow-x-hidden">{children}</main>
      </div>

      {/* Login Dialog */}
      <LoginDialog />

      {/* Toast Notifications */}
      <ToastContainer />
    </div>
  );
}
