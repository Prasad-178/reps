"use client";

import { useRef } from "react";
import { motion, useMotionValue, useSpring, useTransform } from "framer-motion";
import { Button } from "@/components/ui/button";
import { ArrowRight, Terminal } from "lucide-react";
import Link from "next/link";

const easeOut: [number, number, number, number] = [0.23, 1, 0.32, 1];

export function Hero() {
  const sectionRef = useRef<HTMLElement | null>(null);
  const mx = useMotionValue(0.5);
  const my = useMotionValue(0.4);
  const springX = useSpring(mx, { stiffness: 80, damping: 18 });
  const springY = useSpring(my, { stiffness: 80, damping: 18 });
  const orbX = useTransform(springX, (v) => `${v * 100}%`);
  const orbY = useTransform(springY, (v) => `${v * 100}%`);

  function onMove(e: React.MouseEvent<HTMLElement>) {
    const r = sectionRef.current?.getBoundingClientRect();
    if (!r) return;
    mx.set((e.clientX - r.left) / r.width);
    my.set((e.clientY - r.top) / r.height);
  }

  return (
    <section
      ref={sectionRef}
      onMouseMove={onMove}
      className="relative overflow-hidden"
    >
      {/* halo gradient */}
      <div
        aria-hidden
        className="absolute inset-0 pointer-events-none"
        style={{ background: "var(--gradient-bg)" }}
      />
      {/* mouse-tracking orb (decorative; springs for natural feel) */}
      <motion.div
        aria-hidden
        className="absolute pointer-events-none size-[520px] -translate-x-1/2 -translate-y-1/2 blur-3xl opacity-50 hidden md:block will-change-transform"
        style={{
          left: orbX,
          top: orbY,
          background:
            "radial-gradient(closest-side, color-mix(in oklch, var(--primary) 35%, transparent) 0%, transparent 70%)",
          transform: "translateZ(0)",
        }}
      />
      {/* grid */}
      <div aria-hidden className="absolute inset-0 bg-grid bg-grid-fade pointer-events-none" />

      <div className="relative mx-auto max-w-5xl px-6 pt-20 pb-24 sm:pt-28 sm:pb-32 text-center">
        <motion.div
          initial={{ opacity: 0, y: 8 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5, ease: easeOut }}
          className="inline-flex items-center gap-2 rounded-full border border-[var(--border)] bg-[var(--card)] px-3 py-1 text-xs font-mono text-[var(--muted-foreground)]"
        >
          <span className="size-1.5 rounded-full bg-[var(--success)]" />
          single Go binary · BYO OpenRouter key
        </motion.div>

        <motion.h1
          initial={{ opacity: 0, y: 16 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.55, ease: easeOut, delay: 0.05 }}
          className="mt-6 text-4xl sm:text-6xl font-semibold tracking-[-0.035em] leading-[1.02]"
        >
          Interview rehearsal that{" "}
          <span
            className="bg-clip-text text-transparent"
            style={{ backgroundImage: "var(--gradient-accent)" }}
          >
            reads everything you&apos;ve shipped
          </span>
          .
        </motion.h1>

        <motion.p
          initial={{ opacity: 0, y: 16 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.55, ease: easeOut, delay: 0.1 }}
          className="mt-6 mx-auto max-w-2xl text-base sm:text-lg text-[var(--muted-foreground)] leading-relaxed"
        >
          Four agents — Planner, Interviewer, Judge, Coach — drill you daily on
          theory questions tied to your real projects and the JDs you&apos;re
          targeting. ELO-tracked. Honest, not generous. No leetcode.
        </motion.p>

        <motion.div
          initial={{ opacity: 0, y: 16 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.55, ease: easeOut, delay: 0.15 }}
          className="mt-9 flex flex-col sm:flex-row items-center justify-center gap-3"
        >
          <Button size="lg" asChild>
            <Link href="/dashboard">
              Start a drill <ArrowRight className="ml-1" />
            </Link>
          </Button>
          <Button size="lg" variant="outline" asChild>
            <a href="#install">
              <Terminal className="mr-1" /> brew install reps
            </a>
          </Button>
        </motion.div>

        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.6, delay: 0.4 }}
          className="mt-10 flex items-center justify-center gap-x-8 gap-y-3 flex-wrap text-xs font-mono uppercase tracking-[0.1em] text-[var(--muted-foreground)]"
        >
          <span>$2 / yr at 1 drill / day</span>
          <span className="hidden sm:inline opacity-30">·</span>
          <span>per-category ELO</span>
          <span className="hidden sm:inline opacity-30">·</span>
          <span>voice in &amp; out</span>
          <span className="hidden sm:inline opacity-30">·</span>
          <span>theory only</span>
        </motion.div>
      </div>
    </section>
  );
}
