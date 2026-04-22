// Package config defines the configuration schema used by the CLI and the
// pipeline. Config values can come from YAML files, DEDUP_* environment
// variables, and command-line flags, with the later overriding the former.
package config

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"deduplicator/internal/model"
)

// Policy selects how the canonical file is chosen within a DuplicateGroup.
// v1.0 only ships KeepOldest; the other values are reserved for v2.0.
type Policy string

const (
	KeepOldest            Policy = "keep-oldest"
	KeepMostRecent        Policy = "keep-most-recent"
	KeepLargest           Policy = "keep-largest"
	KeepInCanonicalFolder Policy = "keep-in-canonical-folder"
)

type OutputCfg struct {
	Dir                string `yaml:"dir"`
	PreserveTimestamps bool   `yaml:"preserve_timestamps"`
}

type DestinationCfg struct {
	Dir string `yaml:"dir"`
}

type ConcurrencyCfg struct {
	Workers       int `yaml:"workers"`
	ActionWorkers int `yaml:"action_workers"`
}

type FiltersCfg struct {
	Include []string `yaml:"include"`
	Exclude []string `yaml:"exclude"`
	Ext     []string `yaml:"ext"`
}

type CanonicalCfg struct {
	Policy          Policy `yaml:"policy"`
	CanonicalFolder string `yaml:"canonical_folder"`
}

type LoggingCfg struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

type AuditCfg struct {
	Dir string `yaml:"dir"`
}

// Config is the top-level configuration struct.
type Config struct {
	Roots          []string       `yaml:"roots"`
	Mode           model.Mode     `yaml:"mode"`
	DryRun         bool           `yaml:"dry_run"`
	SuppressName   bool           `yaml:"suppress_name"`
	Output         OutputCfg      `yaml:"output"`
	Destination    DestinationCfg `yaml:"destination"`
	Concurrency    ConcurrencyCfg `yaml:"concurrency"`
	Filters        FiltersCfg     `yaml:"filters"`
	Canonical      CanonicalCfg   `yaml:"canonical"`
	Logging        LoggingCfg     `yaml:"logging"`
	Audit          AuditCfg       `yaml:"audit"`
	Strict         bool           `yaml:"strict"`
	FollowSymlinks bool           `yaml:"follow_symlinks"`
}

// Default returns a fully populated Config with conservative defaults —
// notably DryRun=true, Mode=report and KeepOldest policy.
func Default() Config {
	return Config{
		Mode:         model.ModeReport,
		DryRun:       true,
		SuppressName: false,
		Output: OutputCfg{
			PreserveTimestamps: true,
		},
		Filters: FiltersCfg{
			Exclude: []string{"**/.git/**"},
			Ext:     []string{".png", ".jpg", ".jpeg", ".webp", ".gif", ".pdf"},
		},
		Canonical: CanonicalCfg{Policy: KeepOldest},
		Logging:   LoggingCfg{Level: "info", Format: "text"},
		Audit:     AuditCfg{Dir: "./.dedup-audit"},
	}
}

// Validate checks the required invariants. Every check maps to a documented
// rule in Docs/11-design-config.md § 11.5.
func (c *Config) Validate() error {
	if len(c.Roots) == 0 {
		return errors.New("config: at least one root is required")
	}
	switch c.Mode {
	case model.ModeReport, model.ModeCopyUnique, model.ModeMoveDuplicates, model.ModeDeleteDuplicates:
	default:
		return fmt.Errorf("config: unknown mode %q", c.Mode)
	}

	if c.Mode == model.ModeCopyUnique {
		if strings.TrimSpace(c.Output.Dir) == "" {
			return errors.New("config: output.dir is required for copy-unique")
		}
		outAbs, err := filepath.Abs(c.Output.Dir)
		if err != nil {
			return err
		}
		for _, r := range c.Roots {
			rAbs, err := filepath.Abs(r)
			if err != nil {
				return err
			}
			if pathContains(rAbs, outAbs) {
				return fmt.Errorf("config: output.dir %q must not be inside root %q", outAbs, rAbs)
			}
		}
	}

	switch c.Canonical.Policy {
	case KeepOldest, KeepMostRecent, KeepLargest, KeepInCanonicalFolder, "":
	default:
		return fmt.Errorf("config: unknown canonical.policy %q", c.Canonical.Policy)
	}
	if c.Canonical.Policy == KeepInCanonicalFolder && strings.TrimSpace(c.Canonical.CanonicalFolder) == "" {
		return errors.New("config: canonical.canonical_folder is required for keep-in-canonical-folder")
	}

	if c.Concurrency.Workers < 0 || c.Concurrency.ActionWorkers < 0 {
		return errors.New("config: concurrency.workers and action_workers must be >= 0")
	}

	for _, e := range c.Filters.Ext {
		if e == "" || !strings.HasPrefix(e, ".") {
			return fmt.Errorf("config: filters.ext entry %q must start with '.'", e)
		}
	}

	return nil
}

// pathContains reports whether child is the same as, or nested under, parent,
// after case-normalisation on Windows.
func pathContains(parent, child string) bool {
	parent = filepath.Clean(parent)
	child = filepath.Clean(child)
	if runtime.GOOS == "windows" {
		parent = strings.ToLower(parent)
		child = strings.ToLower(child)
	}
	if parent == child {
		return true
	}
	sep := string(filepath.Separator)
	if !strings.HasSuffix(parent, sep) {
		parent += sep
	}
	return strings.HasPrefix(child, parent)
}
