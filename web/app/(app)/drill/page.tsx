"use client";

import { useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { PageHeader } from "@/components/app/shell";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { useDrill, type DrillState } from "@/lib/useDrill";
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

  const headerSubtitle =
    state.status === "running"
      ? liveStatus(state)
      : state.status === "awaiting-answer"
        ? "awaiting your answer"
        : state.status === "awaiting-next"
          ? liveStatus(state)
          : state.status === "done"
            ? "session complete"
            : state.status === "error"
              ? "error"
              : "calibrate, then begin";

  return (
    <>
      <PageHeader
        eyebrow={
          state.totalQuestions
            ? `Q ${state.currentQuestionOrd ?? "—"} / ${state.totalQuestions}`
            : "Session"
        }
        title="Drill"
        subtitle={headerSubtitle}
        action={
          !idle && (
            <Button variant="ghost" size="sm" onClick={reset}>
              <X className="mr-1 size-3.5" /> End session
            </Button>
          )
        }
      />

      <div className="p-4 sm:p-6 lg:p-10 max-w-3xl mx-auto">
        {idle && <StartCard qs={qs} setQs={setQs} category={category} setCategory={setCategory} onStart={() => start({ qs, category: category || undefined })} />}

        {!idle && (
          <div className="space-y-6">
            {/* Signature: session timeline */}
            <SessionTimeline state={state} />

            {/* Planner verdict strip */}
            {state.decision && <PlanStrip decision={state.decision} />}

            {/* Pre-question thinking stages (no opening text yet) */}
            {!state.currentText && !state.streamingText && state.status === "running" && (
              <StageCard state={state} />
            )}

            {/* Streaming typewriter — visible while opening tokens arrive */}
            {state.streamingText && state.exchanges.length === 0 && (
              <QuestionCard
                kind="opening"
                text={state.streamingText}
                streaming
              />
            )}

            {/* Running transcript: opening + answer + each follow-up + answer */}
            {state.exchanges.map((ex, i) => {
              const isLast = i === state.exchanges.length - 1;
              return (
                <motion.div
                  key={i}
                  initial={{ opacity: 0, y: 10, filter: "blur(4px)" }}
                  animate={{ opacity: 1, y: 0, filter: "blur(0px)" }}
                  transition={{ duration: 0.35, ease: easeOut, delay: i === 0 ? 0 : 0.04 }}
                >
                  <QuestionCard
                    kind={ex.questionKind}
                    followupIndex={ex.followupIndex}
                    text={ex.question}
                  />
                  {ex.answer !== undefined && (
                    <AnswerBlock
                      text={ex.answer}
                      loading={isLast && awaitingNext}
                      loadingLabel={liveStatus(state)}
                    />
                  )}
                </motion.div>
              );
            })}

            {/* Answer box */}
            {awaiting && (
              <motion.div
                initial={{ opacity: 0, y: 6 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.3, ease: easeOut, delay: 0.05 }}
              >
                <AnswerInput
                  answer={answer}
                  setAnswer={setAnswer}
                  onSubmit={onSubmit}
                  onSkip={endQuestion}
                />
              </motion.div>
            )}

            {/* Verdict */}
            <AnimatePresence>
              {state.verdict && (
                <motion.div
                  key="verdict"
                  initial={{ opacity: 0, y: 10, scale: 0.97, filter: "blur(6px)" }}
                  animate={{ opacity: 1, y: 0, scale: 1, filter: "blur(0px)" }}
                  exit={{ opacity: 0, y: -4, filter: "blur(4px)" }}
                  transition={{ duration: 0.4, ease: easeOut }}
                >
                  <VerdictCard state={state} />
                </motion.div>
              )}
            </AnimatePresence>

            {state.status === "done" && (
              <div className="flex justify-end gap-2 pt-2">
                {state.sessionId && (
                  <a
                    href={`/replay/${state.sessionId}`}
                    className="inline-flex items-center justify-center gap-1.5 rounded-md border border-[var(--border)] px-3.5 h-9 text-sm hover:bg-[var(--secondary)] transition-colors active:scale-[0.97]"
                  >
                    <Sparkles className="size-3.5" /> Review + analyze
                  </a>
                )}
                <Button variant="outline" onClick={reset}>New drill</Button>
              </div>
            )}

            {state.status === "error" && (
              <div className="rounded-xl border border-[var(--destructive)]/30 bg-[var(--destructive)]/[0.04] p-4 text-sm text-[var(--destructive)]">
                {state.error}
              </div>
            )}
          </div>
        )}
      </div>
    </>
  );
}

