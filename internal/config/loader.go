package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"deduplicator/internal/model"
)

// Load reads and parses a YAML config file at path. Missing files surface as
// an os.IsNotExist-able error so callers can decide to fall back.
func Load(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	cfg := Default()
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return Config{}, fmt.Errorf("config: parse %s: %w", path, err)
	}
	return cfg, nil
}

// LoadAuto walks a list of well-known locations until it finds a config
// file. It returns (defaultCfg, "", nil) when none is present.
func LoadAuto() (Config, string, error) {
	for _, p := range autoPaths() {
		if _, err := os.Stat(p); err == nil {
			cfg, err := Load(p)
			if err != nil {
				return Config{}, p, err
			}
			return cfg, p, nil
		}
	}
	return Default(), "", nil
}

func autoPaths() []string {
	var out []string
	out = append(out, "./dedup.yaml", "./dedup.yml")
	if runtime.GOOS == "windows" {
		if appdata := os.Getenv("APPDATA"); appdata != "" {
			out = append(out, filepath.Join(appdata, "dedup", "config.yaml"))
		}
	} else {
		xdg := os.Getenv("XDG_CONFIG_HOME")
		if xdg == "" {
			if home, err := os.UserHomeDir(); err == nil {
				xdg = filepath.Join(home, ".config")
			}
		}
		if xdg != "" {
			out = append(out, filepath.Join(xdg, "dedup", "config.yaml"))
		}
	}
	return out
}

// ApplyEnv layers DEDUP_* environment variables onto cfg. Only scalar fields
// are supported; arrays must come from YAML or flags.
//
// Supported variables (documented in Docs/10-design-cli.md § 10.5):
//
//	DEDUP_MODE, DEDUP_DRY_RUN, DEDUP_SUPPRESS_NAME,
//	DEDUP_OUTPUT_DIR, DEDUP_DEST_DIR,
//	DEDUP_WORKERS, DEDUP_ACTION_WORKERS,
//	DEDUP_LOG_LEVEL, DEDUP_LOG_FORMAT,
//	DEDUP_AUDIT_DIR, DEDUP_STRICT, DEDUP_FOLLOW_SYMLINKS,
//	DEDUP_CANONICAL_POLICY, DEDUP_CANONICAL_FOLDER
func ApplyEnv(cfg Config) Config {
	if v, ok := getEnv("DEDUP_MODE"); ok {
		cfg.Mode = model.Mode(strings.ToLower(v))
	}
	if v, ok := getEnv("DEDUP_DRY_RUN"); ok {
		cfg.DryRun = boolValue(v, cfg.DryRun)
	}
	if v, ok := getEnv("DEDUP_SUPPRESS_NAME"); ok {
		cfg.SuppressName = boolValue(v, cfg.SuppressName)
	}
	if v, ok := getEnv("DEDUP_OUTPUT_DIR"); ok {
		cfg.Output.Dir = v
	}
	if v, ok := getEnv("DEDUP_DEST_DIR"); ok {
		cfg.Destination.Dir = v
	}
	if v, ok := getEnv("DEDUP_WORKERS"); ok {
		cfg.Concurrency.Workers = intValue(v, cfg.Concurrency.Workers)
	}
	if v, ok := getEnv("DEDUP_ACTION_WORKERS"); ok {
		cfg.Concurrency.ActionWorkers = intValue(v, cfg.Concurrency.ActionWorkers)
	}
	if v, ok := getEnv("DEDUP_LOG_LEVEL"); ok {
		cfg.Logging.Level = v
	}
	if v, ok := getEnv("DEDUP_LOG_FORMAT"); ok {
		cfg.Logging.Format = v
	}
	if v, ok := getEnv("DEDUP_AUDIT_DIR"); ok {
		cfg.Audit.Dir = v
	}
	if v, ok := getEnv("DEDUP_STRICT"); ok {
		cfg.Strict = boolValue(v, cfg.Strict)
	}
	if v, ok := getEnv("DEDUP_FOLLOW_SYMLINKS"); ok {
		cfg.FollowSymlinks = boolValue(v, cfg.FollowSymlinks)
	}
	if v, ok := getEnv("DEDUP_CANONICAL_POLICY"); ok {
		cfg.Canonical.Policy = Policy(v)
	}
	if v, ok := getEnv("DEDUP_CANONICAL_FOLDER"); ok {
		cfg.Canonical.CanonicalFolder = v
	}
	return cfg
}

func getEnv(k string) (string, bool) {
	v, ok := os.LookupEnv(k)
	if !ok {
		return "", false
	}
	v = strings.TrimSpace(v)
	if v == "" {
		return "", false
	}
	return v, true
}

func boolValue(s string, def bool) bool {
	if b, err := strconv.ParseBool(s); err == nil {
		return b
	}
	return def
}

func intValue(s string, def int) int {
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}

