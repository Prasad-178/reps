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

export type SessionCritique = {
  headline: string;
  verdict: "good" | "mixed" | "bad" | string;
  overall_rating: number;
  patterns: { name: string; evidence: string; fix: string }[];
  strengths: string[];
  growth_edge: { action: string; why: string }[];
  drill_again: string[];
  reading: { topic: string; why: string; url?: string }[];
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
  analyzeSession: (id: string) =>
    call<SessionCritique>(`/api/sessions/${id}/analyze`, { method: "POST" }),
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

  // ---- ingestion (added in HI-1)
  addGithub:    (user: string) =>
    call<{ id: string }>(`/api/sources/github`,    { method: "POST", body: JSON.stringify({ user }) }),
  addPortfolio: (ref: string) => {
    const isURL = /^https?:\/\//i.test(ref);
    return call<{ id: string }>(`/api/sources/portfolio`, {
      method: "POST",
      body: JSON.stringify(isURL ? { url: ref } : { path: ref }),
    });
  },
  addJD: (url: string) =>
    call<{ id: string }>(`/api/sources/jd`,        { method: "POST", body: JSON.stringify({ url }) }),
  addLinkedIn: (ref: string, text: string) =>
    call<{ id: string }>(`/api/sources/linkedin`,  { method: "POST", body: JSON.stringify({ ref, text }) }),
  addX: (ref: string, text: string) =>
    call<{ id: string }>(`/api/sources/x`,         { method: "POST", body: JSON.stringify({ ref, text }) }),
  addNote: (name: string, content: string) =>
    call<{ id: string }>(`/api/sources/note`,      { method: "POST", body: JSON.stringify({ name, content }) }),
  addResume: async (file: File) => {
    const fd = new FormData();
    fd.append("file", file);
    const r = await fetch(`${BASE}/api/sources/resume`, { method: "POST", body: fd });
    if (!r.ok) throw new Error(`${r.status}: ${await r.text()}`);
    return (await r.json()) as { id: string };
  },
  deleteSource: (id: string) =>
    call<{ ok: true }>(`/api/sources/${id}`, { method: "DELETE" }),

  rebuildProfile: () =>
    call<RebuildStatus>(`/api/profile/rebuild`, { method: "POST" }),
  rebuildStatus: () =>
    call<RebuildStatus>(`/api/profile/rebuild/status`),

  insights: (force = false) =>
    call<InsightsResponse>(`/api/insights${force ? "?force=1" : ""}`),

  getConfig:   () => call<ConfigPublic>("/api/config"),
  patchConfig: (patch: ConfigPatch) =>
    call<ConfigPublic>("/api/config", { method: "PUT", body: JSON.stringify(patch) }),
  probeModel:  (model: string) =>
    call<{ ok: boolean; error?: string }>(
      "/api/config/probe-model",
      { method: "POST", body: JSON.stringify({ model }) }
    ),
};

export type ConfigPublic = {
  llm: {
    provider: string;
    model: string;
    embed_model: string;
    judge_model: string;
    rerank_model: string;
    api_key_mask: string;
  };
  drill: { default_qs: number; followup_max: number; time_warn_sec: number };
  elo: { k_factor: number; start_rating: number };
  voice: {
    tts_enabled: boolean;
    tts_provider: string;
    tts_voice: string;
    tts_model: string;
    tts_rate: number;
  };
  paths: { home: string };
};

export type ConfigPatch = {
  llm?: Partial<{
    model: string;
    embed_model: string;
    judge_model: string;
    rerank_model: string;
    api_key: string;
  }>;
  drill?: Partial<{
    default_qs: number;
    followup_max: number;
    time_warn_sec: number;
  }>;
  elo?: Partial<{ k_factor: number; start_rating: number }>;
  voice?: Partial<{
    tts_enabled: boolean;
    tts_provider: string;
    tts_voice: string;
    tts_model: string;
    tts_rate: number;
  }>;
};

export type RebuildStatus = {
  running: boolean;
  started_at: number;
  finished_at: number;
  error?: string;
  last_line?: string;
};

export type InsightsResponse = {
  summary: string;
  panels: InsightPanel[];
  built_at: number;
  cached: boolean;
  error?: string;
};

export type InsightPanel = {
  id: string;
  title: string;
  kind: "headline" | "callout" | "stat-row" | "sparkline" | "tag-cloud" | "list";
  severity: "good" | "warn" | "bad" | "info";
  headline: string;
  body: string;
  stats?: { label: string; value: string; delta?: number; unit?: string }[];
  tags?: string[];
  items?: string[];
  suggestion?: string;
};
