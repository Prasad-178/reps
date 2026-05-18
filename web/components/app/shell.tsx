"use client";

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

export function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();

  return (
    <div className="min-h-screen flex">
      <aside className="hidden md:flex md:flex-col w-60 shrink-0 border-r border-[var(--border)] bg-[var(--card)] sticky top-0 h-screen">
        <Link href="/" className="flex items-center gap-2 px-5 h-14 border-b border-[var(--border)]">
          <span className="grid place-items-center size-7 rounded-md bg-[var(--primary)] text-[var(--primary-foreground)] font-mono text-sm font-bold">
            R
          </span>
          <span className="font-semibold tracking-[-0.01em]">reps</span>
        </Link>

        <nav className="flex flex-col gap-0.5 px-3 py-3 flex-1 overflow-y-auto">
          {nav.map((n) => {
            const active = pathname === n.href || pathname.startsWith(n.href + "/");
            return (
              <Link
                key={n.href}
                href={n.href}
                className={cn(
                  "flex items-center gap-2.5 px-3 py-2 rounded-md text-sm",
                  "transition-colors duration-150 [transition-timing-function:var(--ease-out)]",
                  active
                    ? "bg-[color-mix(in_oklch,var(--primary)_14%,transparent)] text-foreground"
                    : "text-[var(--muted-foreground)] hover:bg-[var(--secondary)] hover:text-foreground"
                )}
              >
                <n.icon className="size-4 shrink-0" />
                <span>{n.label}</span>
              </Link>
            );
          })}
        </nav>

        <div className="px-3 py-3 border-t border-[var(--border)]">
          <Link
            href="/profile"
            className={cn(
              "flex items-center gap-2.5 px-3 py-2 rounded-md text-sm",
              "transition-colors duration-150 [transition-timing-function:var(--ease-out)]",
              pathname.startsWith("/profile")
                ? "bg-[var(--secondary)] text-foreground"
                : "text-[var(--muted-foreground)] hover:bg-[var(--secondary)] hover:text-foreground"
            )}
          >
            <User className="size-4 shrink-0" />
            <span>Profile</span>
          </Link>
        </div>
      </aside>

      <main className="flex-1 min-w-0">
        {children}
      </main>
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
    <div className="border-b border-[var(--border)] sticky top-0 z-30 bg-background/80 backdrop-blur-xl">
      <div className="px-6 sm:px-8 h-16 flex items-center justify-between">
        <div className="min-w-0">
          <h1 className="text-lg font-semibold tracking-[-0.015em] truncate">{title}</h1>
          {subtitle && (
            <p className="text-xs text-[var(--muted-foreground)] truncate">{subtitle}</p>
          )}
        </div>
        {action}
      </div>
    </div>
  );
}
