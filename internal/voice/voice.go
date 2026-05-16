package voice

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Prasad-178/reps/internal/config"
	"github.com/google/uuid"
)

type Recorder struct {
	Cfg config.Config
}

func New(cfg config.Config) *Recorder { return &Recorder{Cfg: cfg} }

// Available reports whether the configured recorder + whisper binary are usable.
func (r *Recorder) Available() error {
	if _, err := exec.LookPath(r.recorderBin()); err != nil {
		return fmt.Errorf("recorder %q not on PATH (brew install sox)", r.recorderBin())
	}
	if r.Cfg.Voice.WhisperBin == "" {
		return fmt.Errorf("voice.whisper_bin not set in config")
	}
	if _, err := os.Stat(expand(r.Cfg.Voice.WhisperBin)); err != nil {
		return fmt.Errorf("whisper bin %q not found", r.Cfg.Voice.WhisperBin)
	}
	if r.Cfg.Voice.WhisperModel == "" {
		return fmt.Errorf("voice.whisper_model not set")
	}
	if _, err := os.Stat(expand(r.Cfg.Voice.WhisperModel)); err != nil {
		return fmt.Errorf("whisper model %q not found (download with scripts/install-whisper.sh)", r.Cfg.Voice.WhisperModel)
	}
	return nil
}

func (r *Recorder) recorderBin() string {
	if r.Cfg.Voice.Recorder != "" {
		return r.Cfg.Voice.Recorder
	}
	return "sox"
}

// RecordAndTranscribe records mic audio until the user presses Enter, then
// transcribes it via whisper-cli. Returns the transcribed text.
func (r *Recorder) RecordAndTranscribe(ctx context.Context, in io.Reader, out io.Writer) (string, error) {
	wav := filepath.Join(r.Cfg.Paths.Tmp, uuid.NewString()+".wav")
	defer os.Remove(wav)

	fmt.Fprintln(out, "[voice] press Enter to start recording...")
	if _, err := bufio.NewReader(in).ReadString('\n'); err != nil {
		return "", err
	}
	fmt.Fprintln(out, "[voice] recording — press Enter again to stop.")

	recCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	cmd := exec.CommandContext(recCtx, r.recorderBin(), "-d", "-t", "wav", "-r", "16000", "-c", "1", wav)
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("start recorder: %w", err)
	}
	go func() {
		_, _ = bufio.NewReader(in).ReadString('\n')
		cancel()
	}()
	if err := cmd.Wait(); err != nil {
		// sox returns non-zero on signal; that's fine
	}
	time.Sleep(150 * time.Millisecond)

	if st, err := os.Stat(wav); err != nil || st.Size() < 1000 {
		return "", fmt.Errorf("no audio captured")
	}

	fmt.Fprintln(out, "[voice] transcribing...")
	txt, err := r.transcribe(ctx, wav)
	if err != nil {
		return "", err
	}
	txt = strings.TrimSpace(txt)
	if txt == "" {
		return "", fmt.Errorf("transcript empty")
	}
	fmt.Fprintln(out, "[voice] transcript:")
	fmt.Fprintln(out, indent(txt, "  "))
	fmt.Fprint(out, "[voice] accept? (Y/n/e to edit) ")
	resp, _ := bufio.NewReader(in).ReadString('\n')
	resp = strings.TrimSpace(strings.ToLower(resp))
	switch resp {
	case "n":
		return "", fmt.Errorf("transcript rejected")
	case "e":
		fmt.Fprintln(out, "Type the edited answer; finish with /end on a new line.")
		return readUntilEnd(in)
	default:
		return txt, nil
	}
}

func (r *Recorder) transcribe(ctx context.Context, wav string) (string, error) {
	bin := expand(r.Cfg.Voice.WhisperBin)
	model := expand(r.Cfg.Voice.WhisperModel)
	out := wav + ".txt"
	defer os.Remove(out)
	cmd := exec.CommandContext(ctx, bin,
		"-m", model,
		"-f", wav,
		"-otxt",
		"-nt",
		"-of", strings.TrimSuffix(wav, ".wav"),
	)
	if _, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("whisper-cli: %w", err)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		return "", err
	}
	return string(b), nil
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

func indent(s, pad string) string {
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = pad + lines[i]
	}
	return strings.Join(lines, "\n")
}

func readUntilEnd(r io.Reader) (string, error) {
	rd := bufio.NewReader(r)
	var sb strings.Builder
	for {
		line, err := rd.ReadString('\n')
		t := strings.TrimRight(line, "\r\n")
		if t == "/end" {
			break
		}
		sb.WriteString(line)
		if err != nil {
			if err == io.EOF {
				break
			}
			return sb.String(), err
		}
	}
	return strings.TrimRight(sb.String(), "\n"), nil
}
