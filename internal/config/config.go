// Package config loads ~/.anchor/config.json and builds the active
// storage.Backend from it. The config file can hold settings for more than
// one backend at once (e.g. keep Dropbox credentials on file while running
// on local, or vice versa) — the top-level "backend" field just picks which
// one is active. Local is the default and needs no config at all.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/VishalDalwadi/anchor/internal/storage"
)

const (
	BackendLocal   = "local"
	BackendDropbox = "dropbox"

	dropboxTokenEnvVar = "ANCHOR_DROPBOX_TOKEN"
	defaultDropboxRoot = "/Anchor"
)

// DropboxSettings holds the config needed to talk to a Dropbox account.
// The token here is a fallback only: ANCHOR_DROPBOX_TOKEN always wins, so
// the token never has to live in this file (or the CLI) if the caller
// prefers an environment variable.
type DropboxSettings struct {
	Token string `json:"token,omitempty"`
	Root  string `json:"root,omitempty"`
}

// Backends holds settings for every backend anchor knows about besides
// local, which needs none. More than one may be populated at a time;
// Backend selects which is active.
type Backends struct {
	Dropbox *DropboxSettings `json:"dropbox,omitempty"`
}

// Config is the shape of ~/.anchor/config.json.
type Config struct {
	Backend  string   `json:"backend"` // "local" (default) or "dropbox"
	Backends Backends `json:"backends"`
}

func dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".anchor"), nil
}

func path() (string, error) {
	d, err := dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "config.json"), nil
}

// Load reads ~/.anchor/config.json. A missing file is not an error: it
// yields the zero-config default (local backend).
func Load() (*Config, error) {
	p, err := path()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return &Config{Backend: BackendLocal}, nil
	}
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", p, err)
	}
	if cfg.Backend == "" {
		cfg.Backend = BackendLocal
	}
	return &cfg, nil
}

// NewBackend builds the storage.Backend selected by cfg.Backend. Command
// logic never sees which concrete type comes back — only the interface.
func NewBackend(cfg *Config) (storage.Backend, error) {
	switch cfg.Backend {
	case "", BackendLocal:
		d, err := dir()
		if err != nil {
			return nil, err
		}
		return storage.NewLocal(filepath.Join(d, "data"))

	case BackendDropbox:
		token := os.Getenv(dropboxTokenEnvVar)
		root := defaultDropboxRoot
		if ds := cfg.Backends.Dropbox; ds != nil {
			if token == "" {
				token = ds.Token
			}
			if ds.Root != "" {
				root = ds.Root
			}
		}
		if token == "" {
			return nil, fmt.Errorf(
				"dropbox backend selected but no token found: set %q in config.json under \"backends\".\"dropbox\".\"token\", or export %s",
				"backends.dropbox.token", dropboxTokenEnvVar)
		}
		return storage.NewDropbox(token, root), nil

	default:
		return nil, fmt.Errorf("unknown backend %q: must be %q or %q", cfg.Backend, BackendLocal, BackendDropbox)
	}
}
