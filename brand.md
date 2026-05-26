# reps — Brand

> **reps** — personalized, agentic interview rehearsal CLI. Drills you on your real shipped work against target JDs.
>
> Category: general tech / CS (interview prep tooling — neither AI-specific nor pure dev-tooling).
> Mood: **premium** + **bold** (with high data density). Dashboard-heavy.

---

## Palette — Phosphor Ember

Burnished amber on graphite. Terminal-CRT warmth with editorial restraint.
Replaces the prior "Obsidian Spark" violet — see commit history for the
transition rationale.

### Seeds (OKLCH)

| Token | Light | Dark |
| --- | --- | --- |
| `--background` | `oklch(0.99 0.004 70)` | `oklch(0.135 0.006 60)` |
| `--foreground` | `oklch(0.18 0.012 60)` | `oklch(0.97 0.006 70)` |
| `--card` | `oklch(0.985 0.006 70)` | `oklch(0.175 0.008 60)` |
| `--primary` | `oklch(0.62 0.16 70)` | `oklch(0.78 0.16 75)` |
| `--primary-foreground` | `oklch(0.13 0.012 60)` | `oklch(0.13 0.012 60)` |
| `--muted` | `oklch(0.95 0.008 70)` | `oklch(0.2 0.01 60)` |
| `--muted-foreground` | `oklch(0.48 0.015 60)` | `oklch(0.68 0.012 70)` |
| `--accent` | `oklch(0.93 0.05 70)` | `oklch(0.27 0.06 70)` |
| `--destructive` | `oklch(0.58 0.22 25)` | `oklch(0.7 0.2 25)` |
| `--success` | `oklch(0.55 0.16 145)` | `oklch(0.78 0.18 145)` |
| `--warning` | `oklch(0.7 0.16 75)` | `oklch(0.85 0.16 80)` |
| `--border` | `oklch(0.92 0.008 70)` | `oklch(0.26 0.014 60)` |
| `--ring` | `oklch(0.62 0.16 70 / 0.5)` | `oklch(0.78 0.16 75 / 0.55)` |
| `--radius` | `0.75rem` | `0.75rem` |

All foreground/background pairs verified ≥ WCAG AA (4.5:1 body text, 3:1 large/icons).

### Gradients

- `--gradient-bg` — radial halo at top, primary-tinted, fades to transparent. Use behind hero sections.
- `--gradient-accent` — 135° linear, primary → primary-soft-shifted. Use for CTA buttons, brand pills, sparkline strokes.

## Typography

- **Display (editorial accent):** Instrument Serif (italic) — `--font-display`.
  Used for: the `reps.` wordmark, interviewer pull-quotes, verdict headlines,
  empty-state hero copy. Distinctive italic serif — characterful but legible.
- **Sans (body):** Inter (variable, latin) — `--font-sans`. Default for
  everything except code/data/display moments.
- **Mono (data):** JetBrains Mono — `--font-mono`. Used for numbers (ELO,
  deltas), category tags, technical labels, code blocks, IDs, eyebrows.

Wired via `next/font/google` in `web/app/layout.tsx`. Font features
`cv11 ss01 ss03` enabled in `body`.

## Logo

A serif italic wordmark: `reps` (foreground) + amber `.` (primary). No icon,
no bounded box. The period is the brand mark. Hovering the logo nudges the
period one pixel to the right — small life-signal on an otherwise still mark.

## Motion / interaction (Emil Kowalski rules embedded)

- All UI animations < 300ms.
- `--ease-out: cubic-bezier(0.23, 1, 0.32, 1)` for enters.
- `--ease-in-out: cubic-bezier(0.77, 0, 0.175, 1)` for on-screen morphs.
- `--ease-drawer: cubic-bezier(0.32, 0.72, 0, 1)` for sheets/drawers.
- Buttons scale to `0.97` on `:active` for press feedback.
- Never animate from `scale(0)`; start `0.95` + opacity.
- Tooltips: skip delay + skip animation after the first opens.
- Hover effects only inside `@media (hover: hover) and (pointer: fine)`.

## Tone / voice

- **Terse.** Senior eng register. No marketing fluff, no exclamation points.
- **Specific.** Quote real numbers (ELO deltas, ratings, hit counts) — never adjectives.
- **Honest, not generous.** The Judge agent's voice is harsh-but-fair; the UI mirrors that. No "Great job!" toasts.
- Headlines: short and declarative ("Drill harder.", "Your weakest topic this week."). No questions in headlines.
- Buttons: imperative verbs ("Start drill", "Rebuild profile", "Export"). Never "Click here".

## Dos and don'ts

- ✅ Lead with data. Numbers first, labels under them in `--muted-foreground`.
- ✅ Mono font for every number, tag, and ID.
- ✅ Dark by default. Light mode exists but is the secondary surface.
- ✅ One primary CTA per screen.
- ❌ Don't use the violet primary for body text. Only CTAs, active nav, focus rings, accent strokes.
- ❌ Don't use rounded-full on cards. Use `rounded-xl` / `rounded-md`.
- ❌ No drop shadows except `0 1px 0` inset highlights on primary buttons.
- ❌ Never animate keyboard-triggered actions (cmd palette, /drill shortcut).

## Files

- Palette tokens: `web/app/globals.css`
- Backup of pre-brand defaults: `web/app/globals.css.bak`
- Layout / font wiring: `web/app/layout.tsx`
- Preview (regenerable; gitignored): `.brand-preview/index.html`
