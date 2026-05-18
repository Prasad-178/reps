package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/Prasad-178/reps/internal/config"
	"github.com/Prasad-178/reps/internal/llm"
)

type configPublic struct {
	LLM struct {
		Provider    string `json:"provider"`
		Model       string `json:"model"`
		EmbedModel  string `json:"embed_model"`
		JudgeModel  string `json:"judge_model"`
		RerankModel string `json:"rerank_model"`
		APIKeyMask  string `json:"api_key_mask"`
	} `json:"llm"`
	Drill struct {
		DefaultQs   int `json:"default_qs"`
		FollowupMax int `json:"followup_max"`
		TimeWarnSec int `json:"time_warn_sec"`
	} `json:"drill"`
	Elo struct {
		KFactor     int `json:"k_factor"`
		StartRating int `json:"start_rating"`
	} `json:"elo"`
	Voice struct {
		TTSEnabled  bool   `json:"tts_enabled"`
		TTSProvider string `json:"tts_provider"`
		TTSVoice    string `json:"tts_voice"`
		TTSModel    string `json:"tts_model"`
		TTSRate     int    `json:"tts_rate"`
	} `json:"voice"`
	Paths struct {
		Home string `json:"home"`
	} `json:"paths"`
}

func (s *Server) getConfig(w http.ResponseWriter, _ *http.Request) {
	cfg, err := config.Load()
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	out := projectConfig(cfg)
	writeJSON(w, 200, out)
}

type configPatch struct {
	LLM *struct {
		Model       *string `json:"model"`
		EmbedModel  *string `json:"embed_model"`
		JudgeModel  *string `json:"judge_model"`
		RerankModel *string `json:"rerank_model"`
		APIKey      *string `json:"api_key"`
	} `json:"llm"`
	Drill *struct {
		DefaultQs   *int `json:"default_qs"`
		FollowupMax *int `json:"followup_max"`
		TimeWarnSec *int `json:"time_warn_sec"`
	} `json:"drill"`
	Elo *struct {
		KFactor     *int `json:"k_factor"`
		StartRating *int `json:"start_rating"`
	} `json:"elo"`
	Voice *struct {
		TTSEnabled  *bool   `json:"tts_enabled"`
		TTSProvider *string `json:"tts_provider"`
		TTSVoice    *string `json:"tts_voice"`
		TTSModel    *string `json:"tts_model"`
		TTSRate     *int    `json:"tts_rate"`
	} `json:"voice"`
}

func (s *Server) putConfig(w http.ResponseWriter, r *http.Request) {
	var p configPatch
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&p); err != nil {
		writeErr(w, 400, err)
		return
	}
	cfg, err := config.Load()
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	if p.LLM != nil {
		if p.LLM.Model != nil       { cfg.LLM.Model = *p.LLM.Model }
		if p.LLM.EmbedModel != nil  { cfg.LLM.EmbedModel = *p.LLM.EmbedModel }
		if p.LLM.JudgeModel != nil  { cfg.LLM.JudgeModel = *p.LLM.JudgeModel }
		if p.LLM.RerankModel != nil { cfg.LLM.RerankModel = *p.LLM.RerankModel }
		if p.LLM.APIKey != nil && *p.LLM.APIKey != "" {
			cfg.LLM.APIKey = *p.LLM.APIKey
		}
	}
	if p.Drill != nil {
		if p.Drill.DefaultQs != nil   { cfg.Drill.DefaultQs = *p.Drill.DefaultQs }
		if p.Drill.FollowupMax != nil { cfg.Drill.FollowupMax = *p.Drill.FollowupMax }
		if p.Drill.TimeWarnSec != nil { cfg.Drill.TimeWarnSec = *p.Drill.TimeWarnSec }
	}
	if p.Elo != nil {
		if p.Elo.KFactor != nil     { cfg.Elo.KFactor = *p.Elo.KFactor }
		if p.Elo.StartRating != nil { cfg.Elo.StartRating = *p.Elo.StartRating }
	}
	if p.Voice != nil {
		if p.Voice.TTSEnabled != nil  { cfg.Voice.TTSEnabled = *p.Voice.TTSEnabled }
		if p.Voice.TTSProvider != nil { cfg.Voice.TTSProvider = *p.Voice.TTSProvider }
		if p.Voice.TTSVoice != nil    { cfg.Voice.TTSVoice = *p.Voice.TTSVoice }
		if p.Voice.TTSModel != nil    { cfg.Voice.TTSModel = *p.Voice.TTSModel }
		if p.Voice.TTSRate != nil     { cfg.Voice.TTSRate = *p.Voice.TTSRate }
	}
	if err := config.Save(cfg); err != nil {
		writeErr(w, 500, err)
		return
	}
	// hot-swap running client
	s.Cfg = cfg
	s.Client.Model = cfg.LLM.Model
	s.Client.EmbedModel = cfg.LLM.EmbedModel
	s.Client.JudgeModel = cfg.LLM.JudgeModel
	s.Client.RerankModel = cfg.LLM.RerankModel
	if cfg.LLM.APIKey != "" {
		s.Client.APIKey = cfg.LLM.APIKey
	}
	writeJSON(w, 200, projectConfig(cfg))
}

func projectConfig(cfg config.Config) configPublic {
	var out configPublic
	out.LLM.Provider = cfg.LLM.Provider
	out.LLM.Model = cfg.LLM.Model
	out.LLM.EmbedModel = cfg.LLM.EmbedModel
	out.LLM.JudgeModel = cfg.LLM.JudgeModel
	out.LLM.RerankModel = cfg.LLM.RerankModel
	out.LLM.APIKeyMask = maskKey(cfg.LLM.APIKey)
	out.Drill.DefaultQs = cfg.Drill.DefaultQs
	out.Drill.FollowupMax = cfg.Drill.FollowupMax
	out.Drill.TimeWarnSec = cfg.Drill.TimeWarnSec
	out.Elo.KFactor = cfg.Elo.KFactor
	out.Elo.StartRating = cfg.Elo.StartRating
	out.Voice.TTSEnabled = cfg.Voice.TTSEnabled
	out.Voice.TTSProvider = cfg.Voice.TTSProvider
	out.Voice.TTSVoice = cfg.Voice.TTSVoice
	out.Voice.TTSModel = cfg.Voice.TTSModel
	out.Voice.TTSRate = cfg.Voice.TTSRate
	out.Paths.Home = cfg.Paths.Home
	return out
}

func maskKey(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 8 {
		return "***"
	}
	return s[:6] + "…" + s[len(s)-4:]
}

func (s *Server) probeModel(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Model string `json:"model"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<14)).Decode(&body); err != nil {
		writeErr(w, 400, err)
		return
	}
	if s.Client.APIKey == "" {
		writeErr(w, 400, fmt.Errorf("no API key set"))
		return
	}
	if err := llm.ProbeKey(r.Context(), s.Client.APIKey, body.Model); err != nil {
		writeJSON(w, 200, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}
