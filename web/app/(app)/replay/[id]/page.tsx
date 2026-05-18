"use client";

import { use } from "react";
import { PageHeader } from "@/components/app/shell";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { useApi } from "@/lib/useApi";
import { api } from "@/lib/api";
import { formatRelative } from "@/lib/utils";

export default function ReplayPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const { data, loading, error } = useApi(() => api.replay(id), [id]);

  return (
    <>
      <PageHeader
        title={`Session ${id.slice(0, 8)}`}
        subtitle={
          data?.session
            ? `${new Date(data.session.started_at * 1000).toLocaleString()} · ${data.session.mode} · ${data.questions.length} questions`
            : loading
              ? "Loading…"
              : "Replay"
        }
      />
      <div className="p-6 sm:p-8 max-w-3xl space-y-4">
        {loading && (
          <div className="space-y-3">
            {Array.from({ length: 2 }).map((_, i) => (
              <Skeleton key={i} className="h-48" />
            ))}
          </div>
        )}

        {error && (
          <Card className="border-[var(--destructive)]/30">
            <CardContent className="p-4 text-sm text-[var(--destructive)]">
              {String(error.message)}
            </CardContent>
          </Card>
        )}

        {data?.questions.map((q) => (
          <Card key={q.ord}>
            <CardContent className="p-6 space-y-4">
              <div className="flex items-center gap-2 flex-wrap text-xs">
                <Badge variant="primary">Q{q.ord}</Badge>
                <Badge variant="outline">{q.category}</Badge>
                <span className="font-mono text-[var(--muted-foreground)] truncate">
                  {q.topic}
                </span>
                <span className="font-mono text-[var(--muted-foreground)] ml-auto">
                  target {q.target_elo}
                </span>
              </div>
              {q.rationale && (
                <p className="text-xs text-[var(--muted-foreground)] italic">{q.rationale}</p>
              )}

              <div className="space-y-3">
                {q.turns.map((t) => (
                  <div
                    key={t.ord}
                    className={
                      t.speaker === "interviewer"
                        ? "border-l-2 border-[var(--primary)] pl-4"
                        : "border-l-2 border-[var(--border)] pl-4"
                    }
                  >
                    <p className="text-[10px] font-mono uppercase tracking-[0.08em] text-[var(--muted-foreground)] mb-1">
                      {t.speaker} · {t.kind}
                    </p>
                    <p className="text-sm leading-relaxed whitespace-pre-wrap">{t.text || "—"}</p>
                  </div>
                ))}
              </div>

              {q.judgment && (
                <div className="pt-4 border-t border-[var(--border)] space-y-3 text-sm">
                  <div className="flex items-center justify-between">
                    <span className="text-xs font-mono uppercase tracking-[0.08em] text-[var(--muted-foreground)]">
                      Judgment
                    </span>
                    <Badge
                      variant={
                        q.judgment.rating >= 4
                          ? "success"
                          : q.judgment.rating >= 3
                            ? "warning"
                            : "danger"
                      }
                    >
                      {q.judgment.rating}/5
                    </Badge>
                  </div>
                  {q.judgment.strengths.length > 0 && (
                    <ul className="space-y-1">
                      {q.judgment.strengths.map((s, i) => (
                        <li key={i} className="text-[var(--success)]">+ {s}</li>
                      ))}
                    </ul>
                  )}
                  {q.judgment.missed.length > 0 && (
                    <ul className="space-y-1">
                      {q.judgment.missed.map((s, i) => (
                        <li key={i} className="text-[var(--destructive)]">− {s}</li>
                      ))}
                    </ul>
                  )}
                  {q.judgment.better_sketch && (
                    <p className="text-[var(--foreground)] leading-relaxed italic">
                      {q.judgment.better_sketch}
                    </p>
                  )}
                </div>
              )}
            </CardContent>
          </Card>
        ))}
      </div>
    </>
  );
}
