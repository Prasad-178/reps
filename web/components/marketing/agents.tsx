"use client";

import { motion } from "framer-motion";
import { Compass, MessageSquare, Gavel, Sparkles } from "lucide-react";

const easeOut: [number, number, number, number] = [0.23, 1, 0.32, 1];

const agents = [
  {
    icon: Compass,
    name: "Planner",
    role: "Decides what to drill next",
    body: "Reads your per-category ELO, the weakest 10 topics, the last 20 drills, and your target JDs. Returns a single decision: category, topic, target difficulty, and a one-line rationale.",
    tag: "1 LLM call · ~$0.0002",
  },
  {
    icon: MessageSquare,
    name: "Interviewer",
    role: "Asks the question. Then probes.",
    body: "Generates a question grounded in your real shipped work — not generic. Decides per turn whether to ask a follow-up (only when your answer leaves a gap) or end. Hard cap: 3 follow-ups.",
    tag: "1 + ≤3 LLM calls",
  },
  {
    icon: Gavel,
    name: "Judge",
    role: "Grades 1–5. Honest, not generous.",
    body: "Loads the category rubric (system design, crypto, ML, Solana, JD-specific, general). Scores the full transcript. Returns strengths, what you missed, a better-answer sketch, and reading recs.",
    tag: "1 LLM call · structured JSON",
  },
  {
    icon: Sparkles,
    name: "Coach",
    role: "Weekly study plan",
    body: "Offline. Reads 30 days of topic hits. Clusters them into 4–8 themes. Orders by hits × (4 − mean rating) × JD relevance. Writes a Markdown plan you can dump to your prep folder.",
    tag: "1 LLM call · runs on /plan",
  },
];

export function Agents() {
  return (
    <section id="agents" className="relative py-24 sm:py-32">
      <div className="mx-auto max-w-6xl px-6">
        <div className="max-w-2xl">
          <p className="font-mono text-xs uppercase tracking-[0.12em] text-[var(--primary)] mb-3">
            Four agents
          </p>
          <h2 className="text-3xl sm:text-4xl font-semibold tracking-[-0.025em] leading-[1.05]">
            One drill. Four jobs. None of them are you.
          </h2>
          <p className="mt-4 text-[var(--muted-foreground)] leading-relaxed">
            Each agent has one prompt template and one job. They talk to OpenRouter,
            not to each other, with structured JSON between hops. Total cost for a
            3-question drill: about half a cent.
          </p>
        </div>

        <div className="mt-14 grid grid-cols-1 md:grid-cols-2 gap-4">
          {agents.map((a, i) => (
            <motion.div
              key={a.name}
              initial={{ opacity: 0, y: 12 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true, margin: "-80px" }}
              transition={{ duration: 0.45, ease: easeOut, delay: i * 0.04 }}
              className="group relative rounded-2xl border border-[var(--border)] bg-[var(--card)] p-6 overflow-hidden"
            >
              {/* hover halo */}
              <div
                aria-hidden
                className="absolute inset-0 opacity-0 group-hover:opacity-100 transition-opacity duration-300 pointer-events-none"
                style={{
                  background:
                    "radial-gradient(60% 60% at 20% 0%, color-mix(in oklch, var(--primary) 12%, transparent) 0%, transparent 70%)",
                }}
              />
              <div className="relative flex items-start justify-between gap-4">
                <div className="flex items-center gap-3">
                  <div className="grid place-items-center size-10 rounded-lg bg-[color-mix(in_oklch,var(--primary)_14%,transparent)] text-[var(--primary)]">
                    <a.icon className="size-5" />
                  </div>
                  <div>
                    <h3 className="text-lg font-semibold tracking-[-0.01em]">{a.name}</h3>
                    <p className="text-xs text-[var(--muted-foreground)]">{a.role}</p>
                  </div>
                </div>
                <span className="font-mono text-[10px] uppercase tracking-[0.08em] text-[var(--muted-foreground)] hidden sm:inline">
                  {a.tag}
                </span>
              </div>
              <p className="relative mt-4 text-sm text-[var(--muted-foreground)] leading-relaxed">
                {a.body}
              </p>
            </motion.div>
          ))}
        </div>
      </div>
    </section>
  );
}
