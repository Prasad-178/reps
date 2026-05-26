"use client";

import Link from "next/link";
import { Button } from "@/components/ui/button";

function GithubIcon({ className }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="currentColor"
      className={className ?? "size-4"}
      aria-hidden
    >
      <path d="M12 .5C5.65.5.5 5.65.5 12c0 5.08 3.29 9.39 7.86 10.91.58.11.79-.25.79-.55v-2.05c-3.2.7-3.87-1.37-3.87-1.37-.52-1.33-1.28-1.68-1.28-1.68-1.05-.72.08-.7.08-.7 1.16.08 1.77 1.19 1.77 1.19 1.03 1.76 2.7 1.25 3.36.95.1-.75.4-1.25.73-1.54-2.55-.29-5.24-1.28-5.24-5.69 0-1.26.45-2.29 1.19-3.1-.12-.29-.52-1.47.11-3.06 0 0 .97-.31 3.18 1.18a11.06 11.06 0 0 1 5.79 0c2.21-1.49 3.18-1.18 3.18-1.18.63 1.59.23 2.77.11 3.06.74.81 1.19 1.84 1.19 3.1 0 4.42-2.69 5.4-5.25 5.68.41.35.78 1.04.78 2.1v3.11c0 .3.21.66.8.55C20.21 21.39 23.5 17.08 23.5 12 23.5 5.65 18.35.5 12 .5Z"/>
    </svg>
  );
}

export function MarketingNav() {
  return (
    <header className="sticky top-0 z-40 backdrop-blur-xl bg-background/70 border-b border-[var(--border)]">
      <div className="mx-auto max-w-6xl px-6 h-14 flex items-center justify-between">
        <Link href="/" className="flex items-baseline gap-1.5 group" aria-label="reps home">
          <span
            className="italic text-2xl leading-none tracking-[-0.02em] text-foreground transition-colors duration-200 [transition-timing-function:var(--ease-out)] group-hover:text-[var(--primary)]"
            style={{ fontFamily: "var(--font-display), Georgia, serif" }}
          >
            reps
          </span>
          <span
            className="text-2xl leading-none text-[var(--primary)] -translate-y-px transition-transform duration-200 [transition-timing-function:var(--ease-out)] group-hover:translate-x-0.5"
            style={{ fontFamily: "var(--font-display), Georgia, serif" }}
            aria-hidden
          >
            .
          </span>
          <span className="hidden sm:inline text-[10px] font-mono uppercase tracking-[0.18em] text-[var(--muted-foreground)] ml-2 self-center">
            v0.1
          </span>
        </Link>

        <nav className="hidden md:flex items-center gap-7 text-sm text-[var(--muted-foreground)]">
          <Link href="#how" className="hover:text-foreground transition-colors duration-150">
            How it works
          </Link>
          <Link href="#agents" className="hover:text-foreground transition-colors duration-150">
            Agents
          </Link>
          <Link href="#dashboard" className="hover:text-foreground transition-colors duration-150">
            Dashboard
          </Link>
          <Link href="#install" className="hover:text-foreground transition-colors duration-150">
            Install
          </Link>
        </nav>

        <div className="flex items-center gap-2">
          <Button variant="ghost" size="icon" asChild>
            <a href="https://github.com/Prasad-178/reps" target="_blank" rel="noreferrer" aria-label="GitHub">
              <GithubIcon />
            </a>
          </Button>
          <Button asChild size="sm">
            <Link href="/dashboard">Open app</Link>
          </Button>
        </div>
      </div>
    </header>
  );
}
