package tui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/Prasad-178/reps/internal/config"
	"github.com/Prasad-178/reps/internal/ingest"
	"github.com/Prasad-178/reps/internal/llm"
	"github.com/Prasad-178/reps/internal/store"
)

type WizardResult struct {
	APIKey      string
	Model       string
	EmbedModel  string
	Sources     []SourcePlan
	SkipRebuild bool
}

type SourcePlan struct {
	Kind string // resume | github | portfolio | jd | linkedin | x | note
	Ref  string
	From string // optional paste file for linkedin/x
}

const totalSteps = 5

// RunInit runs the full onboarding wizard. Returns nil on success.
// If reset=true, wipes ~/.reps/* before starting.
func RunInit(ctx context.Context, version string, reset bool) error {
	cfgEarly, _ := config.Load()
	if reset {
		if err := wipeRepsHome(cfgEarly); err != nil {
			return fmt.Errorf("reset: %w", err)
		}
		fmt.Println(OK.Render(IconOK) + " wiped " + Mono.Render(cfgEarly.Paths.Home))
		fmt.Println()
	}

	fmt.Println(Banner(version))

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := config.EnsureDirs(cfg); err != nil {
		return err
	}

	// ---- Step 1: API key (+ live probe) ------------------------------------
	fmt.Print(Heading(1, totalSteps, "API key"))
	apiKey, source := resolveAPIKey(cfg)
	if apiKey == "" {
		apiKey, err = promptAPIKey(ctx)
		if err != nil {
			return err
		}
		if err := writeEnvKey(cfg, apiKey); err != nil {
			fmt.Println(Wn.Render(IconWarn+" couldn't save to .env: ") + Dim.Render(err.Error()))
		} else {
			fmt.Println(OK.Render(IconOK) + " saved to " + Mono.Render(filepath.Join(cfg.Paths.Home, ".env")))
		}
	} else {
		fmt.Println(OK.Render(IconOK) + " key detected " + Dim.Render("("+source+")"))
		// validate quickly even if pre-existing
		if err := RunWithSpinner("validating existing key", func(ctx context.Context, _ func(string)) error {
			return llm.ProbeKey(ctx, apiKey, cfg.LLM.Model)
		}); err != nil {
			// give user a chance to override if it's broken
			fmt.Println(Wn.Render(IconWarn) + " existing key didn't pass — paste a new one")
			apiKey, err = promptAPIKey(ctx)
			if err != nil {
				return err
			}
			if err := writeEnvKey(cfg, apiKey); err != nil {
				fmt.Println(Wn.Render(IconWarn+" couldn't save to .env: ") + Dim.Render(err.Error()))
			}
		}
	}
	cfg.LLM.APIKey = apiKey
	fmt.Println()

	// ---- Step 2: Model (+ live validation for custom IDs) -----------------
	fmt.Print(Heading(2, totalSteps, "Model"))
	chosen, err := pickModel(ctx, cfg)
	if err != nil {
		return err
	}
	cfg.LLM.Model = chosen
	fmt.Println(OK.Render(IconOK) + " primary " + Dim.Render(chosen))
	fmt.Println()

	// ---- Step 3: Source kinds (multi-select, validated) -------------------
	fmt.Print(Heading(3, totalSteps, "Sources"))
	fmt.Println(Dim.Render("  use ↑/↓ to navigate, x or space to select, enter to confirm"))
	fmt.Println()

	var kinds []string
	err = huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("What should the agents read?").
				Description("Press SPACE on each item you want to add (selected ones get a • marker).\nPick everything you have. Anything left out can be added later via `reps add ...`.").
				Options(
					huh.NewOption("Resume PDF",                  "resume"),
					huh.NewOption("GitHub username (repos+READMEs)", "github"),
					huh.NewOption("Portfolio URL",                "portfolio"),
					huh.NewOption("Job description URLs",         "jd"),
					huh.NewOption("LinkedIn (URL + paste)",       "linkedin"),
					huh.NewOption("X handle (paste)",             "x"),
					huh.NewOption("Markdown note",                "note"),
				).
				Validate(func(v []string) error {
					if len(v) == 0 {
						return errors.New("press SPACE to select at least one (you pressed ENTER without selecting). Or pick a single option to skip later.")
					}
					return nil
				}).
				Value(&kinds),
		),
	).Run()
	if err != nil {
		return wrapAbort(err)
	}
	fmt.Println(OK.Render(IconOK) + " selected " + Dim.Render(strings.Join(kinds, ", ")))
	fmt.Println()

	plans, err := collectSourceRefs(kinds)
	if err != nil {
		return err
	}
	checkBinaries(plans)

	// ---- Step 4: Ingest with spinners --------------------------------------
	fmt.Print(Heading(4, totalSteps, "Ingest"))
	if err := config.Save(cfg); err != nil {
		return err
	}
	if len(plans) == 0 {
		fmt.Println(Dim.Render("nothing to ingest"))
	} else {
		dim := llm.EmbedDim(cfg.LLM.EmbedModel)
		if dim == 0 {
			dim = 1536
		}
		s, err := store.Open(config.DBPath(cfg), dim)
		if err != nil {
			return err
		}
		defer s.Close()
		client := llm.New(cfg.LLM.APIKey, cfg.LLM.Model, cfg.LLM.EmbedModel,
			cfg.LLM.JudgeModel, cfg.LLM.RerankModel)
		pipe := ingest.NewPipeline(cfg, s, client)

		for _, p := range plans {
			label := fmt.Sprintf("ingest %s %s", p.Kind, Dim.Render("("+truncate(p.Ref, 40)+")"))
			if err := RunWithSpinner(label, func(ctx context.Context, log func(string)) error {
				return ingestOne(ctx, pipe, p, log)
			}); err != nil {
				fmt.Println(Wn.Render(IconWarn) + " continuing past failed source")
			}
		}
		fmt.Println()

		// ---- Step 5: Profile rebuild ---------------------------------------
		fmt.Print(Heading(5, totalSteps, "Profile"))
		if cfg.LLM.APIKey == "" {
			fmt.Println(Wn.Render(IconWarn) + " skipping rebuild — no API key")
		} else {
			err := RunWithSpinner("chunking + embedding + synthesizing profile", func(ctx context.Context, log func(string)) error {
				log("running pipeline.Rebuild()…")
				return pipe.Rebuild(ctx)
			})
			if err != nil {
				fmt.Println(Wn.Render(IconWarn) + " profile rebuild failed — fix with `reps profile --rebuild`")
			}
		}
	}
	fmt.Println()

	// ---- Done card --------------------------------------------------------
	summary := []string{
		Mono.Render(IconArrow+" config  ") + Dim.Render(config.Path()),
		Mono.Render(IconArrow+" data    ") + Dim.Render(cfg.Paths.Home),
		Mono.Render(IconArrow+" model   ") + Dim.Render(cfg.LLM.Model),
		Mono.Render(IconArrow+" sources ") + Dim.Render(fmt.Sprintf("%d ingested", len(plans))),
	}
	fmt.Println(Done(summary))
	return nil
}

