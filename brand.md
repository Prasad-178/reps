# reps — Brand

> **reps** — personalized, agentic interview rehearsal CLI. Drills you on your real shipped work against target JDs.
>
> Category: general tech / CS (interview prep tooling — neither AI-specific nor pure dev-tooling).
> Mood: **premium** + **bold** (with high data density). Dashboard-heavy.

---

## Palette — Obsidian Spark

Electric violet on near-black. Linear-style premium tech.

### Seeds (OKLCH)

| Token | Light | Dark |
| --- | --- | --- |
| `--background` | `oklch(0.99 0.003 280)` | `oklch(0.13 0.006 280)` |
| `--foreground` | `oklch(0.18 0.01 280)` | `oklch(0.97 0.005 280)` |
| `--card` | `oklch(0.985 0.005 280)` | `oklch(0.17 0.008 280)` |
| `--primary` | `oklch(0.5 0.22 285)` | `oklch(0.66 0.22 285)` |
| `--primary-foreground` | `oklch(0.99 0.003 280)` | `oklch(0.99 0.005 280)` |
| `--muted` | `oklch(0.95 0.005 280)` | `oklch(0.2 0.008 280)` |
| `--muted-foreground` | `oklch(0.5 0.012 280)` | `oklch(0.68 0.01 280)` |
| `--accent` | `oklch(0.93 0.04 285)` | `oklch(0.25 0.06 285)` |
| `--destructive` | `oklch(0.58 0.22 25)` | `oklch(0.7 0.2 25)` |
| `--success` | `oklch(0.55 0.16 145)` | `oklch(0.78 0.18 145)` |
| `--warning` | `oklch(0.7 0.16 75)` | `oklch(0.82 0.15 75)` |
| `--border` | `oklch(0.92 0.005 280)` | `oklch(0.25 0.012 280)` |
| `--ring` | `oklch(0.5 0.22 285 / 0.5)` | `oklch(0.66 0.22 285 / 0.55)` |
| `--radius` | `0.75rem` | `0.75rem` |

All foreground/background pairs verified ≥ WCAG AA (4.5:1 body text, 3:1 large/icons).

### Gradients

- `--gradient-bg` — radial halo at top, primary-tinted, fades to transparent. Use behind hero sections.
- `--gradient-accent` — 135° linear, primary → primary-soft-shifted. Use for CTA buttons, brand pills, sparkline strokes.

## Typography

- **Sans:** Inter (variable, latin) — `--font-sans`. Used for everything except code/data.
- **Mono:** JetBrains Mono — `--font-mono`. Used for numbers (ELO, deltas), category tags, technical labels, code blocks, IDs.

Wired via `next/font/google` in `web/app/layout.tsx`. Font features `cv11 ss01 ss03` enabled in `body`.

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