// ─────────────────────────────────────────────────────────────────────────
// Subcomponents
// ─────────────────────────────────────────────────────────────────────────

function StartCard({
  qs,
  setQs,
  category,
  setCategory,
  onStart,
}: {
  qs: number;
  setQs: (n: number) => void;
  category: string;
  setCategory: (c: string) => void;
  onStart: () => void;
}) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 12, filter: "blur(6px)" }}
      animate={{ opacity: 1, y: 0, filter: "blur(0px)" }}
      transition={{ duration: 0.45, ease: easeOut }}
      className="space-y-8"
    >
      <div className="space-y-3">
        <p className="font-mono text-[10px] uppercase tracking-[0.2em] text-[var(--muted-foreground)]">
          Today's rehearsal
        </p>
        <h2 className="pull-quote text-3xl sm:text-4xl leading-[1.15]">
          <span className="pull-quote-mark text-5xl sm:text-6xl mr-1 align-top">“</span>
          Show up before they ask you to.
        </h2>
        <p className="text-sm text-[var(--muted-foreground)] max-w-xl leading-relaxed">
          Calibrate the next drill. The Planner reads your weakest topics, ELO, and the
          JDs you're chasing — then picks something you'd rather avoid.
        </p>
      </div>

      <div className="rounded-2xl border border-[var(--border)] bg-[var(--card)]/60 backdrop-blur-sm p-6 sm:p-7 space-y-7">
        <div>
          <label className="text-[10px] font-mono uppercase tracking-[0.18em] text-[var(--muted-foreground)] mb-3 block">
            Questions
          </label>
          <div className="flex gap-2 flex-wrap">
            {[1, 2, 3, 5, 7].map((n) => (
              <button
                key={n}
                onClick={() => setQs(n)}
                className={
                  "px-4 h-9 rounded-md font-mono text-xs border transition-[transform,background-color,border-color] duration-150 [transition-timing-function:var(--ease-out)] active:scale-[0.97] " +
                  (qs === n
                    ? "bg-[var(--primary)] text-[var(--primary-foreground)] border-transparent shadow-[0_1px_0_0_rgba(255,255,255,0.15)_inset]"
                    : "bg-[var(--card)] text-[var(--foreground)] border-[var(--border)] hover:bg-[var(--secondary)]")
                }
              >
                {n}Q
              </button>
            ))}
          </div>
        </div>

        <div>
          <label className="text-[10px] font-mono uppercase tracking-[0.18em] text-[var(--muted-foreground)] mb-3 block">
            Category
          </label>
          <div className="flex gap-2 flex-wrap">
            {CATEGORIES.map((c) => (
              <button
                key={c.id}
                onClick={() => setCategory(c.id)}
                className={
                  "px-3.5 h-9 rounded-md text-xs border transition-[transform,background-color,border-color] duration-150 [transition-timing-function:var(--ease-out)] active:scale-[0.97] " +
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

        <Button size="lg" onClick={onStart} className="w-full sm:w-auto">
          <Brain className="mr-1.5" /> Start drill
        </Button>
      </div>
    </motion.div>
  );
}

function SessionTimeline({ state }: { state: DrillState }) {
  const total = state.totalQuestions ?? 1;
  const cur = state.currentQuestionOrd ?? 1;
  return (
    <div className="flex items-center justify-between gap-4">
      <div className="flex items-center gap-2">
        {Array.from({ length: total }).map((_, i) => {
          const n = i + 1;
          const s = n < cur ? "done" : n === cur ? "active" : "pending";
          return <span key={i} className="timeline-node" data-state={s} aria-label={`Question ${n} (${s})`} />;
        })}
      </div>
      <p className="font-mono text-[10px] uppercase tracking-[0.18em] text-[var(--muted-foreground)]">
        {state.status === "done" ? "complete" : `Q ${cur}/${total}`}
      </p>
    </div>
  );
}

function PlanStrip({
  decision,
}: {
  decision: NonNullable<DrillState["decision"]>;
}) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 4 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.3, ease: easeOut }}
      className="flex items-baseline gap-3 flex-wrap pb-3 border-b border-[var(--border)]"
    >
      <Badge variant="primary">{decision.category}</Badge>
      <span className="font-mono text-xs text-[var(--muted-foreground)] truncate max-w-full">
        {decision.target_topic}
      </span>
      <span className="font-mono text-[10px] uppercase tracking-[0.15em] text-[var(--muted-foreground)] ml-auto">
        target ELO {decision.target_difficulty}
      </span>
    </motion.div>
  );
}