// ----------------------------------------------------------------------------
// API key + model pickers
// ----------------------------------------------------------------------------

func promptAPIKey(ctx context.Context) (string, error) {
	for {
		var entered string
		err := huh.NewForm(
			huh.NewGroup(
				huh.NewNote().
					Title("OpenRouter API key").
					Description("Get one at openrouter.ai/keys (free tier covers thousands of drills).\nSaved encrypted-at-rest to ~/.reps/.env (mode 0600); never sent anywhere except OpenRouter."),
				huh.NewInput().
					Title("Paste your key").
					Placeholder("sk-or-v1-...").
					EchoMode(huh.EchoModePassword).
					Validate(func(s string) error {
						s = strings.TrimSpace(s)
						if s == "" {
							return errors.New("required")
						}
						if !strings.HasPrefix(s, "sk-or-") {
							return errors.New("OpenRouter keys start with sk-or-")
						}
						return nil
					}).
					Value(&entered),
			),
		).Run()
		if err != nil {
			return "", wrapAbort(err)
		}
		key := strings.TrimSpace(entered)

		if err := RunWithSpinner("validating key against OpenRouter", func(ctx context.Context, _ func(string)) error {
			return llm.ProbeKey(ctx, key, "google/gemini-2.0-flash-001")
		}); err != nil {
			// allow another try
			var retry bool
			_ = huh.NewForm(huh.NewGroup(
				huh.NewConfirm().Title("Try a different key?").Value(&retry),
			)).Run()
			if !retry {
				return "", fmt.Errorf("invalid key")
			}
			continue
		}
		return key, nil
	}
}

