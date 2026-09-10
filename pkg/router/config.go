package router

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Port       int                `json:"port"`
	Providers  []ProviderConfig   `json:"providers"`
	Combos     []Combo            `json:"combos"`
	TokenSaver TokenSaverConfig   `json:"token_saver"`
	Logging    LoggingConfig      `json:"logging"`
	Vault      VaultConfig        `json:"vault"`
}

type ProviderConfig struct {
	Name     string            `json:"name"`
	Type     string            `json:"type"` // "openai", "claude", "gemini", "custom"
	BaseURL  string            `json:"base_url"`
	APIKey   string            `json:"api_key,omitempty"`
	Models   []string          `json:"models"`
	Headers  map[string]string `json:"headers,omitempty"`
	Enabled  bool              `json:"enabled"`
	Tier     int               `json:"tier"` // 1=subscription, 2=cheap, 3=free
	Priority int               `json:"priority"`
}

type Combo struct {
	Name     string   `json:"name"`
	Models   []string `json:"models"` // ordered by priority
	Fallback bool     `json:"fallback"`
}

type TokenSaverConfig struct {
	RTK      bool `json:"rtk"`
	Caveman  bool `json:"caveman"`
	Ponytail string `json:"ponytail"` // "off", "lite", "full", "ultra"
	Headroom string `json:"headroom_url"` // optional external compressor
}

type LoggingConfig struct {
	Enabled bool   `json:"enabled"`
	Level   string `json:"level"` // "debug", "info", "warn", "error"
	File    string `json:"file,omitempty"`
}

type VaultConfig struct {
	Enabled  bool   `json:"enabled"`
	KeyFile  string `json:"key_file,omitempty"`
}

func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	configDir := filepath.Join(home, ".7relay")
	os.MkdirAll(configDir, 0700)

	return &Config{
		Port: 20128,
		Providers: []ProviderConfig{
			{
				Name:    "openai",
				Type:    "openai",
				BaseURL: "https://api.openai.com/v1",
				Models:  []string{"gpt-4o", "gpt-4o-mini", "o1", "o1-mini"},
				Enabled: false,
				Tier:    2,
			},
			{
				Name:    "anthropic",
				Type:    "claude",
				BaseURL: "https://api.anthropic.com/v1",
				Models:  []string{"claude-sonnet-4-20250514", "claude-3-5-sonnet-20241022"},
				Enabled: false,
				Tier:    2,
			},
			{
				Name:    "gemini",
				Type:    "gemini",
				BaseURL: "https://generativelanguage.googleapis.com/v1beta",
				Models:  []string{"gemini-2.5-flash", "gemini-2.5-pro"},
				Enabled: false,
				Tier:    2,
			},
		},
		Combos: []Combo{
			{
				Name:     "default",
				Models:   []string{"claude-sonnet-4-20250514", "gpt-4o", "gemini-2.5-flash"},
				Fallback: true,
			},
		},
		TokenSaver: TokenSaverConfig{
			RTK:      true,
			Caveman:  false,
			Ponytail: "off",
		},
		Logging: LoggingConfig{
			Enabled: true,
			Level:   "info",
		},
	}
}

func LoadConfig() (*Config, error) {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".7relay", "config.json")

	// If no config exists, create default
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg := DefaultConfig()
		if err := SaveConfig(cfg); err != nil {
			return nil, fmt.Errorf("failed to save default config: %w", err)
		}
		return cfg, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	cfg := &Config{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return cfg, nil
}

func SaveConfig(cfg *Config) error {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".7relay", "config.json")

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0600)
}
