"use client";

import { useState } from "react";
import { motion } from "framer-motion";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useApi } from "@/lib/useApi";
import { api, type InsightPanel } from "@/lib/api";
import { Sparkles, RefreshCcw, Loader2 } from "lucide-react";
import { cn } from "@/lib/utils";
import { toast } from "sonner";

const easeOut: [number, number, number, number] = [0.23, 1, 0.32, 1];

const SEVERITY_CLASS: Record<InsightPanel["severity"], string> = {
  good: "border-[color-mix(in_oklch,var(--success)_25%,var(--border))]",
  warn: "border-[color-mix(in_oklch,var(--warning)_25%,var(--border))]",
  bad:  "border-[color-mix(in_oklch,var(--destructive)_25%,var(--border))]",
  info: "border-[var(--border)]",
};

const SEVERITY_BADGE: Record<InsightPanel["severity"], "success" | "warning" | "danger" | "primary"> = {
  good: "success",
  warn: "warning",
  bad:  "danger",
  info: "primary",
};

export function Insights() {
  const [refreshing, setRefreshing] = useState(false);
  const [tick, setTick] = useState(0);
  const { data, loading, error } = useApi(() => api.insights(false), [tick]);

  async function refresh() {
    setRefreshing(true);
    try {
      await api.insights(true);
      setTick((t) => t + 1);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "regen failed");
    } finally {
      setRefreshing(false);
    }
  }

  return (
    <Card className="overflow-hidden">
      <CardHeader className="flex-row items-start justify-between gap-4">
        <div className="flex items-start gap-3 min-w-0">
          <div
            className="grid place-items-center size-9 rounded-md text-[var(--primary-foreground)] shrink-0"
            style={{ background: "var(--gradient-accent)" }}
          >
            <Sparkles className="size-4" />
          </div>
          <div className="min-w-0">
            <CardTitle>Analyst</CardTitle>
            <p className="text-xs text-[var(--muted-foreground)] mt-0.5">
              AI-generated patterns from your drill history.{" "}
              {data?.cached && (
                <span className="font-mono uppercase tracking-[0.06em] text-[10px]">
                  · cached
                </span>
              )}
            </p>
          </div>
        </div>
        <Button variant="ghost" size="sm" onClick={refresh} disabled={refreshing || loading}>
          {refreshing ? <Loader2 className="animate-spin" /> : <RefreshCcw />}
          Regenerate
        </Button>
      </CardHeader>

      <CardContent className="space-y-3">
        {data?.summary && (
          <p className="text-sm leading-relaxed text-[var(--foreground)] border-l-2 border-[var(--primary)] pl-3">
            {data.summary}
          </p>
        )}

        {error && (
          <p className="text-sm text-[var(--destructive)]">{String(error.message)}</p>
        )}

        {loading && (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            {Array.from({ length: 4 }).map((_, i) => (
              <Skeleton key={i} className="h-32" />
            ))}
          </div>
        )}

        {!loading && data?.panels && data.panels.length === 0 && (
          <p className="text-sm text-[var(--muted-foreground)]">
            No insights yet — drill a few times and come back.
          </p>
        )}

        {data?.panels && data.panels.length > 0 && (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            {data.panels.map((p, i) => (
              <motion.div
                key={p.id || i}
                initial={{ opacity: 0, y: 6 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.35, ease: easeOut, delay: i * 0.04 }}
              >
                <Panel panel={p} />
              </motion.div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function Panel({ panel }: { panel: InsightPanel }) {
  return (
    <div
      className={cn(
        "lift rounded-lg border bg-[var(--background)]/60 p-4 h-full flex flex-col gap-2",
        SEVERITY_CLASS[panel.severity]
      )}
    >
      <div className="flex items-start justify-between gap-2">
        <h4 className="text-sm font-semibold tracking-[-0.01em] leading-tight">{panel.title}</h4>
        <Badge variant={SEVERITY_BADGE[panel.severity]}>{panel.severity}</Badge>
      </div>

      {panel.headline && (
        <p className="text-base font-semibold tracking-[-0.015em] leading-snug">
          {panel.headline}
        </p>
      )}

      {panel.kind === "stat-row" && panel.stats && (
        <div className="grid grid-cols-2 gap-2 mt-1">
          {panel.stats.map((s, i) => (
            <div key={i} className="rounded border border-[var(--border)] px-2 py-1.5">
              <div className="text-[10px] font-mono uppercase tracking-[0.06em] text-[var(--muted-foreground)]">
                {s.label}
              </div>
              <div className="font-mono text-lg font-bold">
                {s.value}
                {s.unit && <span className="text-xs text-[var(--muted-foreground)]"> {s.unit}</span>}
              </div>
              {typeof s.delta === "number" && s.delta !== 0 && (
                <div
                  className={cn(
                    "text-[10px] font-mono",
                    s.delta < 0 ? "text-[var(--destructive)]" : "text-[var(--success)]"
                  )}
                >
                  {s.delta > 0 ? "+" : ""}
                  {s.delta}
                  {s.unit ?? ""}
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      {panel.kind === "tag-cloud" && panel.tags && panel.tags.length > 0 && (
        <div className="flex flex-wrap gap-1.5 mt-1">
          {panel.tags.map((t) => (
            <Badge key={t} variant="outline" className="font-mono normal-case tracking-normal text-[11px]">
              {t}
            </Badge>
          ))}
        </div>
      )}

      {panel.kind === "list" && panel.items && panel.items.length > 0 && (
        <ul className="space-y-1 mt-1">
          {panel.items.map((it, i) => (
            <li key={i} className="text-sm text-[var(--foreground)]">• {it}</li>
          ))}
        </ul>
      )}

      {panel.body && (
        <p className="text-sm text-[var(--muted-foreground)] leading-relaxed mt-1">{panel.body}</p>
      )}

      {panel.suggestion && (
        <div className="mt-auto pt-2 border-t border-dashed border-[var(--border)]">
          <p className="text-[10px] font-mono uppercase tracking-[0.08em] text-[var(--primary)] mb-1">
            Next action
          </p>
          <p className="text-sm leading-snug">{panel.suggestion}</p>
        </div>
      )}
    </div>
  );
}
