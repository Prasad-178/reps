import Link from "next/link";

export function Footer() {
  return (
    <footer className="border-t border-[var(--border)] py-10">
      <div className="mx-auto max-w-6xl px-6 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 text-sm text-[var(--muted-foreground)]">
        <div className="flex items-center gap-2">
          <span className="grid place-items-center size-6 rounded bg-[var(--primary)] text-[var(--primary-foreground)] font-mono text-[11px] font-bold">
            R
          </span>
          <span>reps · MIT · local-only</span>
        </div>
        <nav className="flex items-center gap-5 text-xs font-mono uppercase tracking-[0.08em]">
          <a href="https://github.com/Prasad-178/reps" className="hover:text-foreground transition-colors">
            GitHub
          </a>
          <Link href="/dashboard" className="hover:text-foreground transition-colors">
            App
          </Link>
          <a href="#install" className="hover:text-foreground transition-colors">
            Install
          </a>
        </nav>
      </div>
    </footer>
  );
}
