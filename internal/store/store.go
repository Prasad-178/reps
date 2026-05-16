package store

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"strconv"
	"strings"
	"time"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schemaSQL string

type Store struct {
	DB       *sql.DB
	embedDim int
}

func init() {
	sqlite_vec.Auto()
}

func Open(path string, embedDim int) (*Store, error) {
	db, err := sql.Open("sqlite3", path+"?_fk=1&_journal=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{DB: db, embedDim: embedDim}
	if err := s.applySchema(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.DB.Close() }

func (s *Store) applySchema() error {
	sqlStr := strings.ReplaceAll(schemaSQL, "{{EMBED_DIM}}", strconv.Itoa(s.embedDim))
	if _, err := s.DB.Exec(sqlStr); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	return s.upsertMeta("embed_dim", strconv.Itoa(s.embedDim))
}

func (s *Store) upsertMeta(k, v string) error {
	_, err := s.DB.Exec(`INSERT INTO schema_meta(key,value) VALUES(?,?)
		ON CONFLICT(key) DO UPDATE SET value=excluded.value`, k, v)
	return err
}

func (s *Store) GetMeta(k string) (string, error) {
	var v string
	err := s.DB.QueryRow(`SELECT value FROM schema_meta WHERE key=?`, k).Scan(&v)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return v, err
}

// RebuildVecTable drops chunks_vec and recreates it with the new dim.
// Caller should immediately re-embed every row in chunks.
func (s *Store) RebuildVecTable(newDim int) error {
	ctx := context.Background()
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS chunks_vec`); err != nil {
		return err
	}
	create := fmt.Sprintf(`CREATE VIRTUAL TABLE chunks_vec USING vec0(
		chunk_id TEXT PRIMARY KEY,
		embedding FLOAT[%d]
	)`, newDim)
	if _, err := tx.ExecContext(ctx, create); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_meta(key,value) VALUES('embed_dim',?)
		ON CONFLICT(key) DO UPDATE SET value=excluded.value`, strconv.Itoa(newDim)); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	s.embedDim = newDim
	return nil
}

func (s *Store) EmbedDim() int { return s.embedDim }

// ---- sources

type Source struct {
	ID        string
	Kind      string
	Ref       string
	RawPath   string
	FetchedAt time.Time
	MetaJSON  string
}

func (s *Store) InsertSource(src Source) error {
	_, err := s.DB.Exec(`INSERT INTO sources(id,kind,ref,raw_path,fetched_at,meta_json)
		VALUES(?,?,?,?,?,?)`,
		src.ID, src.Kind, src.Ref, src.RawPath, src.FetchedAt.Unix(), src.MetaJSON)
	return err
}

func (s *Store) ListSources() ([]Source, error) {
	rows, err := s.DB.Query(`SELECT id,kind,ref,raw_path,fetched_at,COALESCE(meta_json,'')
		FROM sources ORDER BY fetched_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Source
	for rows.Next() {
		var src Source
		var ts int64
		if err := rows.Scan(&src.ID, &src.Kind, &src.Ref, &src.RawPath, &ts, &src.MetaJSON); err != nil {
			return nil, err
		}
		src.FetchedAt = time.Unix(ts, 0)
		out = append(out, src)
	}
	return out, rows.Err()
}

func (s *Store) DeleteChunksBySource(sourceID string) error {
	rows, err := s.DB.Query(`SELECT id FROM chunks WHERE source_id=?`, sourceID)
	if err != nil {
		return err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	rows.Close()
	for _, id := range ids {
		if _, err := s.DB.Exec(`DELETE FROM chunks_vec WHERE chunk_id=?`, id); err != nil {
			return err
		}
	}
	_, err = s.DB.Exec(`DELETE FROM chunks WHERE source_id=?`, sourceID)
	return err
}

// ---- chunks

type Chunk struct {
	ID       string
	SourceID string
	Ord      int
	Text     string
}

func (s *Store) InsertChunk(c Chunk, embedding []float32) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`INSERT INTO chunks(id,source_id,ord,text) VALUES(?,?,?,?)`,
		c.ID, c.SourceID, c.Ord, c.Text); err != nil {
		return err
	}
	if embedding != nil {
		if len(embedding) != s.embedDim {
			return fmt.Errorf("embedding dim %d != schema dim %d", len(embedding), s.embedDim)
		}
		blob, err := sqlite_vec.SerializeFloat32(embedding)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO chunks_vec(chunk_id,embedding) VALUES(?,?)`, c.ID, blob); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ---- jd cards

type JDCard struct {
	ID       string
	SourceID string
	Company  string
	Role     string
	CardJSON string
	Priority int
}

func (s *Store) InsertJDCard(j JDCard) error {
	_, err := s.DB.Exec(`INSERT INTO jd_cards(id,source_id,company,role,card_json,priority)
		VALUES(?,?,?,?,?,?)`,
		j.ID, j.SourceID, j.Company, j.Role, j.CardJSON, j.Priority)
	return err
}

func (s *Store) ListJDCards() ([]JDCard, error) {
	rows, err := s.DB.Query(`SELECT id,source_id,COALESCE(company,''),COALESCE(role,''),card_json,priority
		FROM jd_cards ORDER BY priority DESC, rowid DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []JDCard
	for rows.Next() {
		var j JDCard
		if err := rows.Scan(&j.ID, &j.SourceID, &j.Company, &j.Role, &j.CardJSON, &j.Priority); err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

// ---- profile

func (s *Store) UpsertProfile(markdown, model string) error {
	_, err := s.DB.Exec(`INSERT INTO profile(id,markdown,built_at,model_used) VALUES(1,?,?,?)
		ON CONFLICT(id) DO UPDATE SET markdown=excluded.markdown, built_at=excluded.built_at, model_used=excluded.model_used`,
		markdown, time.Now().Unix(), model)
	return err
}

func (s *Store) GetProfile() (string, time.Time, string, error) {
	var md, model string
	var ts int64
	err := s.DB.QueryRow(`SELECT markdown,built_at,COALESCE(model_used,'') FROM profile WHERE id=1`).
		Scan(&md, &ts, &model)
	if err == sql.ErrNoRows {
		return "", time.Time{}, "", nil
	}
	if err != nil {
		return "", time.Time{}, "", err
	}
	return md, time.Unix(ts, 0), model, nil
}

func (s *Store) HasProfile() (bool, error) {
	md, _, _, err := s.GetProfile()
	return md != "", err
}

// ---- sessions / questions / turns

type Session struct {
	ID         string
	StartedAt  time.Time
	EndedAt    *time.Time
	Mode       string
	ConfigJSON string
}

func (s *Store) InsertSession(sess Session) error {
	var ended *int64
	if sess.EndedAt != nil {
		t := sess.EndedAt.Unix()
		ended = &t
	}
	_, err := s.DB.Exec(`INSERT INTO sessions(id,started_at,ended_at,mode,config_json)
		VALUES(?,?,?,?,?)`,
		sess.ID, sess.StartedAt.Unix(), ended, sess.Mode, sess.ConfigJSON)
	return err
}

func (s *Store) CloseSession(id string, endedAt time.Time) error {
	_, err := s.DB.Exec(`UPDATE sessions SET ended_at=? WHERE id=?`, endedAt.Unix(), id)
	return err
}

type Question struct {
	ID                string
	SessionID         string
	Ord               int
	Category          string
	TargetTopic       string
	TargetELO         int
	Rationale         string
	ContextChunksJSON string
	AskedAt           time.Time
}

func (s *Store) InsertQuestion(q Question) error {
	_, err := s.DB.Exec(`INSERT INTO questions(id,session_id,ord,category,target_topic,target_elo,rationale,context_chunks_json,asked_at)
		VALUES(?,?,?,?,?,?,?,?,?)`,
		q.ID, q.SessionID, q.Ord, q.Category, q.TargetTopic, q.TargetELO,
		q.Rationale, q.ContextChunksJSON, q.AskedAt.Unix())
	return err
}

type Turn struct {
	ID         string
	QuestionID string
	Ord        int
	Speaker    string // interviewer | candidate
	Kind       string // opening | followup | answer
	Text       string
	AudioPath  string
	Ts         time.Time
}

func (s *Store) InsertTurn(t Turn) error {
	_, err := s.DB.Exec(`INSERT INTO turns(id,question_id,ord,speaker,kind,text,audio_path,ts)
		VALUES(?,?,?,?,?,?,?,?)`,
		t.ID, t.QuestionID, t.Ord, t.Speaker, t.Kind, t.Text, t.AudioPath, t.Ts.Unix())
	return err
}

func (s *Store) ListTurnsForQuestion(qID string) ([]Turn, error) {
	rows, err := s.DB.Query(`SELECT id,question_id,ord,speaker,COALESCE(kind,''),text,COALESCE(audio_path,''),ts
		FROM turns WHERE question_id=? ORDER BY ord`, qID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Turn
	for rows.Next() {
		var t Turn
		var ts int64
		if err := rows.Scan(&t.ID, &t.QuestionID, &t.Ord, &t.Speaker, &t.Kind, &t.Text, &t.AudioPath, &ts); err != nil {
			return nil, err
		}
		t.Ts = time.Unix(ts, 0)
		out = append(out, t)
	}
	return out, rows.Err()
}

// RecentQuestionsWithRatings returns up to limit recent questions joined with judgments (if any).
type RecentQuestion struct {
	Tag      string
	Category string
	Rating   int
	AskedAt  time.Time
}

func (s *Store) RecentTopics(limit int) ([]RecentQuestion, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.DB.Query(`
		SELECT q.target_topic, q.category, COALESCE(j.rating, 0), q.asked_at
		FROM questions q
		LEFT JOIN judgments j ON j.question_id = q.id
		ORDER BY q.asked_at DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RecentQuestion
	for rows.Next() {
		var r RecentQuestion
		var ts int64
		if err := rows.Scan(&r.Tag, &r.Category, &r.Rating, &ts); err != nil {
			return nil, err
		}
		r.AskedAt = time.Unix(ts, 0)
		out = append(out, r)
	}
	return out, rows.Err()
}

type WeakestTopic struct {
	Tag        string
	Hits       int
	MeanRating float64
}

func (s *Store) WeakestTopics(limit int) ([]WeakestTopic, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.DB.Query(`
		SELECT tag, COUNT(*) AS hits, AVG(rating) AS mean
		FROM topic_hits
		GROUP BY tag
		HAVING (mean < 3.5) OR (hits >= 3 AND mean < 4)
		ORDER BY mean ASC, hits DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []WeakestTopic
	for rows.Next() {
		var w WeakestTopic
		if err := rows.Scan(&w.Tag, &w.Hits, &w.MeanRating); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// ---- history queries

type SessionSummary struct {
	ID        string
	StartedAt time.Time
	EndedAt   *time.Time
	Mode      string
	QCount    int
	MeanRate  float64
}

func (s *Store) RecentSessions(limit int) ([]SessionSummary, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.DB.Query(`
		SELECT s.id, s.started_at, s.ended_at, COALESCE(s.mode,''),
		  (SELECT COUNT(*) FROM questions q WHERE q.session_id = s.id) AS qc,
		  COALESCE((SELECT AVG(j.rating) FROM judgments j
		            JOIN questions q ON q.id = j.question_id
		            WHERE q.session_id = s.id), 0)
		FROM sessions s
		ORDER BY s.started_at DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SessionSummary
	for rows.Next() {
		var ss SessionSummary
		var started int64
		var ended sql.NullInt64
		if err := rows.Scan(&ss.ID, &started, &ended, &ss.Mode, &ss.QCount, &ss.MeanRate); err != nil {
			return nil, err
		}
		ss.StartedAt = time.Unix(started, 0)
		if ended.Valid {
			t := time.Unix(ended.Int64, 0)
			ss.EndedAt = &t
		}
		out = append(out, ss)
	}
	return out, rows.Err()
}

type QuestionWithJudgment struct {
	Question Question
	Rating   int
	HasJ     bool
	Strengths []string
	Missed    []string
	Better    string
}

func (s *Store) QuestionsBySession(sessionID string) ([]Question, error) {
	rows, err := s.DB.Query(`SELECT id,session_id,ord,category,target_topic,target_elo,
		COALESCE(rationale,''),COALESCE(context_chunks_json,''),asked_at
		FROM questions WHERE session_id=? ORDER BY ord`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Question
	for rows.Next() {
		var q Question
		var asked int64
		if err := rows.Scan(&q.ID, &q.SessionID, &q.Ord, &q.Category, &q.TargetTopic,
			&q.TargetELO, &q.Rationale, &q.ContextChunksJSON, &asked); err != nil {
			return nil, err
		}
		q.AskedAt = time.Unix(asked, 0)
		out = append(out, q)
	}
	return out, rows.Err()
}

func (s *Store) GetJudgment(questionID string) (Judgment, bool, error) {
	var j Judgment
	var ts int64
	err := s.DB.QueryRow(`SELECT question_id,rating,COALESCE(strengths_json,''),COALESCE(missed_json,''),
		COALESCE(better_sketch,''),COALESCE(reading_json,''),graded_at,COALESCE(model_used,'')
		FROM judgments WHERE question_id=?`, questionID).Scan(
		&j.QuestionID, &j.Rating, &j.StrengthsJSON, &j.MissedJSON, &j.BetterSketch,
		&j.ReadingJSON, &ts, &j.ModelUsed)
	if err == sql.ErrNoRows {
		return j, false, nil
	}
	if err != nil {
		return j, false, err
	}
	j.GradedAt = time.Unix(ts, 0)
	return j, true, nil
}

// ---- judgments + topic hits

type Judgment struct {
	QuestionID    string
	Rating        int
	StrengthsJSON string
	MissedJSON    string
	BetterSketch  string
	ReadingJSON   string
	GradedAt      time.Time
	ModelUsed     string
}

func (s *Store) InsertJudgment(j Judgment) error {
	_, err := s.DB.Exec(`INSERT INTO judgments(question_id,rating,strengths_json,missed_json,better_sketch,reading_json,graded_at,model_used)
		VALUES(?,?,?,?,?,?,?,?)
		ON CONFLICT(question_id) DO UPDATE SET
		  rating=excluded.rating, strengths_json=excluded.strengths_json, missed_json=excluded.missed_json,
		  better_sketch=excluded.better_sketch, reading_json=excluded.reading_json,
		  graded_at=excluded.graded_at, model_used=excluded.model_used`,
		j.QuestionID, j.Rating, j.StrengthsJSON, j.MissedJSON, j.BetterSketch,
		j.ReadingJSON, j.GradedAt.Unix(), j.ModelUsed)
	return err
}

func (s *Store) InsertTopicHit(questionID, tag, category string, rating int) error {
	_, err := s.DB.Exec(`INSERT INTO topic_hits(question_id,tag,rating,category,hit_at)
		VALUES(?,?,?,?,?)`,
		questionID, tag, rating, category, time.Now().Unix())
	return err
}

// ---- ELO

func (s *Store) GetAllELO() (map[string]int, error) {
	rows, err := s.DB.Query(`SELECT category, rating FROM elo_state`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var cat string
		var r int
		if err := rows.Scan(&cat, &r); err != nil {
			return nil, err
		}
		out[cat] = r
	}
	return out, rows.Err()
}

func (s *Store) GetELO(category string, defaultRating int) (int, error) {
	var r int
	err := s.DB.QueryRow(`SELECT rating FROM elo_state WHERE category=?`, category).Scan(&r)
	if err == sql.ErrNoRows {
		return defaultRating, nil
	}
	return r, err
}

func (s *Store) UpsertELO(category string, rating int) error {
	_, err := s.DB.Exec(`INSERT INTO elo_state(category,rating,updated_at) VALUES(?,?,?)
		ON CONFLICT(category) DO UPDATE SET rating=excluded.rating, updated_at=excluded.updated_at`,
		category, rating, time.Now().Unix())
	return err
}

func (s *Store) InsertELOHistory(category string, before, after, delta int, questionID string) error {
	var qid any
	if questionID != "" {
		qid = questionID
	}
	_, err := s.DB.Exec(`INSERT INTO elo_history(category,rating_before,rating_after,delta,question_id,at_ts)
		VALUES(?,?,?,?,?,?)`, category, before, after, delta, qid, time.Now().Unix())
	return err
}

// ELODeltaSince returns the net ELO delta per category over the last `dur`.
func (s *Store) ELODeltaSince(since time.Time) (map[string]int, error) {
	rows, err := s.DB.Query(`SELECT category, SUM(delta) FROM elo_history WHERE at_ts >= ? GROUP BY category`,
		since.Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var c string
		var d int
		if err := rows.Scan(&c, &d); err != nil {
			return nil, err
		}
		out[c] = d
	}
	return out, rows.Err()
}
