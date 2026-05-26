"use client";

import { useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { PageHeader } from "@/components/app/shell";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { useDrill } from "@/lib/useDrill";
import { Brain, Send, MicOff, X, Sparkles, ArrowRight, Loader2 } from "lucide-react";
import { toast } from "sonner";

const easeOut: [number, number, number, number] = [0.23, 1, 0.32, 1];

const CATEGORIES = [
  { id: "", label: "Planner picks" },
  { id: "system-design", label: "System design" },
  { id: "domain-crypto", label: "Crypto" },
  { id: "domain-ml", label: "ML" },
  { id: "domain-solana", label: "Solana" },
  { id: "jd-specific", label: "JD specific" },
  { id: "general", label: "General" },
];

export default function DrillPage() {
  const { state, start, submit, endQuestion, reset } = useDrill();
  const [qs, setQs] = useState(3);
  const [category, setCategory] = useState("");
  const [answer, setAnswer] = useState("");

  const idle = state.status === "idle";
  const awaiting = state.status === "awaiting-answer";
  const awaitingNext = state.status === "awaiting-next";

  async function onSubmit() {
    if (!answer.trim()) return;
    const text = answer;
    setAnswer("");
    try {
      await submit(text);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "submit failed");
    }
  }

  return (
    <>
      <PageHeader
        title="Drill"
        subtitle={
          state.status === "running"
            ? `Q${state.currentQuestionOrd ?? "—"}/${state.totalQuestions ?? "—"} · ${liveStatus(state)}`
            : state.status === "awaiting-answer"
              ? `Q${state.currentQuestionOrd}/${state.totalQuestions} · awaiting your answer`
              : state.status === "awaiting-next"
                ? `Q${state.currentQuestionOrd}/${state.totalQuestions} · ${liveStatus(state)}`
                : state.status === "done"
                  ? "Done. Start another?"
                  : state.status === "error"
                    ? "Error"
                    : "Ready"
        }
        action={
          !idle && (
            <Button variant="ghost" size="sm" onClick={reset}>
              <X className="mr-1 size-3.5" /> End session
            </Button>
          )
        }
      />

      <div className="p-4 sm:p-6 lg:p-8 max-w-3xl">
        {idle && (
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Configure</CardTitle>
            </CardHeader>
            <CardContent className="space-y-5">
              <div>
                <label className="text-xs font-mono uppercase tracking-[0.08em] text-[var(--muted-foreground)] mb-2 block">
                  Questions
                </label>
                <div className="flex gap-2 flex-wrap">
                  {[1, 2, 3, 5, 7].map((n) => (
                    <button
                      key={n}
                      onClick={() => setQs(n)}
                      className={
                        "px-3.5 h-8 rounded-md font-mono text-xs border transition-[transform,background-color,border-color] duration-150 [transition-timing-function:var(--ease-out)] active:scale-[0.97] " +
                        (qs === n
                          ? "bg-[var(--primary)] text-[var(--primary-foreground)] border-transparent"
                          : "bg-[var(--card)] text-[var(--foreground)] border-[var(--border)] hover:bg-[var(--secondary)]")
                      }
                    >
                      {n}Q
                    </button>
                  ))}
                </div>
              </div>

              <div>
                <label className="text-xs font-mono uppercase tracking-[0.08em] text-[var(--muted-foreground)] mb-2 block">
                  Category
                </label>
                <div className="flex gap-2 flex-wrap">
                  {CATEGORIES.map((c) => (
                    <button
                      key={c.id}
                      onClick={() => setCategory(c.id)}
                      className={
                        "px-3.5 h-8 rounded-md text-xs border transition-[transform,background-color,border-color] duration-150 [transition-timing-function:var(--ease-out)] active:scale-[0.97] " +
                        (category === c.id
                          ? "bg-[var(--primary)] text-[var(--primary-foreground)] border-transparent"
                          : "bg-[var(--card)] text-[var(--foreground)] border-[var(--border)] hover:bg-[var(--secondary)]")
                      }
                    >
                      {c.label}
                    </button>
                  ))}
                </div>
              </div>

              <Button
                size="lg"
                onClick={() => start({ qs, category: category || undefined })}
              >
                <Brain className="mr-1" /> Start drill
              </Button>
            </CardContent>
          </Card>
        )}

        {!idle && (
          <div className="space-y-5">
            {/* Plan strip */}
            {state.decision && (
              <motion.div
                initial={{ opacity: 0, y: 6 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.3, ease: easeOut }}
                className="flex items-center gap-3 flex-wrap text-xs"
              >
                <Badge variant="primary">{state.decision.category}</Badge>
                <span className="font-mono text-[var(--muted-foreground)]">
                  {state.decision.target_topic}
                </span>
                <span className="font-mono text-[var(--muted-foreground)]">
                  · target {state.decision.target_difficulty}
                </span>
                <span className="text-[var(--muted-foreground)] truncate">
                  — {state.decision.why}
                </span>
              </motion.div>
            )}

            {/* Pre-question thinking stages (no opening text yet) */}
            {!state.currentText && !state.streamingText && state.status === "running" && (
              <Card>
                <CardContent className="p-6">
                  <StageList state={state} />
                </CardContent>
              </Card>
            )}

            {/* Streaming typewriter — visible while opening tokens arrive */}
            {state.streamingText && state.exchanges.length === 0 && (
              <Card>
                <CardContent className="p-6">
                  <motion.div
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    transition={{ duration: 0.2, ease: easeOut }}
                  >
                    <p className="font-mono text-[10px] uppercase tracking-[0.1em] text-[var(--primary)] mb-2">
                      Interviewer
                    </p>
                    <p className="text-lg leading-relaxed">
                      {state.streamingText}
                      <Caret />
                    </p>
                  </motion.div>
                </CardContent>
              </Card>
            )}

            {/* Running transcript: opening + answer + each follow-up + answer */}
            {state.exchanges.map((ex, i) => {
              const isLast = i === state.exchanges.length - 1;
              return (
                <motion.div
                  key={i}
                  initial={{ opacity: 0, y: 6 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ duration: 0.25, ease: easeOut }}
                >
                  <Card>
                    <CardContent className="p-6 space-y-4">
                      <div>
                        <p className="font-mono text-[10px] uppercase tracking-[0.1em] text-[var(--primary)] mb-2">
                          {ex.questionKind === "opening"
                            ? "Interviewer"
                            : `Follow-up ${ex.followupIndex}`}
                        </p>
                        <p className={ex.questionKind === "opening" ? "text-lg leading-relaxed" : "text-base leading-relaxed"}>
                          {ex.question}
                        </p>
                      </div>

                      {ex.answer !== undefined && (
                        <div className="pt-3 border-t border-[var(--border)]">
                          <p className="font-mono text-[10px] uppercase tracking-[0.1em] text-[var(--muted-foreground)] mb-2">
                            You
                          </p>
                          <p className="text-sm leading-relaxed whitespace-pre-wrap text-[var(--foreground)]">
                            {ex.answer}
                          </p>
                          {isLast && awaitingNext && (
                            <p className="mt-3 flex items-center gap-2 text-xs font-mono uppercase tracking-[0.08em] text-[var(--muted-foreground)]">
                              <Loader2 className="size-3 animate-spin" />
                              {liveStatus(state)}
                            </p>
                          )}
                        </div>
                      )}
                    </CardContent>
                  </Card>
                </motion.div>
              );
            })}

            {/* Answer box */}
            {awaiting && (
              <Card>
                <CardContent className="p-4 space-y-3">
                  <Textarea
                    value={answer}
                    onChange={(e) => setAnswer(e.target.value)}
                    placeholder="Type your answer. Cmd/Ctrl+Enter to submit."
                    rows={6}
                    autoFocus
                    onKeyDown={(e) => {
                      if ((e.metaKey || e.ctrlKey) && e.key === "Enter") {
                        e.preventDefault();
                        onSubmit();
                      }
                    }}
                  />
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-1.5 text-[10px] font-mono uppercase tracking-[0.08em] text-[var(--muted-foreground)]">
                      <MicOff className="size-3" /> mic input coming
                    </div>
                    <div className="flex items-center gap-2">
                      <Button variant="ghost" size="sm" onClick={endQuestion}>
                        Skip follow-ups
                      </Button>
                      <Button onClick={onSubmit} disabled={!answer.trim()}>
                        <Send className="mr-1" /> Submit
                      </Button>
                    </div>
                  </div>
                </CardContent>
              </Card>
            )}

            {/* Verdict — blur-bridge entrance (Emil rule: mask imperfect transitions with subtle blur) */}
            <AnimatePresence>
              {state.verdict && (
                <motion.div
                  key="verdict"
                  initial={{ opacity: 0, y: 8, scale: 0.97, filter: "blur(6px)" }}
                  animate={{ opacity: 1, y: 0, scale: 1, filter: "blur(0px)" }}
                  exit={{ opacity: 0, y: -4, filter: "blur(4px)" }}
                  transition={{ duration: 0.32, ease: easeOut }}
                >
                  <Card>
                    <CardHeader className="flex-row items-center justify-between">
                      <CardTitle className="flex items-center gap-2">
                        <Sparkles className="size-4 text-[var(--primary)]" /> Judgment
                      </CardTitle>
                      <Badge
                        variant={
                          state.verdict.rating >= 4
                            ? "success"
                            : state.verdict.rating >= 3
                              ? "warning"
                              : "danger"
                        }
                      >
                        {state.verdict.rating}/5
                      </Badge>
                    </CardHeader>
                    <CardContent className="space-y-4 text-sm">
                      {state.verdict.strengths.length > 0 && (
                        <div>
                          <p className="text-xs font-mono uppercase tracking-[0.08em] text-[var(--success)] mb-1.5">
                            Strengths
                          </p>
                          <ul className="space-y-1">
                            {state.verdict.strengths.map((s, i) => (
                              <li key={i} className="text-[var(--foreground)]">
                                • {s}
                              </li>
                            ))}
                          </ul>
                        </div>
                      )}
                      {state.verdict.missed.length > 0 && (
                        <div>
                          <p className="text-xs font-mono uppercase tracking-[0.08em] text-[var(--destructive)] mb-1.5">
                            Missed
                          </p>
                          <ul className="space-y-1">
                            {state.verdict.missed.map((s, i) => (
                              <li key={i} className="text-[var(--foreground)]">
                                • {s}
                              </li>
                            ))}
                          </ul>
                        </div>
                      )}
                      {state.verdict.better_answer_sketch && (
                        <div>
                          <p className="text-xs font-mono uppercase tracking-[0.08em] text-[var(--muted-foreground)] mb-1.5">
                            Better answer sketch
                          </p>
                          <p className="text-[var(--foreground)] leading-relaxed">
                            {state.verdict.better_answer_sketch}
                          </p>
                        </div>
                      )}
                      {state.verdict.reading.length > 0 && (
                        <div>
                          <p className="text-xs font-mono uppercase tracking-[0.08em] text-[var(--muted-foreground)] mb-1.5">
                            Reading
                          </p>
                          <ul className="space-y-1">
                            {state.verdict.reading.map((r, i) => (
                              <li key={i}>
                                <span className="font-semibold">{r.topic}</span>
                                {r.why && <span className="text-[var(--muted-foreground)]"> — {r.why}</span>}
                                {r.optional_url && (
                                  <>
                                    {" "}
                                    <a
                                      href={r.optional_url}
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
                        </div>
                      )}
                      {state.verdict.topic_tags.length > 0 && (
                        <div className="flex flex-wrap gap-1.5 pt-1">
                          {state.verdict.topic_tags.map((t) => (
                            <Badge key={t} variant="outline">
                              {t}
                            </Badge>
                          ))}
                        </div>
                      )}
                      {state.eloUpdate && (
                        <div className="pt-2 border-t border-[var(--border)] flex items-center justify-between text-xs font-mono">
                          <span className="text-[var(--muted-foreground)]">
                            ELO · {state.eloUpdate.category}
                          </span>
                          <span>
                            {state.eloUpdate.before}{" "}
                            <ArrowRight className="inline size-3 mx-1" />{" "}
                            {state.eloUpdate.after}{" "}
                            <span
                              className={
                                state.eloUpdate.delta >= 0
                                  ? "text-[var(--success)]"
                                  : "text-[var(--destructive)]"
                              }
                            >
                              ({state.eloUpdate.delta >= 0 ? "+" : ""}
                              {state.eloUpdate.delta})
                            </span>
                          </span>
                        </div>
                      )}
                    </CardContent>
                  </Card>
                </motion.div>
              )}
            </AnimatePresence>

            {state.status === "done" && (
              <div className="flex justify-end gap-2">
                {state.sessionId && (
                  <a
                    href={`/replay/${state.sessionId}`}
                    className="inline-flex items-center justify-center gap-1 rounded-md border border-[var(--border)] px-3 h-9 text-sm hover:bg-[var(--secondary)] transition-colors"
                  >
                    <Sparkles className="size-3.5" /> Review + analyze
                  </a>
                )}
                <Button variant="outline" onClick={reset}>New drill</Button>
              </div>
            )}

            {state.status === "error" && (
              <Card className="border-[var(--destructive)]/30">
                <CardContent className="p-4 text-sm text-[var(--destructive)]">
                  {state.error}
                </CardContent>
              </Card>
            )}
          </div>
        )}
      </div>
    </>
  );
}

function liveStatus(state: ReturnType<typeof useDrill>["state"]): string {
  // Walk events in reverse to find the most informative one
  for (let i = state.events.length - 1; i >= 0; i--) {
    const e = state.events[i];
    switch (e.type) {
      case "planner:thinking": return "planner thinking…";
      case "planner:decision": return "planner decided";
      case "rag:retrieve":     return "retrieving context…";
      case "interviewer:thinking": return "drafting question…";
      case "interviewer:deciding": return "deciding follow-up…";
      case "interviewer:done_with_question": return "wrapping up…";
      case "judge:grading":  return "judge grading…";
      case "judge:verdict":  return "judge done";
      case "elo:update":     return "updating ELO…";
      case "question:end":   return "question complete";
    }
  }
  return "starting…";
}

function Caret() {
  return <span aria-hidden className="caret-blink" />;
}

function StageList({ state }: { state: ReturnType<typeof useDrill>["state"] }) {
  const stages = [
    { key: "planner:decision", label: "Planner" },
    { key: "interviewer:opening", label: "Interviewer" },
    { key: "judge:verdict", label: "Judge" },
  ];
  return (
    <ul className="space-y-2">
      {stages.map((s) => {
        const done = state.events.some((e) => e.type === s.key);
        return (
          <li key={s.key} className="flex items-center gap-3 text-sm">
            <span
              className={
                "grid place-items-center size-4 rounded-full border " +
                (done
                  ? "bg-[var(--primary)] border-[var(--primary)]"
                  : "border-[var(--border)]")
              }
            >
              {done && <span className="size-1.5 rounded-full bg-[var(--primary-foreground)]" />}
            </span>
            <span className={done ? "text-foreground" : "text-[var(--muted-foreground)]"}>
              {s.label}
            </span>
          </li>
        );
      })}
    </ul>
  );
}
