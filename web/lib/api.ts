// Thin client for the reps Go backend.
// Backend default: http://localhost:7777. Override with NEXT_PUBLIC_REPS_API.

const BASE =
  (typeof process !== "undefined" && process.env.NEXT_PUBLIC_REPS_API) ||
  "http://localhost:7777";

async function call<T>(path: string, init?: RequestInit): Promise<T> {
  const r = await fetch(`${BASE}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers || {}),
    },
    cache: "no-store",
  });
  if (!r.ok) {
    const text = await r.text();
    throw new Error(`${r.status} ${r.statusText}: ${text}`);
  }
  return (await r.json()) as T;
}

// ---- types

export type StatsResponse = {
  overall: number;
  by_category: Record<string, { rating: number; delta_7d: number }>;
  weakest: WeakTopic[];
};

export type WeakTopic = {
  tag: string;
  hits: number;
  mean_rating: number;
  categories?: string[];
};

export type Source = {
  id: string;
  kind: string;
  ref: string;
  fetched_at: number;
};

export type JDCard = {
  id: string;
  company: string;
  role: string;
  card: {
    company: string;
    role: string;
    location?: string;
    level?: string;
    must_haves: string[];
    nice_to_haves: string[];
    culture: string[];
    tech: string[];
    summary: string;
  };
};

export type Profile = {
  markdown: string;
  built_at: number;
  model_used: string;
};

export type SessionSummary = {
  id: string;
  started_at: number;
  ended_at: number | null;
  mode: string;
  q_count: number;
  mean_rating: number;
};

export type EloPoint = { at: number; category: string; rating: number };

export type Turn = {
  ord: number;
  speaker: "interviewer" | "candidate";
  kind: string;
  text: string;
};

export type Judgment = {
  rating: number;
  strengths: string[];
  missed: string[];
  better_sketch: string;
  reading: { topic: string; why: string; optional_url?: string }[];
  topic_tags: string[];
};

export type Question = {
  ord: number;
  category: string;
  topic: string;
  target_elo: number;
  rationale: string;
  turns: Turn[];
  judgment?: Judgment;
};

export type ReplayResponse = {
  session: SessionSummary;
  questions: Question[];
};

export type Plan = {
  id: string;
  generated_at: number;
  window_days: number;
  markdown: string;
};

// ---- endpoints

export const api = {
  stats:    () => call<StatsResponse>("/api/stats"),
  sources:  () => call<Source[]>("/api/sources"),
  jds:      () => call<JDCard[]>("/api/jds"),
  profile:  () => call<Profile>("/api/profile"),
  sessions: () => call<SessionSummary[]>("/api/sessions"),
  replay:   (id: string) => call<ReplayResponse>(`/api/sessions/${id}`),
  elo:      (days = 30) => call<EloPoint[]>(`/api/elo?days=${days}`),
  plans:    () => call<Plan[]>("/api/plans"),
  latestPlan: () => call<Plan | null>("/api/plans/latest"),

  // Drill streaming endpoint — uses SSE
  drillURL: (opts: {
    qs?: number;
    category?: string;
    topic?: string;
    jd?: string;
    difficulty?: number;
  } = {}) => {
    const p = new URLSearchParams();
    if (opts.qs) p.set("qs", String(opts.qs));
    if (opts.category) p.set("category", opts.category);
    if (opts.topic) p.set("topic", opts.topic);
    if (opts.jd) p.set("jd", opts.jd);
    if (opts.difficulty) p.set("difficulty", String(opts.difficulty));
    return `${BASE}/api/drill/stream?${p.toString()}`;
  },

  submitAnswer: (sessionID: string, questionID: string, text: string) =>
    call<{ ok: true }>(
      `/api/drill/${sessionID}/${questionID}/answer`,
      { method: "POST", body: JSON.stringify({ text }) }
    ),

  endQuestion: (sessionID: string, questionID: string) =>
    call<{ ok: true }>(
      `/api/drill/${sessionID}/${questionID}/end`,
      { method: "POST" }
    ),
};
