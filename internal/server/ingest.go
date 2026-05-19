package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/Prasad-178/reps/internal/ingest"
	"github.com/google/uuid"
)

func (s *Server) ingestRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/sources/resume", s.ingestResume)
	mux.HandleFunc("POST /api/sources/github", s.ingestGithub)
	mux.HandleFunc("POST /api/sources/portfolio", s.ingestPortfolio)
	mux.HandleFunc("POST /api/sources/jd", s.ingestJD)
	mux.HandleFunc("POST /api/sources/linkedin", s.ingestLinkedIn)
	mux.HandleFunc("POST /api/sources/x", s.ingestX)
	mux.HandleFunc("POST /api/sources/note", s.ingestNote)
	mux.HandleFunc("DELETE /api/sources/{id}", s.deleteSource)
	mux.HandleFunc("POST /api/profile/rebuild", s.rebuildProfile)
}

func (s *Server) pipeline() *ingest.Pipeline {
	return ingest.NewPipeline(s.Cfg, s.Store, s.Client)
}

// ---- resume (multipart upload)

func (s *Server) ingestResume(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		writeErr(w, 400, fmt.Errorf("parse multipart: %w", err))
		return
	}
	f, hdr, err := r.FormFile("file")
	if err != nil {
		writeErr(w, 400, fmt.Errorf("missing 'file' field: %w", err))
		return
	}
	defer f.Close()

	tmp := filepath.Join(s.Cfg.Paths.Tmp, uuid.NewString()+"-"+filepath.Base(hdr.Filename))
	dst, err := os.Create(tmp)
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	if _, err := io.Copy(dst, f); err != nil {
		dst.Close()
		_ = os.Remove(tmp)
		writeErr(w, 500, err)
		return
	}
	dst.Close()
	defer os.Remove(tmp)

	id, err := s.pipeline().IngestResume(r.Context(), tmp)
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	writeJSON(w, 200, map[string]any{"id": id})
}

// ---- url/handle/text body sources

func (s *Server) ingestGithub(w http.ResponseWriter, r *http.Request) {
	var body struct{ User string `json:"user"` }
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil || body.User == "" {
		writeErr(w, 400, fmt.Errorf("missing 'user'"))
		return
	}
	id, err := s.pipeline().IngestGitHub(r.Context(), body.User)
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	writeJSON(w, 200, map[string]any{"id": id})
}

