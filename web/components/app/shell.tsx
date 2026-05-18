"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  LayoutDashboard,
  Brain,
  FileText,
  Briefcase,
  Map,
  History,
  User,
  Settings as SettingsIcon,
  Menu,
  X,
} from "lucide-react";
import { cn } from "@/lib/utils";

const nav = [
  { href: "/dashboard", label: "Dashboard", icon: LayoutDashboard },
  { href: "/drill",     label: "Drill",     icon: Brain },
  { href: "/sources",   label: "Sources",   icon: FileText },
  { href: "/jds",       label: "JDs",       icon: Briefcase },
  { href: "/plan",      label: "Plan",      icon: Map },
  { href: "/history",   label: "History",   icon: History },
];

const secondaryNav = [
  { href: "/profile",  label: "Profile",  icon: User },
  { href: "/settings", label: "Settings", icon: SettingsIcon },
];

function NavLink({
  href,
  label,
  Icon,
  active,
  variant = "primary",
  onClick,
}: {
  href: string;
  label: string;
  Icon: React.ComponentType<{ className?: string }>;
  active: boolean;
  variant?: "primary" | "secondary";
  onClick?: () => void;
}) {
  return (
    <Link
      href={href}
      onClick={onClick}
      className={cn(
        "flex items-center gap-2.5 px-3 py-2 rounded-md text-sm",
        "transition-colors duration-150 [transition-timing-function:var(--ease-out)]",
        active
          ? variant === "primary"
            ? "bg-[color-mix(in_oklch,var(--primary)_14%,transparent)] text-foreground"
            : "bg-[var(--secondary)] text-foreground"
          : "text-[var(--muted-foreground)] hover:bg-[var(--secondary)] hover:text-foreground"
      )}
    >
      <Icon className="size-4 shrink-0" />
      <span>{label}</span>
    </Link>
  );
}

function SidebarBody({
  pathname,
  onItemClick,
}: {
  pathname: string;
  onItemClick?: () => void;
}) {
  return (
    <>
      <Link
        href="/"
        className="flex items-center gap-2 px-5 h-14 border-b border-[var(--border)]"
        onClick={onItemClick}
      >
        <span className="grid place-items-center size-7 rounded-md bg-[var(--primary)] text-[var(--primary-foreground)] font-mono text-sm font-bold">
          R
        </span>
        <span className="font-semibold tracking-[-0.01em]">reps</span>
      </Link>

      <nav className="flex flex-col gap-0.5 px-3 py-3 flex-1 overflow-y-auto">
        {nav.map((n) => (
          <NavLink
            key={n.href}
            href={n.href}
            label={n.label}
            Icon={n.icon}
            active={pathname === n.href || pathname.startsWith(n.href + "/")}
            onClick={onItemClick}
          />
        ))}
      </nav>

      <div className="px-3 py-3 border-t border-[var(--border)] space-y-0.5">
        {secondaryNav.map((n) => (
          <NavLink
            key={n.href}
            href={n.href}
            label={n.label}
            Icon={n.icon}
            active={pathname === n.href || pathname.startsWith(n.href + "/")}
            variant="secondary"
            onClick={onItemClick}
          />
        ))}
      </div>
    </>
  );
}

export function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const [drawerOpen, setDrawerOpen] = useState(false);

  // close drawer on route change
  useEffect(() => {
    setDrawerOpen(false);
  }, [pathname]);

  // lock body scroll while drawer is open
  useEffect(() => {
    if (drawerOpen) {
      document.body.style.overflow = "hidden";
      return () => {
        document.body.style.overflow = "";
      };
    }
  }, [drawerOpen]);

  return (
    <div className="min-h-screen flex">
      {/* Desktop sidebar */}
      <aside className="hidden md:flex md:flex-col w-60 shrink-0 border-r border-[var(--border)] bg-[var(--card)] sticky top-0 h-screen">
        <SidebarBody pathname={pathname} />
      </aside>

      {/* Mobile drawer */}
      {drawerOpen && (
        <>
          <div
            className="fixed inset-0 z-40 bg-black/60 backdrop-blur-sm md:hidden animate-in fade-in-0 duration-200"
            onClick={() => setDrawerOpen(false)}
            aria-hidden
          />
          <aside
            className={cn(
              "fixed inset-y-0 left-0 z-50 w-64 flex-col border-r border-[var(--border)] bg-[var(--card)] flex md:hidden",
              "animate-in slide-in-from-left-2 fade-in-0 duration-200"
            )}
          >
            <SidebarBody pathname={pathname} onItemClick={() => setDrawerOpen(false)} />
          </aside>
        </>
      )}

      <div className="flex-1 min-w-0 flex flex-col">
        {/* Mobile topbar (shown < md) */}
        <header className="md:hidden sticky top-0 z-30 flex items-center gap-3 h-14 px-4 border-b border-[var(--border)] bg-background/85 backdrop-blur-xl">
          <button
            onClick={() => setDrawerOpen((o) => !o)}
            className="grid place-items-center size-9 rounded-md hover:bg-[var(--secondary)] transition-colors duration-150 [transition-timing-function:var(--ease-out)] active:scale-[0.97]"
            aria-label="Open navigation"
          >
            {drawerOpen ? <X className="size-5" /> : <Menu className="size-5" />}
          </button>
          <Link href="/" className="flex items-center gap-2">
            <span className="grid place-items-center size-6 rounded bg-[var(--primary)] text-[var(--primary-foreground)] font-mono text-[11px] font-bold">
              R
            </span>
            <span className="font-semibold text-sm tracking-[-0.01em]">reps</span>
          </Link>
        </header>

        <main className="flex-1 min-w-0">{children}</main>
      </div>
    </div>
  );
}

export function PageHeader({
  title,
  subtitle,
  action,
}: {
  title: string;
  subtitle?: string;
  action?: React.ReactNode;
}) {
  return (
    <div className="border-b border-[var(--border)] sticky top-14 md:top-0 z-20 bg-background/80 backdrop-blur-xl">
      <div className="px-4 sm:px-6 md:px-8 h-14 sm:h-16 flex items-center justify-between gap-3">
        <div className="min-w-0">
          <h1 className="text-base sm:text-lg font-semibold tracking-[-0.015em] truncate">{title}</h1>
          {subtitle && (
            <p className="text-[11px] sm:text-xs text-[var(--muted-foreground)] truncate">
              {subtitle}
            </p>
          )}
        </div>
        {action && <div className="shrink-0 flex items-center gap-2">{action}</div>}
      </div>
    </div>
  );
}
