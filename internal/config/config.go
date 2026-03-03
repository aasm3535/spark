package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// ─── Theme ────────────────────────────────────────────────────────────────────

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

// ─── Keybinds ─────────────────────────────────────────────────────────────────

// Keybinds maps logical action names to key chord strings.
//
// A chord is written as a "+"-separated list of modifiers followed by a key
// name, e.g. "Ctrl+Shift+T" or "Ctrl+PageDown".
//
// Recognised modifier tokens (case-insensitive): Ctrl, Shift, Alt.
// Key names follow Gio conventions: letters are upper-case single chars
// ("T", "W"), special keys use their Gio name without the "Name" prefix
// ("PageUp", "PageDown", "UpArrow", "DownArrow").
//
// An empty string disables the action.
type Keybinds struct {
	// Tab management
	NewTab   string `json:"new_tab"`
	CloseTab string `json:"close_tab"`
	NextTab  string `json:"next_tab"`
	PrevTab  string `json:"prev_tab"`

	// Scrollback
	ScrollUp       string `json:"scroll_up"`
	ScrollDown     string `json:"scroll_down"`
	ScrollPageUp   string `json:"scroll_page_up"`
	ScrollPageDown string `json:"scroll_page_down"`
}

// DefaultKeybinds returns the built-in key bindings.
func DefaultKeybinds() Keybinds {
	return Keybinds{
		NewTab:         "Ctrl+Shift+T",
		CloseTab:       "Ctrl+Shift+W",
		NextTab:        "Ctrl+PageDown",
		PrevTab:        "Ctrl+PageUp",
		ScrollUp:       "Shift+UpArrow",
		ScrollDown:     "Shift+DownArrow",
		ScrollPageUp:   "Shift+PageUp",
		ScrollPageDown: "Shift+PageDown",
	}
}

// Merge returns a copy of d with any non-empty fields from o applied on top.
// This lets users override only the binds they care about.
func (d Keybinds) Merge(o Keybinds) Keybinds {
	if o.NewTab != "" {
		d.NewTab = o.NewTab
	}
	if o.CloseTab != "" {
		d.CloseTab = o.CloseTab
	}
	if o.NextTab != "" {
		d.NextTab = o.NextTab
	}
	if o.PrevTab != "" {
		d.PrevTab = o.PrevTab
	}
	if o.ScrollUp != "" {
		d.ScrollUp = o.ScrollUp
	}
	if o.ScrollDown != "" {
		d.ScrollDown = o.ScrollDown
	}
	if o.ScrollPageUp != "" {
		d.ScrollPageUp = o.ScrollPageUp
	}
	if o.ScrollPageDown != "" {
		d.ScrollPageDown = o.ScrollPageDown
	}
	return d
}

// ─── Config ───────────────────────────────────────────────────────────────────

// Config holds all user preferences for spark.
type Config struct {
	FontFamily  string       `json:"font_family"`
	FontSize    int          `json:"font_size"`
	Theme       string       `json:"theme"`
	CustomTheme *ThemeColors `json:"custom_theme,omitempty"`
	Keybinds    Keybinds     `json:"keybinds"`
}

// DefaultConfig returns the default terminal settings.
func DefaultConfig() *Config {
	return &Config{
		FontFamily: "Iosevka Fixed, Go Mono, monospace",
		FontSize:   14,
		Theme:      "default",
		Keybinds:   DefaultKeybinds(),
	}
}

// ResolvedKeybinds returns the effective keybinds: defaults merged with any
// user overrides stored in cfg.Keybinds.
func (cfg *Config) ResolvedKeybinds() Keybinds {
	return DefaultKeybinds().Merge(cfg.Keybinds)
}

// ─── Persistence ──────────────────────────────────────────────────────────────

// GetConfigPath resolves the path to ~/.spark/config.json.
func GetConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".spark", "config.json"), nil
}

// Load reads the configuration from disk, or creates and saves a default one
// if the file does not yet exist.
func Load() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			cfg := DefaultConfig()
			return cfg, Save(cfg)
		}
		return nil, err
	}

	// Start from defaults so missing fields stay at their default values.
	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Save writes the configuration to ~/.spark/config.json.
func Save(cfg *Config) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}