func (s *Server) ingestPortfolio(w http.ResponseWriter, r *http.Request) {
	var body struct {
		URL  string `json:"url"`
		Path string `json:"path"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
		writeErr(w, 400, err)
		return
	}
	ref := body.URL
	if ref == "" {
		ref = body.Path
	}
	if ref == "" {
		writeErr(w, 400, fmt.Errorf("require 'url' or 'path'"))
		return
	}
	id, err := s.pipeline().IngestPortfolio(r.Context(), ref)
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	writeJSON(w, 200, map[string]any{"id": id})
}

func (s *Server) ingestJD(w http.ResponseWriter, r *http.Request) {
	var body struct{ URL string `json:"url"` }
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil || body.URL == "" {
		writeErr(w, 400, fmt.Errorf("missing 'url'"))
		return
	}
	id, err := s.pipeline().IngestJD(r.Context(), body.URL)
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	writeJSON(w, 200, map[string]any{"id": id})
}

func (s *Server) ingestLinkedIn(w http.ResponseWriter, r *http.Request) {
	if err := s.ingestPasteForm(r, w, "linkedin"); err != nil {
		return
	}
}

func (s *Server) ingestX(w http.ResponseWriter, r *http.Request) {
	if err := s.ingestPasteForm(r, w, "x"); err != nil {
		return
	}
}

// ingestPasteForm accepts JSON {ref, text} for linkedin / x.
// Writes the text to a temp file and dispatches to the kind-specific pipeline.
func (s *Server) ingestPasteForm(r *http.Request, w http.ResponseWriter, kind string) error {
	var body struct {
		Ref  string `json:"ref"`
		Text string `json:"text"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil || body.Ref == "" || body.Text == "" {
		writeErr(w, 400, fmt.Errorf("require 'ref' and 'text'"))
		return fmt.Errorf("bad")
	}
	tmp := filepath.Join(s.Cfg.Paths.Tmp, uuid.NewString()+"-"+kind+".txt")
	if err := os.WriteFile(tmp, []byte(body.Text), 0o600); err != nil {
		writeErr(w, 500, err)
		return err
	}
	defer os.Remove(tmp)

	var id string
	var err error
	switch kind {
	case "linkedin":
		id, err = s.pipeline().IngestLinkedIn(r.Context(), body.Ref, tmp)
	case "x":
		id, err = s.pipeline().IngestX(r.Context(), body.Ref, tmp)
	}
	if err != nil {
		writeErr(w, 500, err)
		return err
	}
	writeJSON(w, 200, map[string]any{"id": id})
	return nil
}

// ---- note (raw markdown body)

func (s *Server) ingestNote(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name    string `json:"name"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil || body.Content == "" {
		writeErr(w, 400, fmt.Errorf("require 'content'"))
		return
	}
	if body.Name == "" {
		body.Name = "web-note.md"
	}
	tmp := filepath.Join(s.Cfg.Paths.Tmp, uuid.NewString()+"-"+filepath.Base(body.Name))
	if err := os.WriteFile(tmp, []byte(body.Content), 0o600); err != nil {
		writeErr(w, 500, err)
		return
	}
	defer os.Remove(tmp)
	id, err := s.pipeline().IngestNote(r.Context(), tmp)
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	writeJSON(w, 200, map[string]any{"id": id})
}

// ---- delete source

func (s *Server) deleteSource(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeErr(w, 400, fmt.Errorf("missing id"))
		return
	}
	// chunks + chunks_vec rows
	if err := s.Store.DeleteChunksBySource(id); err != nil {
		writeErr(w, 500, fmt.Errorf("delete chunks: %w", err))
		return
	}
	// jd_cards children (cascade FK is set; explicit row delete here too)
	if _, err := s.Store.DB.Exec(`DELETE FROM jd_cards WHERE source_id=?`, id); err != nil {
		writeErr(w, 500, fmt.Errorf("delete jd_card: %w", err))
		return
	}
	// raw file
	rows, err := s.Store.DB.Query(`SELECT raw_path FROM sources WHERE id=?`, id)
	if err == nil {
		for rows.Next() {
			var p string
			if rows.Scan(&p) == nil {
				_ = os.Remove(p)
			}
		}
		rows.Close()
	}
	if _, err := s.Store.DB.Exec(`DELETE FROM sources WHERE id=?`, id); err != nil {
		writeErr(w, 500, err)
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

// ---- profile rebuild (long-running) — fire-and-forget on the server, but
// returns a job id the client can poll. Simple impl: a single in-memory
// status entry per server instance.

type rebuildStatus struct {
	Running bool   `json:"running"`
	StartedAt int64 `json:"started_at"`
	FinishedAt int64 `json:"finished_at"`
	Error string `json:"error,omitempty"`
	LastLine string `json:"last_line,omitempty"`
}

var currentRebuild rebuildStatus

func (s *Server) rebuildProfile(w http.ResponseWriter, r *http.Request) {
	if currentRebuild.Running {
		writeJSON(w, 200, currentRebuild)
		return
	}
	currentRebuild = rebuildStatus{Running: true, StartedAt: nowUnix()}
	go func() {
		ctx := context.Background()
		err := s.pipeline().Rebuild(ctx)
		currentRebuild.Running = false
		currentRebuild.FinishedAt = nowUnix()
		if err != nil {
			currentRebuild.Error = err.Error()
		}
	}()
	writeJSON(w, 202, currentRebuild)
}

func (s *Server) rebuildStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, currentRebuild)
}

func nowUnix() int64 { return time.Now().Unix() }