func pickModel(ctx context.Context, cfg config.Config) (string, error) {
	const customSentinel = "__custom__"
	current := cfg.LLM.Model
	if current == "" {
		current = "google/gemini-2.0-flash-001"
	}
	options := []huh.Option[string]{
		huh.NewOption("Gemini 2.0 Flash  — cheap, fast (recommended)", "google/gemini-2.0-flash-001"),
		huh.NewOption("Claude 3.5 Haiku — sharper, ~3× cost", "anthropic/claude-3.5-haiku"),
		huh.NewOption("Claude 3.5 Sonnet — best quality, ~10× cost", "anthropic/claude-3.5-sonnet"),
		huh.NewOption("GPT-4o mini       — middling", "openai/gpt-4o-mini"),
		huh.NewOption("Llama 3.3 70B (free tier)", "meta-llama/llama-3.3-70b-instruct:free"),
		huh.NewOption("Custom model ID…", customSentinel),
	}
	if current != "" {
		options = append(options, huh.NewOption("Keep current ("+current+")", current))
	}

	for {
		var pick string
		err := huh.NewForm(huh.NewGroup(
			huh.NewSelect[string]().
				Title("Primary model (Planner + Interviewer + Coach)").
				Description("Used for all agent calls. Override later: `reps config llm.model <id>`.").
				Options(options...).
				Value(&pick),
		)).Run()
		if err != nil {
			return "", wrapAbort(err)
		}
		if pick != customSentinel {
			return pick, nil
		}
		// Custom path — text input + live probe
		custom, err := promptCustomModel(ctx, cfg.LLM.APIKey)
		if err != nil {
			// allow user to back out to the select list
			fmt.Println(Wn.Render(IconWarn) + " " + err.Error() + Dim.Render(" — choose again"))
			continue
		}
		return custom, nil
	}
}

func promptCustomModel(ctx context.Context, apiKey string) (string, error) {
	for {
		var id string
		err := huh.NewForm(huh.NewGroup(
			huh.NewNote().
				Title("Custom model ID").
				Description("Any model on openrouter.ai/models, in <provider>/<model> form.\nExamples: anthropic/claude-3-opus, mistralai/mistral-large, google/gemini-pro-1.5."),
			huh.NewInput().
				Title("Model ID").
				Placeholder("provider/model-name").
				Validate(func(s string) error {
					s = strings.TrimSpace(s)
					if s == "" {
						return errors.New("required")
					}
					if !strings.Contains(s, "/") {
						return errors.New("must look like provider/model")
					}
					return nil
				}).
				Value(&id),
		)).Run()
		if err != nil {
			return "", wrapAbort(err)
		}
		id = strings.TrimSpace(id)

		if apiKey == "" {
			// can't probe without key — accept and warn
			fmt.Println(Wn.Render(IconWarn) + " no API key set — skipping live validation")
			return id, nil
		}
		if err := RunWithSpinner("checking "+id+" on OpenRouter", func(ctx context.Context, _ func(string)) error {
			return llm.ProbeKey(ctx, apiKey, id)
		}); err != nil {
			// don't return — retry
			var retry bool
			_ = huh.NewForm(huh.NewGroup(
				huh.NewConfirm().
					Title("Try another model ID?").
					Description(fmt.Sprintf("Couldn't verify %s. Reason: %v", id, err)).
					Value(&retry),
			)).Run()
			if !retry {
				return "", errors.New("user gave up on custom model")
			}
			continue
		}
		return id, nil
	}
}

