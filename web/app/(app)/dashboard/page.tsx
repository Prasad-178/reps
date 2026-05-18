"use client";

import Link from "next/link";
import { motion } from "framer-motion";
import {
  LineChart,
  Line,
  AreaChart,
  Area,
  ResponsiveContainer,
  XAxis,
  YAxis,
  Tooltip,
  CartesianGrid,
} from "recharts";
import { PageHeader } from "@/components/app/shell";
import { Insights } from "@/components/app/insights";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useApi } from "@/lib/useApi";
import { api } from "@/lib/api";
import { ArrowRight, Brain } from "lucide-react";
import { formatRelative } from "@/lib/utils";

const easeOut: [number, number, number, number] = [0.23, 1, 0.32, 1];

const CATEGORY_LABEL: Record<string, string> = {
  "system-design": "System design",
  "domain-crypto": "Domain · Crypto",
  "domain-ml":     "Domain · ML",
  "domain-solana": "Domain · Solana",
  "jd-specific":   "JD specific",
  "general":       "General",
};

export default function DashboardPage() {
  const stats    = useApi(() => api.stats(), []);
  const elo      = useApi(() => api.elo(30), []);
  const sessions = useApi(() => api.sessions(), []);

  // Reduce elo points into per-category time series for the chart.
  const series = (() => {
    if (!elo.data) return [] as { ts: number; t: string; overall: number }[];
    type Row = { ts: number; t: string; values: Record<string, number> };
    const buckets: Row[] = [];
    const running: Record<string, number> = {};
    for (const p of elo.data) {
      running[p.category] = p.rating;
      const ts = p.at * 1000;
      buckets.push({
        ts,
        t: new Date(ts).toLocaleDateString(undefined, { month: "short", day: "numeric" }),
        values: { ...running },
      });
    }
    // Flatten to one overall avg per row across known categories so chart isn't multi-line crowded
    return buckets.map((b) => ({
      ts: b.ts,
      t: b.t,
      overall:
        Object.values(b.values).reduce((a, c) => a + c, 0) /
        Math.max(1, Object.keys(b.values).length),
    }));
  })();

  const isEmpty =
    !stats.loading &&
    stats.data &&
    Object.values(stats.data.by_category).every((c) => c.rating === 1200 && c.delta_7d === 0) &&
    (!sessions.data || sessions.data.length === 0);

  return (
    <>
      <PageHeader
        title="Dashboard"
        subtitle={
          sessions.data
            ? `${sessions.data.length} sessions · mean ${
                sessions.data.length
                  ? (
                      sessions.data.reduce((a, c) => a + (c.mean_rating || 0), 0) /
                      sessions.data.length
                    ).toFixed(2)
                  : "—"
              }`
            : "Loading…"
        }
        action={
          <Button asChild>
            <Link href="/drill">
              <Brain className="mr-1" /> Start drill
            </Link>
          </Button>
        }
      />

      <div className="p-6 sm:p-8 space-y-6 max-w-[1200px]">
        {isEmpty && <EmptyState />}

        {/* KPI row */}
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-3">
          {stats.loading
            ? Array.from({ length: 4 }).map((_, i) => (
                <Skeleton key={i} className="h-[88px]" />
              ))
            : stats.data &&
              ([
                ["Overall ELO", String(stats.data.overall), null] as const,
                ...Object.entries(stats.data.by_category)
                  .slice(0, 3)
                  .map(
                    ([k, v]) =>
                      [
                        CATEGORY_LABEL[k] ?? k,
                        v.rating.toLocaleString(),
                        v.delta_7d,
                      ] as const
                  ),
              ].map(([label, value, delta], i) => (
                <motion.div
                  key={String(label)}
                  initial={{ opacity: 0, y: 6 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ duration: 0.3, ease: easeOut, delay: i * 0.04 }}
                >
                  <Card>
                    <CardContent className="p-4">
                      <div className="text-[10px] font-mono uppercase tracking-[0.08em] text-[var(--muted-foreground)]">
                        {label}
                      </div>
                      <div className="font-mono text-3xl font-bold tracking-[-0.02em] mt-1">
                        {value}
                      </div>
                      {delta !== null && (
                        <div
                          className={`font-mono text-xs mt-1 ${
                            (delta as number) < 0
                              ? "text-[var(--destructive)]"
                              : (delta as number) > 0
                                ? "text-[var(--success)]"
                                : "text-[var(--muted-foreground)]"
                          }`}
                        >
                          {(delta as number) >= 0 ? "+" : ""}
                          {delta} / 7d
                        </div>
                      )}
                    </CardContent>
                  </Card>
                </motion.div>
              )))}
        </div>

        {/* Chart + weakest */}
        <div className="grid grid-cols-1 lg:grid-cols-[1.6fr_1fr] gap-3">
          <Card>
            <CardHeader className="flex-row items-center justify-between">
              <CardTitle>ELO — last 30 days</CardTitle>
              <span className="font-mono text-[10px] text-[var(--muted-foreground)] uppercase tracking-[0.06em]">
                overall avg
              </span>
            </CardHeader>
            <CardContent className="pt-2 h-[220px]">
              {elo.loading ? (
                <Skeleton className="h-full w-full" />
              ) : series.length === 0 ? (
                <div className="grid place-items-center h-full text-sm text-[var(--muted-foreground)]">
                  No ELO history yet — finish a drill to see your curve.
                </div>
              ) : (
                <ResponsiveContainer width="100%" height="100%">
                  <AreaChart data={series} margin={{ left: -10, right: 8, top: 4, bottom: 0 }}>
                    <defs>
                      <linearGradient id="grad-elo" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="0%" stopColor="var(--primary)" stopOpacity={0.5} />
                        <stop offset="100%" stopColor="var(--primary)" stopOpacity={0} />
                      </linearGradient>
                    </defs>
                    <CartesianGrid stroke="var(--border)" strokeDasharray="3 3" vertical={false} />
                    <XAxis
                      dataKey="t"
                      tick={{ fontSize: 11, fill: "var(--muted-foreground)" }}
                      tickLine={false}
                      axisLine={false}
                    />
                    <YAxis
                      tick={{ fontSize: 11, fill: "var(--muted-foreground)" }}
                      tickLine={false}
                      axisLine={false}
                      width={42}
                      domain={["dataMin - 10", "dataMax + 10"]}
                    />
                    <Tooltip
                      contentStyle={{
                        background: "var(--card)",
                        border: "1px solid var(--border)",
                        borderRadius: 8,
                        fontFamily: "var(--font-mono)",
                        fontSize: 12,
                      }}
                      labelStyle={{ color: "var(--muted-foreground)" }}
                    />
                    <Area
                      type="monotone"
                      dataKey="overall"
                      stroke="var(--primary)"
                      strokeWidth={2}
                      fill="url(#grad-elo)"
                    />
                  </AreaChart>
                </ResponsiveContainer>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex-row items-center justify-between">
              <CardTitle>Weakest topics</CardTitle>
              {stats.data && (
                <Badge variant="primary">{stats.data.weakest.length}</Badge>
              )}
            </CardHeader>
            <CardContent>
              {stats.loading ? (
                <div className="space-y-2">
                  {Array.from({ length: 4 }).map((_, i) => (
                    <Skeleton key={i} className="h-8" />
                  ))}
                </div>
              ) : !stats.data?.weakest.length ? (
                <p className="text-sm text-[var(--muted-foreground)]">
                  No tagged weak topics yet — they appear as you drill.
                </p>
              ) : (
                <ul className="divide-y divide-[var(--border)]">
                  {stats.data.weakest.map((w) => (
                    <li
                      key={w.tag}
                      className="flex items-center justify-between py-2.5 text-sm"
                    >
                      <span className="font-mono text-[12px]">{w.tag}</span>
                      <span className="flex items-center gap-2 text-[var(--muted-foreground)] text-xs">
                        {w.hits} hits
                        <Badge
                          variant={
                            w.mean_rating < 2.5
                              ? "danger"
                              : w.mean_rating < 3.5
                                ? "warning"
                                : "success"
                          }
                        >
                          {w.mean_rating.toFixed(1)}
                        </Badge>
                      </span>
                    </li>
                  ))}
                </ul>
              )}
            </CardContent>
          </Card>
        </div>

        {/* AI insights */}
        <Insights />

        {/* Recent sessions */}
        <Card>
          <CardHeader className="flex-row items-center justify-between">
            <CardTitle>Recent sessions</CardTitle>
            <Button asChild variant="ghost" size="sm">
              <Link href="/history">
                View all <ArrowRight className="ml-1 size-3.5" />
              </Link>
            </Button>
          </CardHeader>
          <CardContent>
            {sessions.loading ? (
              <div className="space-y-2">
                {Array.from({ length: 3 }).map((_, i) => (
                  <Skeleton key={i} className="h-10" />
                ))}
              </div>
            ) : !sessions.data?.length ? (
              <p className="text-sm text-[var(--muted-foreground)]">
                No drills yet — run one from{" "}
                <Link href="/drill" className="underline">
                  Drill
                </Link>
                .
              </p>
            ) : (
              <ul className="divide-y divide-[var(--border)]">
                {sessions.data.slice(0, 6).map((s) => (
                  <li
                    key={s.id}
                    className="grid grid-cols-[1fr_auto_auto_auto] items-center gap-4 py-2.5 text-sm"
                  >
                    <Link
                      href={`/replay/${s.id.slice(0, 8)}`}
                      className="font-mono text-[12px] truncate hover:text-[var(--primary)] transition-colors"
                    >
                      {s.id.slice(0, 8)}
                    </Link>
                    <span className="text-[var(--muted-foreground)] text-xs">
                      {formatRelative(s.started_at)}
                    </span>
                    <span className="font-mono text-xs text-[var(--muted-foreground)]">
                      {s.q_count}Q
                    </span>
                    <Badge variant={s.mean_rating >= 3.5 ? "success" : s.mean_rating >= 2.5 ? "warning" : s.mean_rating > 0 ? "danger" : "default"}>
                      {s.mean_rating > 0 ? s.mean_rating.toFixed(1) : "—"}
                    </Badge>
                  </li>
                ))}
              </ul>
            )}
          </CardContent>
        </Card>
      </div>
    </>
  );
}

function EmptyState() {
  return (
    <Card className="border-dashed">
      <CardContent className="p-8 text-center space-y-3">
        <p className="font-mono text-[10px] uppercase tracking-[0.1em] text-[var(--primary)]">
          New here?
        </p>
        <h2 className="text-xl font-semibold tracking-[-0.015em]">
          You haven&apos;t drilled yet.
        </h2>
        <p className="text-sm text-[var(--muted-foreground)] max-w-md mx-auto">
          Start by ingesting your resume + GitHub + a JD, rebuild the profile, then run a
          drill. Numbers populate from there.
        </p>
        <div className="flex items-center justify-center gap-2 pt-2">
          <Button asChild>
            <Link href="/sources">Add a source</Link>
          </Button>
          <Button variant="outline" asChild>
            <Link href="/drill">Try a drill</Link>
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
