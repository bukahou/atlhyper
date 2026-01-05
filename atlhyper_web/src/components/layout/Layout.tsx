"use client";

import { useState } from "react";
import { Sidebar } from "@/components/navigation/Sidebar";
import { Navbar } from "@/components/navigation/Navbar";
import { MobileMenu } from "@/components/navigation/MobileMenu";
import { LoginDialog } from "@/components/auth/LoginDialog";
import { ToastContainer } from "@/components/common";
import { useAuthError } from "@/hooks/useAuthError";

interface LayoutProps {
  children: React.ReactNode;
}

export function Layout({ children }: LayoutProps) {
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  // 全局监听权限错误，自动触发登录对话框
  useAuthError();

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