// ----------------------------------------------------------------------------
// Source ref collection
// ----------------------------------------------------------------------------

func collectSourceRefs(kinds []string) ([]SourcePlan, error) {
	var plans []SourcePlan
	for _, k := range kinds {
		switch k {
		case "resume":
			plan, err := promptOne("Resume PDF path", "~/Documents/resume.pdf", k, validatePDFPath)
			if err != nil {
				return nil, err
			}
			plan.Ref = expandPath(plan.Ref)
			plans = append(plans, plan)
		case "github":
			plan, err := promptOne("GitHub username", "Prasad-178", k, validateNonEmpty)
			if err != nil {
				return nil, err
			}
			plans = append(plans, plan)
		case "portfolio":
			plan, err := promptOne("Portfolio URL", "https://prasadjs.me", k, validateURL)
			if err != nil {
				return nil, err
			}
			plans = append(plans, plan)
		case "jd":
			i := 0
			for {
				var v string
				title := "Job description URL"
				if i > 0 {
					title = fmt.Sprintf("Another JD URL (blank to finish, %d added)", i)
				}
				if err := huh.NewForm(huh.NewGroup(
					huh.NewInput().Title(title).Placeholder("https://jobs.example.com/staff-eng").Value(&v),
				)).Run(); err != nil {
					return nil, wrapAbort(err)
				}
				v = strings.TrimSpace(v)
				if v == "" {
					break
				}
				plans = append(plans, SourcePlan{Kind: "jd", Ref: v})
				i++
			}
		case "linkedin":
			plan, err := promptOne("LinkedIn URL or @handle", "https://www.linkedin.com/in/you", k, validateNonEmpty)
			if err != nil {
				return nil, err
			}
			plan.From, err = promptFile("Path to a text/markdown paste of your LinkedIn content (blank to skip — site blocks scrapers)")
			if err != nil {
				return nil, err
			}
			plans = append(plans, plan)
		case "x":
			plan, err := promptOne("X handle", "@you", k, validateNonEmpty)
			if err != nil {
				return nil, err
			}
			plan.From, err = promptFile("Path to a text paste of recent posts (blank to skip)")
			if err != nil {
				return nil, err
			}
			plans = append(plans, plan)
		case "note":
			plan, err := promptOne("Markdown note path", "~/Documents/notes/about-me.md", k, validatePath)
			if err != nil {
				return nil, err
			}
			plan.Ref = expandPath(plan.Ref)
			plans = append(plans, plan)
		}
	}
	return plans, nil
}

func promptOne(title, placeholder, kind string, validate func(string) error) (SourcePlan, error) {
	var v string
	err := huh.NewForm(huh.NewGroup(
		huh.NewInput().Title(title).Placeholder(placeholder).Validate(validate).Value(&v),
	)).Run()
	if err != nil {
		return SourcePlan{}, wrapAbort(err)
	}
	return SourcePlan{Kind: kind, Ref: strings.TrimSpace(v)}, nil
}

func promptFile(title string) (string, error) {
	var v string
	err := huh.NewForm(huh.NewGroup(
		huh.NewInput().Title(title).Placeholder("/path/to/paste.txt or blank").Value(&v),
	)).Run()
	if err != nil {
		return "", wrapAbort(err)
	}
	v = strings.TrimSpace(v)
	if v == "" {
		return "", nil
	}
	return expandPath(v), nil
}

// ----------------------------------------------------------------------------
// Helpers
// ----------------------------------------------------------------------------

func ingestOne(ctx context.Context, pipe *ingest.Pipeline, p SourcePlan, log func(string)) error {
	switch p.Kind {
	case "resume":
		log("parsing PDF…")
		_, err := pipe.IngestResume(ctx, p.Ref)
		return err
	case "github":
		log("listing repos via gh CLI…")
		_, err := pipe.IngestGitHub(ctx, p.Ref)
		return err
	case "portfolio":
		log("fetching page…")
		_, err := pipe.IngestPortfolio(ctx, p.Ref)
		return err
	case "jd":
		log("scraping + extracting card…")
		_, err := pipe.IngestJD(ctx, p.Ref)
		return err
	case "linkedin":
		log("reading paste…")
		_, err := pipe.IngestLinkedIn(ctx, p.Ref, p.From)
		return err
	case "x":
		log("reading paste…")
		_, err := pipe.IngestX(ctx, p.Ref, p.From)
		return err
	case "note":
		log("reading note…")
		_, err := pipe.IngestNote(ctx, p.Ref)
		return err
	}
	return fmt.Errorf("unknown kind %q", p.Kind)
}