function StageCard({ state }: { state: DrillState }) {
  const stages = [
    { key: "planner:decision", label: "Planner" },
    { key: "interviewer:opening", label: "Interviewer" },
    { key: "judge:verdict", label: "Judge" },
  ];
  return (
    <div className="rounded-2xl border border-[var(--border)] bg-[var(--card)]/40 backdrop-blur-sm p-6">
      <p className="font-mono text-[10px] uppercase tracking-[0.18em] text-[var(--muted-foreground)] mb-3">
        Pipeline
      </p>
      <ul className="space-y-2.5">
        {stages.map((s) => {
          const done = state.events.some((e) => e.type === s.key);
          return (
            <li key={s.key} className="flex items-center gap-3 text-sm">
              <span
                className={
                  "grid place-items-center size-4 rounded-full border transition-colors duration-200 " +
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
    </div>
  );
}

function QuestionCard({
  kind,
  followupIndex,
  text,
  streaming,
}: {
  kind: "opening" | "followup";
  followupIndex?: number;
  text: string;
  streaming?: boolean;
}) {
  const isOpening = kind === "opening";
  return (
    <div className="rounded-2xl border border-[var(--border)] bg-[var(--card)]/70 backdrop-blur-sm p-6 sm:p-8 relative overflow-hidden">
      {/* Subtle violet hairline at the top */}
      <span
        aria-hidden
        className="absolute inset-x-0 top-0 h-px"
        style={{
          background:
            "linear-gradient(to right, transparent 0%, color-mix(in oklch, var(--primary) 50%, transparent) 50%, transparent 100%)",
        }}
      />
      <p className="font-mono text-[10px] uppercase tracking-[0.2em] text-[var(--primary)] mb-3 flex items-center gap-2">
        <span className="inline-block size-1.5 rounded-full bg-[var(--primary)]" />
        {isOpening ? "Interviewer" : `Follow-up ${followupIndex ?? ""}`}
      </p>
      {isOpening ? (
        <p className="pull-quote text-2xl sm:text-[1.65rem] leading-[1.4] text-[var(--foreground)]">
          <span className="pull-quote-mark text-4xl sm:text-5xl mr-1 align-top">“</span>
          {text}
          {streaming && <Caret />}
        </p>
      ) : (
        <p className="text-base sm:text-lg leading-relaxed text-[var(--foreground)]">
          {text}
          {streaming && <Caret />}
        </p>
      )}
    </div>
  );
}

function AnswerBlock({
  text,
  loading,
  loadingLabel,
}: {
  text: string;
  loading?: boolean;
  loadingLabel?: string;
}) {
  return (
    <div className="mt-3 ml-2 sm:ml-6 pl-4 sm:pl-5 border-l-2 border-[var(--border)] py-2">
      <p className="font-mono text-[10px] uppercase tracking-[0.18em] text-[var(--muted-foreground)] mb-2">
        You
      </p>
      <p className="text-sm sm:text-[15px] leading-relaxed whitespace-pre-wrap text-[var(--foreground)]/95">
        {text}
      </p>
      {loading && (
        <p className="mt-3 flex items-center gap-2 text-xs font-mono uppercase tracking-[0.12em] text-[var(--muted-foreground)]">
          <Loader2 className="size-3 animate-spin text-[var(--primary)]" />
          {loadingLabel}
        </p>
      )}
    </div>
  );
}

function AnswerInput({
  answer,
  setAnswer,
  onSubmit,
  onSkip,
}: {
  answer: string;
  setAnswer: (s: string) => void;
  onSubmit: () => void;
  onSkip: () => void;
}) {
  return (
    <div className="rounded-2xl border border-[var(--primary)]/25 bg-[var(--card)]/70 backdrop-blur-sm p-4 sm:p-5 space-y-3 shadow-[0_0_0_1px_color-mix(in_oklch,var(--primary)_15%,transparent)_inset]">
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
      <div className="flex items-center justify-between gap-2 flex-wrap">
        <div className="flex items-center gap-1.5 text-[10px] font-mono uppercase tracking-[0.12em] text-[var(--muted-foreground)]">
          <MicOff className="size-3" /> mic input coming
        </div>
        <div className="flex items-center gap-2">
          <Button variant="ghost" size="sm" onClick={onSkip}>
            Skip follow-ups
          </Button>
          <Button onClick={onSubmit} disabled={!answer.trim()}>
            <Send className="mr-1" /> Submit
          </Button>
        </div>
      </div>
    </div>
  );
}

function VerdictCard({ state }: { state: DrillState }) {
  const v = state.verdict!;
  const variant = v.rating >= 4 ? "success" : v.rating >= 3 ? "warning" : "danger";
  return (
    <div className="rounded-2xl border border-[var(--border)] bg-[var(--card)]/70 backdrop-blur-sm overflow-hidden">
      <div className="p-6 sm:p-7 flex items-start justify-between gap-4 border-b border-[var(--border)]">
        <div>
          <p className="font-mono text-[10px] uppercase tracking-[0.2em] text-[var(--primary)] mb-2 flex items-center gap-2">
            <Sparkles className="size-3" /> Judgment
          </p>
          <p className="pull-quote text-2xl sm:text-3xl leading-tight">
            {v.rating >= 4 ? "Strong." : v.rating >= 3 ? "Mixed." : "Off the mark."}
          </p>
        </div>
        <Badge variant={variant}>{v.rating}/5</Badge>
      </div>

      <div className="p-6 sm:p-7 space-y-5 text-sm">
        {v.strengths.length > 0 && (
          <Section title="Strengths" tone="success">
            <ul className="space-y-1.5">
              {v.strengths.map((s, i) => (
                <li key={i}>+ {s}</li>
              ))}
            </ul>
          </Section>
        )}
        {v.missed.length > 0 && (
          <Section title="Missed" tone="danger">
            <ul className="space-y-1.5">
              {v.missed.map((s, i) => (
                <li key={i}>− {s}</li>
              ))}
            </ul>
          </Section>
        )}
        {v.better_answer_sketch && (
          <Section title="A better answer" tone="muted">
            <p className="italic leading-relaxed font-[var(--font-display)] text-[var(--foreground)]">
              {v.better_answer_sketch}
            </p>
          </Section>
        )}
        {v.reading.length > 0 && (
          <Section title="Reading" tone="muted">
            <ul className="space-y-1">
              {v.reading.map((r, i) => (
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
          </Section>
        )}
        {v.topic_tags.length > 0 && (
          <div className="flex flex-wrap gap-1.5 pt-1">
            {v.topic_tags.map((t) => (
              <Badge key={t} variant="outline">
                {t}
              </Badge>
            ))}
          </div>
        )}
        {state.eloUpdate && (
          <div className="pt-3 border-t border-[var(--border)] flex items-center justify-between text-xs font-mono">
            <span className="text-[var(--muted-foreground)] uppercase tracking-[0.15em]">
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
      </div>
    </div>
  );
}

function Section({
  title,
  tone,
  children,
}: {
  title: string;
  tone: "success" | "danger" | "muted";
  children: React.ReactNode;
}) {
  const color =
    tone === "success"
      ? "text-[var(--success)]"
      : tone === "danger"
        ? "text-[var(--destructive)]"
        : "text-[var(--muted-foreground)]";
  return (
    <div>
      <p className={`font-mono text-[10px] uppercase tracking-[0.18em] mb-2 ${color}`}>{title}</p>
      {children}
    </div>
  );
}

function liveStatus(state: DrillState): string {
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
