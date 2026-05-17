"use client";

import { motion } from "framer-motion";

const easeOut: [number, number, number, number] = [0.23, 1, 0.32, 1];

const steps = [
  {
    n: "01",
    title: "Ingest your work",
    body: "Resume PDF, GitHub repos, portfolio scrape, JD URLs, LinkedIn paste, raw notes. Chunked, embedded, stored locally in SQLite + sqlite-vec.",
    code: `$ reps add resume ~/resume.pdf
$ reps add github prasad-178
$ reps add jd https://jobs.example.com/staff-ml-eng`,
  },
  {
    n: "02",
    title: "Synthesize your profile",
    body: "One LLM pass over everything → a ~1.5k-token profile that gets stuffed into every agent prompt. You can hand-edit it.",
    code: `$ reps profile --rebuild
✓ profile written.`,
  },
  {
    n: "03",
    title: "Drill",
    body: "Planner picks. RAG retrieves grounding chunks. Interviewer asks. You answer (text or mic). Up to 3 follow-ups if your answer leaves a gap.",
    code: `$ reps drill --qs 3
Plan: domain-crypto | topic="multi-tenant FHE"
Interviewer drafting question...`,
  },
  {
    n: "04",
    title: "Judge + ELO + plan",
    body: "Judge grades against the category rubric. ELO updates. Topic tags accumulate. Coach turns 30 days of weak topics into a clustered study plan.",
    code: `Rating: 3/5
ELO: domain-crypto 1200 → 1208 (+8)
$ reps plan --days 30`,
  },
];

export function How() {
  return (
    <section id="how" className="relative py-24 sm:py-32 border-t border-[var(--border)]">
      <div className="mx-auto max-w-6xl px-6">
        <div className="max-w-2xl">
          <p className="font-mono text-xs uppercase tracking-[0.12em] text-[var(--primary)] mb-3">
            How it works
          </p>
          <h2 className="text-3xl sm:text-4xl font-semibold tracking-[-0.025em] leading-[1.05]">
            Four steps. The compound is the point.
          </h2>
        </div>

        <div className="mt-14 grid gap-px bg-[var(--border)] rounded-2xl overflow-hidden border border-[var(--border)]">
          {steps.map((s, i) => (
            <motion.div
              key={s.n}
              initial={{ opacity: 0, y: 10 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true, margin: "-60px" }}
              transition={{ duration: 0.4, ease: easeOut, delay: i * 0.03 }}
              className="bg-[var(--card)] p-6 sm:p-8 grid grid-cols-1 sm:grid-cols-[120px_1fr_minmax(0,420px)] gap-6 sm:gap-10 items-start"
            >
              <span className="font-mono text-3xl sm:text-4xl text-[color-mix(in_oklch,var(--foreground)_25%,transparent)]">
                {s.n}
              </span>
              <div>
                <h3 className="text-xl font-semibold tracking-[-0.015em]">{s.title}</h3>
                <p className="mt-2 text-[var(--muted-foreground)] leading-relaxed text-sm">
                  {s.body}
                </p>
              </div>
              <pre className="font-mono text-xs leading-relaxed text-[var(--muted-foreground)] bg-[var(--background)] border border-[var(--border)] rounded-lg p-4 overflow-x-auto">
                <code>{s.code}</code>
              </pre>
            </motion.div>
          ))}
        </div>
      </div>
    </section>
  );
}
