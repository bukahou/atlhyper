"use client";

import { Menu, Bell } from "lucide-react";
import { LanguageSwitcher } from "./LanguageSwitcher";
import { ThemeSwitcher } from "./ThemeSwitcher";
import { UserMenu } from "./UserMenu";
import { ClusterSelector } from "./ClusterSelector";

interface NavbarProps {
  onMenuClick?: () => void;
}

export function Navbar({ onMenuClick }: NavbarProps) {
  return (
    <header className="h-14 bg-[var(--background)] border-b border-[var(--border-color)]/50 px-4 flex items-center justify-between">
      {/* Left side */}
      <div className="flex items-center gap-4">
        {/* Mobile menu toggle */}
        <button
          onClick={onMenuClick}
          className="p-2 rounded-lg hover-bg lg:hidden"
          aria-label="Toggle menu"
        >
          <Menu className="w-5 h-5 text-secondary" />
        </button>

        {/* Cluster Selector */}
        <ClusterSelector />
      </div>

      {/* Right side */}
      <div className="flex items-center gap-1">
        <button
          className="p-2 rounded-lg hover-bg relative"
          aria-label="Notifications"
        >
          <Bell className="w-[18px] h-[18px] text-muted" />
        </button>
        <LanguageSwitcher />
        <ThemeSwitcher />
        <UserMenu />
      </div>
    </header>
  );
}
