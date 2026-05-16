-- reps schema. {{EMBED_DIM}} is substituted at init time from config.

CREATE TABLE IF NOT EXISTS schema_meta (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS sources (
  id          TEXT PRIMARY KEY,
  kind        TEXT NOT NULL,
  ref         TEXT NOT NULL,
  raw_path    TEXT NOT NULL,
  fetched_at  INTEGER NOT NULL,
  meta_json   TEXT
);

CREATE TABLE IF NOT EXISTS chunks (
  id          TEXT PRIMARY KEY,
  source_id   TEXT NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
  ord         INTEGER NOT NULL,
  text        TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_chunks_source ON chunks(source_id);

CREATE VIRTUAL TABLE IF NOT EXISTS chunks_vec USING vec0(
  chunk_id TEXT PRIMARY KEY,
  embedding FLOAT[{{EMBED_DIM}}]
);

CREATE TABLE IF NOT EXISTS profile (
  id          INTEGER PRIMARY KEY CHECK (id = 1),
  markdown    TEXT NOT NULL,
  built_at    INTEGER NOT NULL,
  model_used  TEXT
);

CREATE TABLE IF NOT EXISTS jd_cards (
  id         TEXT PRIMARY KEY,
  source_id  TEXT NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
  company    TEXT,
  role       TEXT,
  card_json  TEXT NOT NULL,
  priority   INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS sessions (
  id          TEXT PRIMARY KEY,
  started_at  INTEGER NOT NULL,
  ended_at    INTEGER,
  mode        TEXT,
  config_json TEXT
);

CREATE TABLE IF NOT EXISTS questions (
  id                  TEXT PRIMARY KEY,
  session_id          TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
  ord                 INTEGER NOT NULL,
  category            TEXT NOT NULL,
  target_topic        TEXT,
  target_elo          INTEGER,
  rationale           TEXT,
  context_chunks_json TEXT,
  asked_at            INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS turns (
  id          TEXT PRIMARY KEY,
  question_id TEXT NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
  ord         INTEGER NOT NULL,
  speaker     TEXT NOT NULL,
  kind        TEXT,
  text        TEXT NOT NULL,
  audio_path  TEXT,
  ts          INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS judgments (
  question_id    TEXT PRIMARY KEY REFERENCES questions(id) ON DELETE CASCADE,
  rating         INTEGER NOT NULL CHECK (rating BETWEEN 1 AND 5),
  strengths_json TEXT,
  missed_json    TEXT,
  better_sketch  TEXT,
  reading_json   TEXT,
  graded_at      INTEGER NOT NULL,
  model_used     TEXT
);

CREATE TABLE IF NOT EXISTS topic_hits (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  question_id TEXT NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
  tag         TEXT NOT NULL,
  rating      INTEGER NOT NULL,
  category    TEXT NOT NULL,
  hit_at      INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_topic_hits_tag ON topic_hits(tag);
CREATE INDEX IF NOT EXISTS idx_topic_hits_at  ON topic_hits(hit_at);

CREATE TABLE IF NOT EXISTS elo_state (
  category   TEXT PRIMARY KEY,
  rating     INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS elo_history (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  category      TEXT NOT NULL,
  rating_before INTEGER,
  rating_after  INTEGER,
  delta         INTEGER,
  question_id   TEXT REFERENCES questions(id) ON DELETE SET NULL,
  at_ts         INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS plans (
  id           TEXT PRIMARY KEY,
  generated_at INTEGER NOT NULL,
  window_days  INTEGER,
  markdown     TEXT NOT NULL
);
