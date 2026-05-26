"use client";

import Link from "next/link";
import { motion } from "framer-motion";
import { PageHeader } from "@/components/app/shell";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { useApi } from "@/lib/useApi";
import { api } from "@/lib/api";
import { formatRelative } from "@/lib/utils";
import { ArrowRight } from "lucide-react";

const easeOut: [number, number, number, number] = [0.23, 1, 0.32, 1];

export default function HistoryPage() {
  const { data, loading } = useApi(() => api.sessions(), []);

  return (
    <>
      <PageHeader eyebrow="Archive" title="History" subtitle={data ? `${data.length} sessions` : "Loading…"} />
      <div className="p-4 sm:p-6 lg:p-8 max-w-4xl">
        {loading && (
          <div className="space-y-2">
            {Array.from({ length: 5 }).map((_, i) => (
              <Skeleton key={i} className="h-14" />
            ))}
          </div>
        )}
        {data?.length === 0 && (
          <Card>
            <CardContent className="p-6 text-center text-sm text-[var(--muted-foreground)]">
              No sessions yet.
            </CardContent>
          </Card>
        )}
        <Card>
          <CardContent className="p-0 divide-y divide-[var(--border)]">
            {data?.map((s, i) => (
              <motion.div
                key={s.id}
                initial={{ opacity: 0, y: 4 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.22, ease: easeOut, delay: Math.min(i, 12) * 0.025 }}
              >
              <Link
                href={`/replay/${s.id.slice(0, 8)}`}
                className="grid grid-cols-[auto_1fr_auto_auto] gap-4 items-center px-4 py-3 hover:bg-[var(--secondary)] transition-colors duration-150 [transition-timing-function:var(--ease-out)]"
              >
                <span className="font-mono text-[12px] text-[var(--foreground)]">
                  {s.id.slice(0, 8)}
                </span>
                <span className="text-sm text-[var(--muted-foreground)] truncate">
                  {new Date(s.started_at * 1000).toLocaleString()} ·{" "}
                  {formatRelative(s.started_at)}
                </span>
                <span className="font-mono text-xs text-[var(--muted-foreground)]">
                  {s.q_count}Q · {s.mode}
                </span>
                <Badge
                  variant={
                    s.mean_rating >= 3.5
                      ? "success"
                      : s.mean_rating >= 2.5
                        ? "warning"
                        : s.mean_rating > 0
                          ? "danger"
                          : "default"
                  }
                >
                  {s.mean_rating > 0 ? s.mean_rating.toFixed(2) : "—"}
                </Badge>
                <ArrowRight className="size-3.5 text-[var(--muted-foreground)] col-span-4 sm:col-auto justify-self-end" />
              </Link>
              </motion.div>
            ))}
          </CardContent>
        </Card>
      </div>
    </>
  );
}
