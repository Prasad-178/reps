"use client";

import { useState } from "react";
import { Check, Copy } from "lucide-react";
import { Button } from "@/components/ui/button";

const blocks = [
  { label: "Homebrew", cmd: "brew install repsai/reps" },
  { label: "go install", cmd: "go install github.com/Prasad-178/reps/cmd/reps@latest" },
  { label: "From source", cmd: "git clone https://github.com/Prasad-178/reps && cd reps && go build ./cmd/reps" },
];

export function Install() {
  const [copied, setCopied] = useState<string | null>(null);
  function copy(s: string) {
    navigator.clipboard.writeText(s);
    setCopied(s);
    setTimeout(() => setCopied((c) => (c === s ? null : c)), 1400);
  }

  return (
    <section id="install" className="relative py-24 sm:py-32 border-t border-[var(--border)]">
      <div className="mx-auto max-w-4xl px-6">
        <div className="max-w-2xl mx-auto text-center">
          <p className="font-mono text-xs uppercase tracking-[0.12em] text-[var(--primary)] mb-3">
            Install
          </p>
          <h2 className="text-3xl sm:text-4xl font-semibold tracking-[-0.025em] leading-[1.05]">
            One binary. Local data. BYO key.
          </h2>
          <p className="mt-4 text-[var(--muted-foreground)] leading-relaxed">
            macOS (arm64 + amd64) and Linux. Needs <code className="font-mono text-xs px-1.5 py-0.5 rounded bg-[var(--muted)] text-[var(--foreground)]">poppler</code> for PDF and{" "}
            <code className="font-mono text-xs px-1.5 py-0.5 rounded bg-[var(--muted)] text-[var(--foreground)]">gh</code> for GitHub ingestion. Optional <code className="font-mono text-xs px-1.5 py-0.5 rounded bg-[var(--muted)] text-[var(--foreground)]">sox</code> + <code className="font-mono text-xs px-1.5 py-0.5 rounded bg-[var(--muted)] text-[var(--foreground)]">whisper-cli</code> for voice.
          </p>
        </div>

        <div className="mt-10 space-y-3">
          {blocks.map((b) => (
            <div
              key={b.label}
              className="group flex items-center justify-between gap-4 rounded-xl border border-[var(--border)] bg-[var(--card)] px-4 py-3"
            >
              <div className="flex items-center gap-4 min-w-0 flex-1">
                <span className="font-mono text-[10px] uppercase tracking-[0.1em] text-[var(--muted-foreground)] w-20 shrink-0">
                  {b.label}
                </span>
                <code className="font-mono text-sm text-[var(--foreground)] truncate">
                  {b.cmd}
                </code>
              </div>
              <Button
                variant="ghost"
                size="icon"
                onClick={() => copy(b.cmd)}
                aria-label={`Copy ${b.label}`}
              >
                {copied === b.cmd ? (
                  <Check className="text-[var(--success)]" />
                ) : (
                  <Copy />
                )}
              </Button>
            </div>
          ))}
        </div>

        <div className="mt-8 text-center text-sm text-[var(--muted-foreground)]">
          Then <code className="font-mono text-xs px-1.5 py-0.5 rounded bg-[var(--muted)] text-[var(--foreground)]">reps init</code> and you&apos;re three commands from your first drill.
        </div>
      </div>
    </section>
  );
}
