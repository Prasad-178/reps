# reps

Personalized, agentic interview rehearsal CLI. Theory-only — system design, domain (crypto/ML/Solana), JD-specific, general. No leetcode.

`reps` reads your real shipped work (resume, portfolio, GitHub, JDs, notes), ingests it once, and then runs daily ~15-minute drills with four agents:

- **Planner** picks what to ask given your weakest topics, ELO, and target JDs.
- **Interviewer** asks the opening question and decides whether to probe with up to 3 follow-ups.
- **Judge** grades 1–5 against a category rubric and emits strengths/missed/better-answer-sketch/reading recs.
- **Coach** (offline) synthesizes a clustered study plan from your accumulated weak topics.

Voice answers via `whisper.cpp`. Text input too. Per-category ELO. Persistent SQLite store.

## Install

Requires Go 1.23+, `pdftotext` (poppler), `gh` (GitHub CLI). Optional: `sox` + `whisper-cli` for voice.

```bash
brew install poppler gh
gh auth login

git clone https://github.com/Prasad-178/reps && cd reps
go build -o /usr/local/bin/reps ./cmd/reps
```

## Quick start

```bash
export OPENROUTER_API_KEY=sk-or-...

reps init
reps add resume ~/path/to/resume.pdf
reps add github your-username
reps add portfolio https://you.dev
reps add jd https://jobs.example.com/staff-ml-eng
reps profile --rebuild

reps drill --qs 3
reps stats
reps plan
```

## CLI

```
reps init                            interactive personalization wizard
reps add resume <path>               ingest resume PDF (uses pdftotext)
reps add portfolio <url>             scrape portfolio (chromedp fallback)
reps add github <user>               list repos + READMEs via gh CLI
reps add linkedin <ref> [--from-file p]
reps add x <handle> [--from-file p]
reps add jd <url>                    scrape JD + extract structured card
reps add note <path>                 ingest a markdown note

reps profile [--rebuild]             show or rebuild synthesized profile

reps drill                           default: 3 Qs, text input
  --voice                            mic input via whisper.cpp
  --category <cat>                   force a category
  --topic <str>                      force a topic
  --jd <id>                          focus on one JD
  --qs N                             1..10
  --difficulty <elo>                 override target ELO

reps stats                           per-category ELO + 7-day trend + weakest topics
reps history [--last N]              recent sessions
reps replay <id>                     re-print a session
reps plan [--days 30]                generate Markdown study plan
reps export [--md|--json]            dump corpus + drills

reps config <key> [value]            get or set a config key
reps reset --yes [--all|--data|--sources]
```

## Config

`~/.reps/config.toml`. Override via env: `OPENROUTER_API_KEY`, `REPS_MODEL`, `REPS_EMBED_MODEL`, `REPS_JUDGE_MODEL`, `REPS_HOME`.

```toml
[llm]
provider     = "openrouter"
model        = "google/gemini-2.0-flash-001"
embed_model  = "openai/text-embedding-3-small"
judge_model  = "anthropic/claude-3.5-haiku"

[voice]
enabled       = true
whisper_bin   = "/opt/homebrew/bin/whisper-cli"
whisper_model = "~/.reps/models/ggml-base.en.bin"
recorder      = "sox"

[drill]
default_qs   = 3
followup_max = 3
time_warn_sec = 240

[elo]
k_factor     = 24
start_rating = 1200
```

## Cost

Defaults to Gemini 2.0 Flash for all four agents. One 3-question drill ≈ $0.005. Daily for a year ≈ $2.

## Voice setup

```bash
./scripts/install-whisper.sh        # installs whisper-cpp + sox via brew, downloads base.en
```

## Web UI

A Next.js frontend lives in `web/`. The Go backend exposes an HTTP API via
`reps serve` (default `:7777`). They run independently — no embedded SPA, no
bundled binary inflation.

**Easiest:** drop a `.env` at the repo root and use `make dev`:

```bash
cp .env.example .env       # then edit and set OPENROUTER_API_KEY
make dev                   # backend :7777 + frontend :3000, one terminal
```

The Go binary auto-loads `.env` from the current directory, `$REPS_HOME/.env`, or
`$REPS_ENV_FILE`. Real shell `export`s always win, so the loader only fills gaps.

If you prefer separate terminals:

```bash
# terminal 1 — backend
reps serve                 # reads ./.env automatically

# terminal 2 — frontend (dev)
cd web && bun install && bun dev
# open http://localhost:3000
```

Pages:
- `/` landing (marketing)
- `/dashboard` ELO chart, KPIs, weakest topics, recent sessions
- `/drill` live drill via SSE — Planner → Interviewer → Judge → ELO
- `/sources` ingested resume / GitHub / portfolio / JDs / notes
- `/jds` parsed JD cards
- `/plan` latest study plan (Markdown render)
- `/history` session list
- `/replay/[id]` full transcript + judgment per question
- `/profile` synthesized profile

Brand: `Obsidian Spark` — electric violet on near-black. See [brand.md](../brand.md).

## How it works

```
                                ┌─────────────┐
   ~/.reps/sources/  ──ingest─▶ │   sqlite    │
   resume / GH / JD              │ + sqlite-vec│
                                └──────┬──────┘
                                       ▼
                                ┌─────────────┐
                                │   Planner   │  picks (cat, topic, difficulty)
                                └──────┬──────┘
                                       ▼
                                 RAG retrieve + rerank (top 3 chunks)
                                       ▼
                                ┌─────────────┐
                                │ Interviewer │  opening → answer → maybe follow-up (≤3)
                                └──────┬──────┘
                                       ▼
                                ┌─────────────┐
                                │    Judge    │  rubric → rating, tags, reading
                                └──────┬──────┘
                                       ▼
                                  ELO update + topic hits
                                       ▼
                                ┌─────────────┐
                                │    Coach    │  weekly study plan
                                └─────────────┘
```

## License

MIT. BYO OpenRouter key. Local-only data.
