package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wonli/weixin-ilink/internal/ai"
	"gopkg.in/yaml.v3"
)

var defaultConfigNames = []string{
	"config.yaml",
	"config.yml",
	"weixin-echo.yaml",
	"weixin-echo.yml",
}

type rawConfig struct {
	AI struct {
		Provider     string `yaml:"provider"`
		Model        string `yaml:"model"`
		APIKey       string `yaml:"api_key"`
		Key          string `yaml:"key"`
		BaseURL      string `yaml:"base_url"`
		SystemPrompt string `yaml:"system_prompt"`
	} `yaml:"ai"`
}

type Config struct {
	Path string
	AI   ai.Config
}

func ResolvePath(explicitPath string) string {
	if p := strings.TrimSpace(explicitPath); p != "" {
		return p
	}

	cwd, err := os.Getwd()
	if err != nil {
		return defaultConfigNames[0]
	}
	for dir := cwd; ; dir = filepath.Dir(dir) {
		for _, name := range defaultConfigNames {
			candidate := filepath.Join(dir, name)
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}

	return filepath.Join(cwd, defaultConfigNames[0])
}

func Load(path string) (*Config, error) {
	resolved := ResolvePath(path)
	b, err := os.ReadFile(resolved)
	if err != nil {
		return nil, fmt.Errorf("read config yaml %s: %w", resolved, err)
	}

	var raw rawConfig
	if err := yaml.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("parse config yaml %s: %w", resolved, err)
	}

	apiKey := strings.TrimSpace(raw.AI.APIKey)
	if apiKey == "" {
		apiKey = strings.TrimSpace(raw.AI.Key)
	}

	cfg := &Config{
		Path: resolved,
		AI: ai.Config{
			Provider:     strings.TrimSpace(raw.AI.Provider),
			Model:        strings.TrimSpace(raw.AI.Model),
			APIKey:       apiKey,
			BaseURL:      strings.TrimSpace(raw.AI.BaseURL),
			SystemPrompt: strings.TrimSpace(raw.AI.SystemPrompt),
		},
	}
	if cfg.AI.SystemPrompt == "" {
		cfg.AI.SystemPrompt = "回答要适合微信聊天气泡，语气自然，尽量直接。"
	}
	if err := cfg.AI.Validate(); err != nil {
		return nil, fmt.Errorf("invalid ai config in %s: %w", resolved, err)
	}
	return cfg, nil
}
