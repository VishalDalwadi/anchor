// Package config loads and saves anchor's global settings,
// ~/.anchor/config.json. A missing file means all defaults.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config is the shape of ~/.anchor/config.json.
type Config struct {
	// Editor is the command anchor opens text in (e.g. `journal add` with no
	// text), as program and arguments; the file to edit is appended last.
	// Kept as a list so paths with spaces need no quoting. Empty means
	// type into the terminal instead.
	Editor []string `json:"editor,omitempty"`

	// Lookahead is how many days ahead `brief` reminds about due tasks,
	// unless --lookahead is given. nil means the built-in default (0 is a
	// valid setting: today only).
	Lookahead *int `json:"lookahead,omitempty"`
}

// Path is ~/.anchor/config.json.
func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".anchor", "config.json"), nil
}

// Load reads the config; a missing file yields the defaults.
func Load() (*Config, error) {
	p, err := Path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", p, err)
	}
	return &c, nil
}

// Save writes the config back.
func (c *Config) Save() error {
	p, err := Path()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, append(data, '\n'), 0o644)
}
