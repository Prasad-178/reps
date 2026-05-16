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
