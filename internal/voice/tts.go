package voice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/Prasad-178/reps/internal/config"
	"github.com/google/uuid"
)

// Speaker is the TTS interface. Speak is non-blocking by default: it returns
// after kicking off a background goroutine. Wait blocks until the most recent
// utterance has finished playing. Calling Speak while a previous utterance
// is still playing cancels the previous one.
type Speaker struct {
	Cfg config.Config

	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

func NewSpeaker(cfg config.Config) *Speaker { return &Speaker{Cfg: cfg} }

func (s *Speaker) Enabled() bool { return s.Cfg.Voice.TTSEnabled }

// Speak starts speaking `text` in the background. Returns immediately.
func (s *Speaker) Speak(text string) {
	if !s.Enabled() || text == "" {
		return
	}
	s.mu.Lock()
	if s.cancel != nil {
		s.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	done := make(chan struct{})
	s.done = done
	s.mu.Unlock()

	go func() {
		defer close(done)
		_ = s.speakBlocking(ctx, text)
	}()
}

// Wait blocks until the latest Speak finishes. Safe to call when nothing
// is playing.
func (s *Speaker) Wait() {
	s.mu.Lock()
	d := s.done
	s.mu.Unlock()
	if d != nil {
		<-d
	}
}

func (s *Speaker) speakBlocking(ctx context.Context, text string) error {
	switch s.Cfg.Voice.TTSProvider {
	case "", "say":
		return s.sayMac(ctx, text)
	case "openai":
		return s.openAITTS(ctx, text)
	case "elevenlabs":
		return s.elevenlabsTTS(ctx, text)
	default:
		return fmt.Errorf("tts provider %q not supported", s.Cfg.Voice.TTSProvider)
	}
}

func (s *Speaker) sayMac(ctx context.Context, text string) error {
	args := []string{}
	if v := s.Cfg.Voice.TTSVoice; v != "" {
		args = append(args, "-v", v)
	}
	if r := s.Cfg.Voice.TTSRate; r > 0 {
		args = append(args, "-r", strconv.Itoa(r))
	}
	args = append(args, text)
	cmd := exec.CommandContext(ctx, "say", args...)
	return cmd.Run()
}

func (s *Speaker) openAITTS(ctx context.Context, text string) error {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		return fmt.Errorf("OPENAI_API_KEY not set")
	}
	model := s.Cfg.Voice.TTSModel
	if model == "" {
		model = "tts-1"
	}
	voice := s.Cfg.Voice.TTSVoice
	if voice == "" {
		voice = "onyx"
	}
	body, _ := json.Marshal(map[string]any{
		"model":  model,
		"input":  text,
		"voice":  voice,
		"format": "mp3",
	})
	req, err := http.NewRequestWithContext(ctx, "POST",
		"https://api.openai.com/v1/audio/speech", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	httpC := &http.Client{Timeout: 60 * time.Second}
	resp, err := httpC.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("openai tts http %d: %s", resp.StatusCode, string(b))
	}
	return s.playStream(ctx, resp.Body, "mp3")
}

func (s *Speaker) elevenlabsTTS(ctx context.Context, text string) error {
	key := os.Getenv("ELEVENLABS_API_KEY")
	if key == "" {
		return fmt.Errorf("ELEVENLABS_API_KEY not set")
	}
	voice := s.Cfg.Voice.TTSVoice
	if voice == "" {
		voice = "21m00Tcm4TlvDq8ikWAM" // Rachel
	}
	model := s.Cfg.Voice.TTSModel
	if model == "" {
		model = "eleven_turbo_v2_5"
	}
	body, _ := json.Marshal(map[string]any{
		"text":     text,
		"model_id": model,
	})
	url := fmt.Sprintf("https://api.elevenlabs.io/v1/text-to-speech/%s", voice)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("xi-api-key", key)
	req.Header.Set("Content-Type", "application/json")
	httpC := &http.Client{Timeout: 60 * time.Second}
	resp, err := httpC.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("elevenlabs http %d: %s", resp.StatusCode, string(b))
	}
	return s.playStream(ctx, resp.Body, "mp3")
}

// playStream writes the audio to a temp file and pipes it through afplay
// (macOS) or ffplay (cross-platform fallback).
func (s *Speaker) playStream(ctx context.Context, body io.Reader, ext string) error {
	tmp := filepath.Join(s.Cfg.Paths.Tmp, uuid.NewString()+"."+ext)
	defer os.Remove(tmp)
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, body); err != nil {
		_ = f.Close()
		return err
	}
	_ = f.Close()

	player := pickPlayer()
	if player == "" {
		return fmt.Errorf("no audio player on PATH (need afplay, mpg123, or ffplay)")
	}
	args := []string{tmp}
	if player == "ffplay" {
		args = []string{"-autoexit", "-nodisp", "-loglevel", "error", tmp}
	}
	cmd := exec.CommandContext(ctx, player, args...)
	return cmd.Run()
}

func pickPlayer() string {
	for _, p := range []string{"afplay", "mpg123", "ffplay"} {
		if _, err := exec.LookPath(p); err == nil {
			return p
		}
	}
	return ""
}