func checkBinaries(plans []SourcePlan) {
	needs := map[string]string{}
	for _, p := range plans {
		switch p.Kind {
		case "resume":
			needs["pdftotext"] = "brew install poppler"
		case "github":
			needs["gh"] = "brew install gh && gh auth login"
		}
	}
	missing := []string{}
	for bin, install := range needs {
		if _, err := exec.LookPath(bin); err != nil {
			missing = append(missing, Wn.Render(IconWarn+" missing ")+Mono.Render(bin)+Dim.Render(" — "+install))
		}
	}
	if len(missing) > 0 {
		fmt.Println(strings.Join(missing, "\n"))
		fmt.Println()
	}
}

func wipeRepsHome(cfg config.Config) error {
	if !looksSafe(cfg.Paths.Home) {
		return fmt.Errorf("refusing to wipe suspicious path %q", cfg.Paths.Home)
	}
	return os.RemoveAll(cfg.Paths.Home)
}

func looksSafe(p string) bool {
	abs, err := filepath.Abs(p)
	if err != nil || abs == "" {
		return false
	}
	for _, bad := range []string{"/", "/usr", "/etc", "/home", "/Users", "/tmp", "/var"} {
		if abs == bad {
			return false
		}
	}
	return strings.Contains(strings.ToLower(filepath.Base(abs)), "reps")
}

func resolveAPIKey(cfg config.Config) (string, string) {
	if v := os.Getenv("OPENROUTER_API_KEY"); v != "" {
		return v, "env: OPENROUTER_API_KEY"
	}
	if cfg.LLM.APIKey != "" {
		return cfg.LLM.APIKey, "config: " + config.Path()
	}
	return "", ""
}

func writeEnvKey(cfg config.Config, key string) error {
	path := filepath.Join(cfg.Paths.Home, ".env")
	existing := map[string]string{}
	if b, err := os.ReadFile(path); err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			line = strings.TrimPrefix(line, "export ")
			eq := strings.IndexByte(line, '=')
			if eq <= 0 {
				continue
			}
			existing[strings.TrimSpace(line[:eq])] = strings.TrimSpace(line[eq+1:])
		}
	}
	existing["OPENROUTER_API_KEY"] = key

	var sb strings.Builder
	sb.WriteString("# reps — environment. Auto-loaded by every `reps` command.\n")
	for k, v := range existing {
		fmt.Fprintf(&sb, "%s=%s\n", k, v)
	}
	return os.WriteFile(path, []byte(sb.String()), 0o600)
}

// ---- validators

func validateNonEmpty(s string) error {
	if strings.TrimSpace(s) == "" {
		return errors.New("required")
	}
	return nil
}

func validateURL(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return errors.New("required")
	}
	if !strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://") {
		return errors.New("URL must start with http(s)://")
	}
	return nil
}

func validatePath(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return errors.New("required")
	}
	if _, err := os.Stat(expandPath(s)); err != nil {
		return fmt.Errorf("can't read %s", s)
	}
	return nil
}

func validatePDFPath(s string) error {
	if err := validatePath(s); err != nil {
		return err
	}
	if !strings.EqualFold(filepath.Ext(s), ".pdf") {
		return errors.New("must be a .pdf")
	}
	return nil
}

func expandPath(p string) string {
	if strings.HasPrefix(p, "~/") {
		if h, err := os.UserHomeDir(); err == nil {
			return filepath.Join(h, p[2:])
		}
	}
	return p
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func wrapAbort(err error) error {
	if errors.Is(err, huh.ErrUserAborted) {
		return fmt.Errorf("setup cancelled")
	}
	return err
}
