package repscli

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Prasad-178/reps/internal/config"
	"github.com/urfave/cli/v3"
)

func ConfigCmd() *cli.Command {
	return &cli.Command{
		Name:      "config",
		Usage:     "get or set a config key (dot path, e.g. llm.model)",
		ArgsUsage: "<key> [value]",
		Action: func(_ context.Context, c *cli.Command) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			args := c.Args().Slice()
			if len(args) == 0 {
				return printAll(cfg)
			}
			key := args[0]
			if len(args) == 1 {
				v, err := getKey(cfg, key)
				if err != nil {
					return err
				}
				fmt.Println(v)
				return nil
			}
			if err := setKey(&cfg, key, args[1]); err != nil {
				return err
			}
			return config.Save(cfg)
		},
	}
}

func printAll(cfg config.Config) error {
	fmt.Printf("config path: %s\n\n", config.Path())
	fmt.Printf("llm.provider        = %s\n", cfg.LLM.Provider)
	fmt.Printf("llm.model           = %s\n", cfg.LLM.Model)
	fmt.Printf("llm.embed_model     = %s\n", cfg.LLM.EmbedModel)
	fmt.Printf("llm.judge_model     = %s\n", cfg.LLM.JudgeModel)
	fmt.Printf("llm.rerank_model    = %s\n", cfg.LLM.RerankModel)
	fmt.Printf("llm.api_key         = %s\n", redact(cfg.LLM.APIKey))
	fmt.Printf("voice.enabled       = %v\n", cfg.Voice.Enabled)
	fmt.Printf("voice.whisper_bin   = %s\n", cfg.Voice.WhisperBin)
	fmt.Printf("voice.whisper_model = %s\n", cfg.Voice.WhisperModel)
	fmt.Printf("voice.tts_enabled   = %v\n", cfg.Voice.TTSEnabled)
	fmt.Printf("voice.tts_provider  = %s\n", cfg.Voice.TTSProvider)
	fmt.Printf("voice.tts_voice     = %s\n", cfg.Voice.TTSVoice)
	fmt.Printf("voice.tts_model     = %s\n", cfg.Voice.TTSModel)
	fmt.Printf("voice.tts_rate      = %d\n", cfg.Voice.TTSRate)
	fmt.Printf("drill.default_qs    = %d\n", cfg.Drill.DefaultQs)
	fmt.Printf("drill.followup_max  = %d\n", cfg.Drill.FollowupMax)
	fmt.Printf("elo.k_factor        = %d\n", cfg.Elo.KFactor)
	fmt.Printf("elo.start_rating    = %d\n", cfg.Elo.StartRating)
	fmt.Printf("paths.home          = %s\n", cfg.Paths.Home)
	return nil
}

func redact(s string) string {
	if s == "" {
		return "(unset)"
	}
	if len(s) <= 8 {
		return "***"
	}
	return s[:4] + "..." + s[len(s)-4:]
}

func getKey(cfg config.Config, key string) (string, error) {
	switch key {
	case "llm.provider":
		return cfg.LLM.Provider, nil
	case "llm.model":
		return cfg.LLM.Model, nil
	case "llm.embed_model":
		return cfg.LLM.EmbedModel, nil
	case "llm.judge_model":
		return cfg.LLM.JudgeModel, nil
	case "llm.rerank_model":
		return cfg.LLM.RerankModel, nil
	case "llm.api_key":
		return redact(cfg.LLM.APIKey), nil
	case "voice.enabled":
		return strconv.FormatBool(cfg.Voice.Enabled), nil
	case "voice.whisper_bin":
		return cfg.Voice.WhisperBin, nil
	case "voice.whisper_model":
		return cfg.Voice.WhisperModel, nil
	case "voice.tts_enabled":
		return strconv.FormatBool(cfg.Voice.TTSEnabled), nil
	case "voice.tts_provider":
		return cfg.Voice.TTSProvider, nil
	case "voice.tts_voice":
		return cfg.Voice.TTSVoice, nil
	case "voice.tts_model":
		return cfg.Voice.TTSModel, nil
	case "voice.tts_rate":
		return strconv.Itoa(cfg.Voice.TTSRate), nil
	case "drill.default_qs":
		return strconv.Itoa(cfg.Drill.DefaultQs), nil
	case "drill.followup_max":
		return strconv.Itoa(cfg.Drill.FollowupMax), nil
	case "elo.k_factor":
		return strconv.Itoa(cfg.Elo.KFactor), nil
	case "elo.start_rating":
		return strconv.Itoa(cfg.Elo.StartRating), nil
	case "paths.home":
		return cfg.Paths.Home, nil
	}
	return "", fmt.Errorf("unknown key: %s", key)
}

func setKey(cfg *config.Config, key, val string) error {
	switch key {
	case "llm.provider":
		cfg.LLM.Provider = val
	case "llm.model":
		cfg.LLM.Model = val
	case "llm.embed_model":
		cfg.LLM.EmbedModel = val
	case "llm.judge_model":
		cfg.LLM.JudgeModel = val
	case "llm.rerank_model":
		cfg.LLM.RerankModel = val
	case "llm.api_key":
		cfg.LLM.APIKey = val
	case "voice.enabled":
		b, err := strconv.ParseBool(val)
		if err != nil {
			return err
		}
		cfg.Voice.Enabled = b
	case "voice.whisper_bin":
		cfg.Voice.WhisperBin = val
	case "voice.whisper_model":
		cfg.Voice.WhisperModel = val
	case "voice.tts_enabled":
		b, err := strconv.ParseBool(val)
		if err != nil {
			return err
		}
		cfg.Voice.TTSEnabled = b
	case "voice.tts_provider":
		cfg.Voice.TTSProvider = val
	case "voice.tts_voice":
		cfg.Voice.TTSVoice = val
	case "voice.tts_model":
		cfg.Voice.TTSModel = val
	case "voice.tts_rate":
		n, err := strconv.Atoi(val)
		if err != nil {
			return err
		}
		cfg.Voice.TTSRate = n
	case "drill.default_qs":
		n, err := strconv.Atoi(val)
		if err != nil {
			return err
		}
		cfg.Drill.DefaultQs = n
	case "drill.followup_max":
		n, err := strconv.Atoi(val)
		if err != nil {
			return err
		}
		cfg.Drill.FollowupMax = n
	case "elo.k_factor":
		n, err := strconv.Atoi(val)
		if err != nil {
			return err
		}
		cfg.Elo.KFactor = n
	case "elo.start_rating":
		n, err := strconv.Atoi(val)
		if err != nil {
			return err
		}
		cfg.Elo.StartRating = n
	default:
		return fmt.Errorf("unknown key: %s", key)
	}
	return nil
}
