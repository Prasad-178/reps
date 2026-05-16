package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	LLM   LLMConfig   `toml:"llm"`
	Voice VoiceConfig `toml:"voice"`
	Drill DrillConfig `toml:"drill"`
	Elo   EloConfig   `toml:"elo"`
	Paths PathsConfig `toml:"paths"`
}

type LLMConfig struct {
	Provider    string `toml:"provider"`
	APIKey      string `toml:"api_key"`
	Model       string `toml:"model"`
	EmbedModel  string `toml:"embed_model"`
	JudgeModel  string `toml:"judge_model"`
	RerankModel string `toml:"rerank_model"`
}

type VoiceConfig struct {
	Enabled      bool   `toml:"enabled"`
	WhisperBin   string `toml:"whisper_bin"`
	WhisperModel string `toml:"whisper_model"`
	Recorder     string `toml:"recorder"`

	// TTS — pluggable provider for the interviewer to read questions aloud.
	TTSEnabled  bool   `toml:"tts_enabled"`
	TTSProvider string `toml:"tts_provider"`  // "say" (macOS, default) | "openai" | "elevenlabs"
	TTSVoice    string `toml:"tts_voice"`     // provider-specific voice name
	TTSModel    string `toml:"tts_model"`     // provider-specific model (OpenAI/ElevenLabs)
	TTSRate     int    `toml:"tts_rate"`      // words/min, used by macOS say
}

type DrillConfig struct {
	DefaultQs    int `toml:"default_qs"`
	FollowupMax  int `toml:"followup_max"`
	TimeWarnSec  int `toml:"time_warn_sec"`
}

type EloConfig struct {
	KFactor     int `toml:"k_factor"`
	StartRating int `toml:"start_rating"`
}

type PathsConfig struct {
	Home    string `toml:"home"`
	Sources string `toml:"sources"`
	Plans   string `toml:"plans"`
	Tmp     string `toml:"tmp"`
}

func Default() Config {
	home := defaultHome()
	return Config{
		LLM: LLMConfig{
			Provider:    "openrouter",
			Model:       "google/gemini-2.0-flash-001",
			EmbedModel:  "openai/text-embedding-3-small",
			JudgeModel:  "anthropic/claude-3.5-haiku",
			RerankModel: "google/gemini-2.0-flash-001",
		},
		Voice: VoiceConfig{
			Enabled:      false,
			WhisperBin:   "/opt/homebrew/bin/whisper-cli",
			WhisperModel: filepath.Join(home, "models", "ggml-base.en.bin"),
			Recorder:     "sox",
			TTSEnabled:   false,
			TTSProvider:  "say",
			TTSVoice:     "Daniel",
			TTSModel:     "tts-1",
			TTSRate:      180,
		},
		Drill: DrillConfig{
			DefaultQs:   3,
			FollowupMax: 3,
			TimeWarnSec: 240,
		},
		Elo: EloConfig{
			KFactor:     24,
			StartRating: 1200,
		},
		Paths: PathsConfig{
			Home:    home,
			Sources: filepath.Join(home, "sources"),
			Plans:   filepath.Join(home, "plans"),
			Tmp:     filepath.Join(home, ".tmp"),
		},
	}
}

func defaultHome() string {
	if v := os.Getenv("REPS_HOME"); v != "" {
		return expand(v)
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return ".reps"
	}
	return filepath.Join(h, ".reps")
}

func expand(p string) string {
	if strings.HasPrefix(p, "~/") {
		h, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(h, p[2:])
		}
	}
	return p
}

func Path() string {
	return filepath.Join(defaultHome(), "config.toml")
}

func Load() (Config, error) {
	cfg := Default()
	p := Path()
	if _, err := os.Stat(p); err == nil {
		if _, err := toml.DecodeFile(p, &cfg); err != nil {
			return cfg, fmt.Errorf("decode config %s: %w", p, err)
		}
	}
	applyEnv(&cfg)
	cfg.Paths.Home = expand(cfg.Paths.Home)
	cfg.Paths.Sources = expand(cfg.Paths.Sources)
	cfg.Paths.Plans = expand(cfg.Paths.Plans)
	cfg.Paths.Tmp = expand(cfg.Paths.Tmp)
	cfg.Voice.WhisperModel = expand(cfg.Voice.WhisperModel)
	return cfg, nil
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("OPENROUTER_API_KEY"); v != "" {
		cfg.LLM.APIKey = v
	}
	if v := os.Getenv("REPS_MODEL"); v != "" {
		cfg.LLM.Model = v
	}
	if v := os.Getenv("REPS_EMBED_MODEL"); v != "" {
		cfg.LLM.EmbedModel = v
	}
	if v := os.Getenv("REPS_JUDGE_MODEL"); v != "" {
		cfg.LLM.JudgeModel = v
	}
}

func EnsureDirs(cfg Config) error {
	for _, d := range []string{cfg.Paths.Home, cfg.Paths.Sources, cfg.Paths.Plans, cfg.Paths.Tmp} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", d, err)
		}
	}
	return nil
}

func Save(cfg Config) error {
	if err := EnsureDirs(cfg); err != nil {
		return err
	}
	f, err := os.Create(Path())
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}

func DBPath(cfg Config) string {
	return filepath.Join(cfg.Paths.Home, "reps.db")
}
