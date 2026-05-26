import Link from "next/link";

export function Footer() {
  return (
    <footer className="border-t border-[var(--border)] py-10">
      <div className="mx-auto max-w-6xl px-6 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 text-sm text-[var(--muted-foreground)]">
        <div className="flex items-baseline gap-2">
          <span className="flex items-baseline gap-0.5">
            <span
              className="italic text-lg leading-none tracking-[-0.02em] text-foreground"
              style={{ fontFamily: "var(--font-display), Georgia, serif" }}
            >
              reps
            </span>
            <span
              className="text-lg leading-none text-[var(--primary)]"
              style={{ fontFamily: "var(--font-display), Georgia, serif" }}
              aria-hidden
            >
              .
            </span>
          </span>
          <span className="text-xs font-mono uppercase tracking-[0.15em]">MIT · local-only</span>
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
