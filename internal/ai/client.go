package ai

import (
	"fmt"
	"strings"
	"time"

	"github.com/wonli/weixin-ilink/internal/ai/deepseek"
	"github.com/wonli/weixin-ilink/internal/ai/kimi"
	"github.com/wonli/weixin-ilink/internal/ai/ollama"
)

type Config struct {
	Provider     string
	Model        string
	APIKey       string
	BaseURL      string
	SystemPrompt string
}

func (c Config) NormalizedProvider() string {
	return strings.ToLower(strings.TrimSpace(c.Provider))
}

func (c Config) Validate() error {
	switch c.NormalizedProvider() {
	case "deepseek", "kimi":
		if strings.TrimSpace(c.Model) == "" {
			return fmt.Errorf("ai.model is required when provider=%s", c.Provider)
		}
		if strings.TrimSpace(c.APIKey) == "" {
			return fmt.Errorf("ai.api_key is required when provider=%s", c.Provider)
		}
	case "ollama":
		if strings.TrimSpace(c.BaseURL) == "" {
			return fmt.Errorf("ai.base_url is required when provider=ollama")
		}
		if strings.TrimSpace(c.Model) == "" {
			return fmt.Errorf("ai.model is required when provider=ollama")
		}
	default:
		return fmt.Errorf("unsupported ai.provider %q, supported: deepseek, kimi, ollama", c.Provider)
	}
	return nil
}

func Reply(cfg Config, userPrompt string) (string, error) {
	return ReplyStream(cfg, PromptContext{}, userPrompt, nil)
}

func ReplyStream(cfg Config, promptCtx PromptContext, userPrompt string, onDelta func(string)) (string, error) {
	if err := cfg.Validate(); err != nil {
		return "", err
	}
	if promptCtx.Now.IsZero() {
		promptCtx.Now = time.Now()
	}
	systemPrompt := buildSystemPrompt(cfg, promptCtx)

	var content strings.Builder
	handler := func(msg []byte) error {
		content.Write(msg)
		return nil
	}

	switch cfg.NormalizedProvider() {
	case "deepseek":
		err := deepseek.NewClient().StartChat(
			strings.TrimSpace(cfg.Model),
			strings.TrimSpace(cfg.APIKey),
			systemPrompt,
			userPrompt,
			handler,
			onDelta,
		)
		return normalizePlainText(content.String()), err
	case "kimi":
		err := kimi.NewClient().StartChat(
			strings.TrimSpace(cfg.Model),
			strings.TrimSpace(cfg.APIKey),
			systemPrompt,
			userPrompt,
			handler,
			onDelta,
		)
		return normalizePlainText(content.String()), err
	case "ollama":
		err := ollama.NewClient().StartChat(
			strings.TrimSpace(cfg.BaseURL),
			strings.TrimSpace(cfg.Model),
			systemPrompt,
			userPrompt,
			handler,
			onDelta,
		)
		return normalizePlainText(content.String()), err
	default:
		return "", fmt.Errorf("unsupported ai.provider %q", cfg.Provider)
	}
}
