package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// ThemeColors holds the color definitions for a terminal theme.
type ThemeColors struct {
	Bg              string `json:"bg,omitempty"`
	Fg              string `json:"fg,omitempty"`
	TitleBar        string `json:"title_bar,omitempty"`
	TitleText       string `json:"title_text,omitempty"`
	Cursor          string `json:"cursor,omitempty"`
	BtnHoverClose   string `json:"btn_hover_close,omitempty"`
	BtnHoverNeutral string `json:"btn_hover_neutral,omitempty"`
	TabActiveBg     string `json:"tab_active_bg,omitempty"`
	TabInactiveBg   string `json:"tab_inactive_bg,omitempty"`
	TabHoverBg      string `json:"tab_hover_bg,omitempty"`
}

// Config holds the user preferences for the terminal.
type Config struct {
	FontFamily  string       `json:"font_family"`
	FontSize    int          `json:"font_size"`
	Theme       string       `json:"theme"`
	CustomTheme *ThemeColors `json:"custom_theme,omitempty"`
}

// DefaultConfig returns the default terminal settings.
func DefaultConfig() *Config {
	return &Config{
		FontFamily: "Iosevka Fixed, Go Mono, monospace",
		FontSize:   14,
		Theme:      "default",
	}
}

// GetConfigPath resolves the path to ~/.spark/config.json.
func GetConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".spark", "config.json"), nil
}

// Load reads the configuration from disk, or creates a default one if it doesn't exist.
func Load() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Create default config and save it to disk
			cfg := DefaultConfig()
			err = Save(cfg)
			return cfg, err
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Save writes the configuration to disk.
func Save(cfg *Config) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	// Ensure the ~/.spark directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}
