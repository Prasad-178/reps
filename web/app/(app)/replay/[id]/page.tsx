"use client";

import { use, useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { PageHeader } from "@/components/app/shell";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useApi } from "@/lib/useApi";
import { api, type SessionCritique } from "@/lib/api";
import { Sparkles, Loader2, AlertTriangle, TrendingUp, BookOpen, Target } from "lucide-react";
import { toast } from "sonner";

const easeOut: [number, number, number, number] = [0.23, 1, 0.32, 1];

export default function ReplayPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const { data, loading, error } = useApi(() => api.replay(id), [id]);

  const [critique, setCritique] = useState<SessionCritique | null>(null);
  const [analyzing, setAnalyzing] = useState(false);

  async function onAnalyze() {
    setAnalyzing(true);
    setCritique(null);
    try {
      const c = await api.analyzeSession(id);
      setCritique(c);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "analyze failed");
    } finally {
      setAnalyzing(false);
    }
  }

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
        action={
          data && (
            <Button onClick={onAnalyze} disabled={analyzing} variant="outline">
              {analyzing ? (
                <Loader2 className="mr-1 size-4 animate-spin" />
              ) : (
                <Sparkles className="mr-1 size-4" />
              )}
              {critique ? "Re-analyze" : "Analyze session"}
            </Button>
          )
        }
      />
      <div className="p-4 sm:p-6 lg:p-8 max-w-3xl space-y-4">
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

        <AnimatePresence>
          {analyzing && !critique && (
            <motion.div
              key="analyzing"
              initial={{ opacity: 0, y: 6 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -4 }}
              transition={{ duration: 0.25, ease: easeOut }}
            >
              <Card className="border-[var(--primary)]/30 bg-[var(--primary)]/[0.03]">
                <CardContent className="p-5 flex items-center gap-3 text-sm">
                  <Loader2 className="size-4 animate-spin text-[var(--primary)]" />
                  <span className="text-[var(--muted-foreground)]">
                    Reading the whole session, pattern-matching across questions…
                  </span>
                </CardContent>
              </Card>
            </motion.div>
          )}
        </AnimatePresence>

        <AnimatePresence>
          {critique && (
            <motion.div
              key="critique"
              initial={{ opacity: 0, y: 8, filter: "blur(6px)" }}
              animate={{ opacity: 1, y: 0, filter: "blur(0px)" }}
              exit={{ opacity: 0, y: -4, filter: "blur(4px)" }}
              transition={{ duration: 0.35, ease: easeOut }}
              className="space-y-4"
            >
              <CritiquePanel critique={critique} />
            </motion.div>
          )}
        </AnimatePresence>

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

function CritiquePanel({ critique }: { critique: SessionCritique }) {
  const verdictClass =
    critique.verdict === "good"
      ? "border-[var(--success)]/40 bg-[var(--success)]/[0.04]"
      : critique.verdict === "bad"
        ? "border-[var(--destructive)]/40 bg-[var(--destructive)]/[0.04]"
        : "border-[var(--primary)]/30 bg-[var(--primary)]/[0.04]";

  return (
    <Card className={verdictClass}>
      <CardContent className="p-6 space-y-6">
        <div className="flex items-start justify-between gap-4">
          <div className="flex-1">
            <p className="font-mono text-[10px] uppercase tracking-[0.1em] text-[var(--primary)] mb-2 flex items-center gap-2">
              <Sparkles className="size-3" /> Session analysis
            </p>
            <p className="text-lg leading-relaxed font-semibold">{critique.headline}</p>
          </div>
          <Badge
            variant={
              critique.verdict === "good"
                ? "success"
                : critique.verdict === "bad"
                  ? "danger"
                  : "warning"
            }
          >
            {critique.overall_rating.toFixed(1)}/5
          </Badge>
        </div>

        {critique.strengths.length > 0 && (
          <Section icon={<TrendingUp className="size-3.5 text-[var(--success)]" />} title="What worked">
            <ul className="space-y-1.5 text-sm">
              {critique.strengths.map((s, i) => (
                <li key={i} className="text-[var(--foreground)]">+ {s}</li>
              ))}
            </ul>
          </Section>
        )}

        {critique.patterns.length > 0 && (
          <Section
            icon={<AlertTriangle className="size-3.5 text-[var(--warning,var(--primary))]" />}
            title="Patterns that hurt you"
          >
            <div className="space-y-4">
              {critique.patterns.map((p, i) => (
                <div key={i} className="rounded-md border border-[var(--border)] p-4 bg-[var(--card)]">
                  <p className="font-semibold text-sm mb-1.5">{p.name}</p>
                  <p className="text-sm text-[var(--muted-foreground)] mb-2 leading-relaxed">
                    <span className="font-mono text-[10px] uppercase tracking-[0.08em] mr-1.5">evidence</span>
                    {p.evidence}
                  </p>
                  <p className="text-sm leading-relaxed">
                    <span className="font-mono text-[10px] uppercase tracking-[0.08em] mr-1.5 text-[var(--primary)]">fix</span>
                    {p.fix}
                  </p>
                </div>
              ))}
            </div>
          </Section>
        )}

        {critique.growth_edge.length > 0 && (
          <Section icon={<Target className="size-3.5 text-[var(--primary)]" />} title="Do this before next session">
            <ol className="space-y-2 text-sm">
              {critique.growth_edge.map((g, i) => (
                <li key={i} className="flex gap-3">
                  <span className="font-mono text-xs text-[var(--muted-foreground)] mt-0.5 w-4 shrink-0">
                    {i + 1}.
                  </span>
                  <div>
                    <p className="font-semibold leading-snug">{g.action}</p>
                    <p className="text-xs text-[var(--muted-foreground)] mt-0.5">{g.why}</p>
                  </div>
                </li>
              ))}
            </ol>
          </Section>
        )}

        {critique.drill_again.length > 0 && (
          <Section title="Drill these again">
            <div className="flex flex-wrap gap-1.5">
              {critique.drill_again.map((t, i) => (
                <Badge key={i} variant="outline">{t}</Badge>
              ))}
            </div>
          </Section>
        )}

        {critique.reading.length > 0 && (
          <Section icon={<BookOpen className="size-3.5 text-[var(--muted-foreground)]" />} title="Reading">
            <ul className="space-y-1.5 text-sm">
              {critique.reading.map((r, i) => (
                <li key={i}>
                  <span className="font-semibold">{r.topic}</span>
                  {r.why && <span className="text-[var(--muted-foreground)]"> — {r.why}</span>}
                  {r.url && (
                    <>
                      {" "}
                      <a
                        href={r.url}
                        target="_blank"
                        rel="noreferrer"
                        className="text-[var(--primary)] underline-offset-2 hover:underline"
                      >
                        open
                      </a>
                    </>
                  )}
                </li>
              ))}
            </ul>
          </Section>
        )}
      </CardContent>
    </Card>
  );
}

function Section({
  icon,
  title,
  children,
}: {
  icon?: React.ReactNode;
  title: string;
  children: React.ReactNode;
}) {
  return (
    <div>
      <p className="font-mono text-[10px] uppercase tracking-[0.1em] text-[var(--muted-foreground)] mb-2 flex items-center gap-1.5">
        {icon}
        {title}
      </p>
      {children}
    </div>
  );
}
