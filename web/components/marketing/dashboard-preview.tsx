"use client";

import { motion } from "framer-motion";
import { Badge } from "@/components/ui/badge";

const easeOut: [number, number, number, number] = [0.23, 1, 0.32, 1];

const eloData = [
  1180, 1184, 1182, 1191, 1188, 1195, 1198, 1204, 1201, 1209, 1213, 1218,
  1215, 1221, 1219, 1225, 1228, 1232, 1230, 1238, 1241, 1245, 1248, 1252,
];

function Sparkline() {
  const W = 480;
  const H = 110;
  const min = Math.min(...eloData);
  const max = Math.max(...eloData);
  const pts = eloData.map((v, i) => {
    const x = (i / (eloData.length - 1)) * W;
    const y = H - ((v - min) / (max - min)) * (H - 14) - 7;
    return [x, y] as const;
  });
  const line = pts
    .map((p, i) => (i === 0 ? "M" : "L") + p[0].toFixed(1) + " " + p[1].toFixed(1))
    .join(" ");
  const area = line + ` L${W} ${H} L0 ${H} Z`;
  return (
    <svg viewBox={`0 0 ${W} ${H}`} preserveAspectRatio="none" className="w-full h-[110px]">
      <defs>
        <linearGradient id="elo-grad" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor="var(--primary)" stopOpacity="0.35" />
          <stop offset="100%" stopColor="var(--primary)" stopOpacity="0" />
        </linearGradient>
      </defs>
      <path d={area} fill="url(#elo-grad)" />
      <path
        d={line}
        fill="none"
        stroke="var(--primary)"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

export function DashboardPreview() {
  return (
    <section id="dashboard" className="relative py-24 sm:py-32 border-t border-[var(--border)] overflow-hidden">
      <div
        aria-hidden
        className="absolute inset-0 pointer-events-none"
        style={{
          background:
            "radial-gradient(50% 50% at 50% 0%, color-mix(in oklch, var(--primary) 8%, transparent) 0%, transparent 70%)",
        }}
      />
      <div className="relative mx-auto max-w-6xl px-6">
        <div className="max-w-2xl">
          <p className="font-mono text-xs uppercase tracking-[0.12em] text-[var(--primary)] mb-3">
            Dashboard
          </p>
          <h2 className="text-3xl sm:text-4xl font-semibold tracking-[-0.025em] leading-[1.05]">
            Numbers that compound. Topics that don&apos;t.
          </h2>
          <p className="mt-4 text-[var(--muted-foreground)] leading-relaxed">
            Per-category ELO. 7-day trend. Weakest topics surfaced before they cost you a
            real loop. Drop into a drill in one tap.
          </p>
        </div>

        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true, margin: "-100px" }}
          transition={{ duration: 0.6, ease: easeOut }}
          className="mt-12 rounded-2xl border border-[var(--border)] bg-[var(--card)] overflow-hidden shadow-[0_30px_60px_-30px_color-mix(in_oklch,var(--primary)_28%,transparent)]"
        >
          {/* fake titlebar */}
          <div className="flex items-center gap-2 px-4 h-9 border-b border-[var(--border)] bg-[color-mix(in_oklch,var(--card)_70%,var(--background))]">
            <span className="size-2.5 rounded-full bg-[color-mix(in_oklch,var(--foreground)_18%,transparent)]" />
            <span className="size-2.5 rounded-full bg-[color-mix(in_oklch,var(--foreground)_14%,transparent)]" />
            <span className="size-2.5 rounded-full bg-[color-mix(in_oklch,var(--foreground)_10%,transparent)]" />
            <span className="ml-3 font-mono text-[11px] text-[var(--muted-foreground)]">
              reps · dashboard
            </span>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-[200px_1fr] divide-y md:divide-y-0 md:divide-x divide-[var(--border)]">
            <aside className="p-5 flex flex-col gap-1 text-sm">
              <div className="flex items-baseline gap-0.5 mb-3">
                <span
                  className="italic text-xl leading-none tracking-[-0.02em] text-foreground"
                  style={{ fontFamily: "var(--font-display), Georgia, serif" }}
                >
                  reps
                </span>
                <span
                  className="text-xl leading-none text-[var(--primary)]"
                  style={{ fontFamily: "var(--font-display), Georgia, serif" }}
                  aria-hidden
                >
                  .
                </span>
              </div>
              {["Dashboard", "Drill", "Sources", "JDs", "Plan", "History"].map((n, i) => (
                <div
                  key={n}
                  className={
                    "px-2.5 py-1.5 rounded-md text-[13px] flex items-center gap-2 " +
                    (i === 0
                      ? "bg-[color-mix(in_oklch,var(--primary)_14%,transparent)] text-[var(--foreground)]"
                      : "text-[var(--muted-foreground)]")
                  }
                >
                  <span className="size-1.5 rounded-full bg-current opacity-60" />
                  {n}
                </div>
              ))}
            </aside>
            <div className="p-6 space-y-5">
              <div className="flex items-end justify-between">
                <div>
                  <h3 className="text-xl font-semibold tracking-[-0.015em]">Dashboard</h3>
                  <p className="text-xs text-[var(--muted-foreground)] mt-0.5">
                    21 drills · last 7 days
                  </p>
                </div>
                <button className="px-3 h-8 rounded-md bg-[var(--primary)] text-[var(--primary-foreground)] text-xs font-semibold transition-transform active:scale-[0.97] [transition-timing-function:var(--ease-out)]">
                  Start drill
                </button>
              </div>

              <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
                {[
                  ["Overall ELO", "1,252", "+72", "up"],
                  ["System design", "1,238", "+11", "up"],
                  ["Domain · ML", "1,098", "-14", "down"],
                  ["JD specific", "1,175", "+6", "up"],
                ].map(([label, value, delta, dir], i) => (
                  <motion.div
                    key={String(label)}
                    initial={{ opacity: 0, y: 6 }}
                    whileInView={{ opacity: 1, y: 0 }}
                    viewport={{ once: true, margin: "-60px" }}
                    transition={{ duration: 0.32, ease: easeOut, delay: i * 0.05 }}
                    className="lift rounded-lg border border-[var(--border)] p-3 bg-[var(--background)]/40"
                  >
                    <div className="text-[10px] font-mono uppercase tracking-[0.06em] text-[var(--muted-foreground)]">
                      {label}
                    </div>
                    <div className="font-mono text-2xl font-bold tracking-[-0.02em] mt-1">
                      {value}
                    </div>
                    <div
                      className={
                        "font-mono text-[11px] mt-1 " +
                        (dir === "down" ? "text-[var(--destructive)]" : "text-[var(--success)]")
                      }
                    >
                      {delta} / 7d
                    </div>
                  </motion.div>
                ))}
              </div>

              <div className="grid grid-cols-1 lg:grid-cols-[1.6fr_1fr] gap-3">
                <div className="rounded-lg border border-[var(--border)] p-4">
                  <div className="flex items-center justify-between mb-2">
                    <span className="text-xs font-semibold">ELO — 30 days</span>
                    <span className="font-mono text-[10px] text-[var(--muted-foreground)] uppercase tracking-[0.06em]">
                      overall
                    </span>
                  </div>
                  <Sparkline />
                </div>
                <div className="rounded-lg border border-[var(--border)] p-4">
                  <div className="flex items-center justify-between mb-3">
                    <span className="text-xs font-semibold">Weakest topics</span>
                    <Badge variant="primary">5</Badge>
                  </div>
                  <ul className="space-y-2 text-xs">
                    {[
                      ["multi-tenant-fhe", "2.1"],
                      ["gpu-memory-budget", "2.3"],
                      ["pq-key-rotation", "2.8"],
                      ["solana-cpi-auth", "3.0"],
                    ].map(([tag, rating]) => (
                      <li key={tag} className="flex items-center justify-between">
                        <span className="font-mono text-[11px]">{tag}</span>
                        <Badge variant={Number(rating) < 2.5 ? "danger" : "warning"}>{rating}</Badge>
                      </li>
                    ))}
                  </ul>
                </div>
              </div>
            </div>
          </div>
        </motion.div>
      </div>
    </section>
  );
}
