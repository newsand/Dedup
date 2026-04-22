// Package cli wires Cobra subcommands around the pipeline. It is the only
// place in the codebase that talks about flags, exit codes, and progress.
package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"deduplicator/internal/config"
	"deduplicator/internal/model"
)

// Version info injected via -ldflags at build time.
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

// Exit codes documented in Docs/10-design-cli.md § 10.6.
const (
	ExitOK              = 0
	ExitGenericError    = 1
	ExitUsageError      = 2
	ExitNoRoots         = 3
	ExitRootNotExist    = 4
	ExitStrictAborted   = 5
	ExitDeleteNeedsYes  = 6
	ExitInvariantFailed = 7
)

// globalFlags is the aggregated state of the persistent flags; populated by
// Cobra PreRun hooks.
type globalFlags struct {
	configPath     string
	logLevel       string
	logFormat      string
	workers        int
	actionWorkers  int
	include        []string
	exclude        []string
	exts           []string
	followSymlinks bool
	strict         bool
	auditDir       string
	format         string
	outReport      string
}

var gFlags globalFlags

var rootCmd = &cobra.Command{
	Use:   "dedup",
	Short: "Exact file deduplicator (BLAKE3)",
	Long: `dedup finds and handles byte-exact duplicates in directories.

v1.0 supports images and PDFs. Duplicity is defined only by equal content
(BLAKE3). No near-duplicate, no content transformation.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command and translates errors into the documented
// exit codes.
func Execute() {
	ctx, cancel := signalContext()
	defer cancel()

	err := rootCmd.ExecuteContext(ctx)
	if err == nil {
		os.Exit(ExitOK)
	}
	var cliErr *Error
	if errors.As(err, &cliErr) {
		fmt.Fprintln(os.Stderr, "error:", cliErr.Err)
		os.Exit(cliErr.Code)
	}
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(ExitGenericError)
}

// Error pairs an exit code with an underlying error.
type Error struct {
	Code int
	Err  error
}

func (e *Error) Error() string { return e.Err.Error() }
func (e *Error) Unwrap() error { return e.Err }

func newCLIError(code int, err error) error {
	if err == nil {
		return nil
	}
	return &Error{Code: code, Err: err}
}

func init() {
	pf := rootCmd.PersistentFlags()
	pf.StringVar(&gFlags.configPath, "config", "", "path to YAML config file")
	pf.StringVar(&gFlags.logLevel, "log-level", "info", "log level (debug|info|warn|error)")
	pf.StringVar(&gFlags.logFormat, "log-format", "text", "log format (text|json)")
	pf.IntVar(&gFlags.workers, "workers", 0, "hashing workers (0 = NumCPU)")
	pf.IntVar(&gFlags.actionWorkers, "action-workers", 0, "action workers (0 = max(2, NumCPU/2))")
	pf.StringArrayVar(&gFlags.include, "include", nil, "glob include (repeatable)")
	pf.StringArrayVar(&gFlags.exclude, "exclude", nil, "glob exclude (repeatable)")
	pf.StringSliceVar(&gFlags.exts, "ext", nil, "extensions to consider (default: images + pdf)")
	pf.BoolVar(&gFlags.followSymlinks, "follow-symlinks", false, "follow symlinks when walking")
	pf.BoolVar(&gFlags.strict, "strict", false, "abort on first error")
	pf.StringVar(&gFlags.auditDir, "audit-dir", "", "where to write the JSONL audit log")
	pf.StringVar(&gFlags.format, "format", "text", "report format (text|json)")
	pf.StringVar(&gFlags.outReport, "out-report", "", "also write the report to this file")

	rootCmd.AddCommand(
		newScanCmd(),
		newReportCmd(),
		newCopyUniqueCmd(),
		newMoveDuplicatesCmd(),
		newDeleteDuplicatesCmd(),
		newVersionCmd(),
	)
}

// loadConfig returns a Config that layers YAML + env + flags. `roots` are
// the positional args given to the subcommand.
func loadConfig(roots []string, mode model.Mode) (config.Config, error) {
	var (
		cfg  config.Config
		path string
		err  error
	)
	if gFlags.configPath != "" {
		cfg, err = config.Load(gFlags.configPath)
		if err != nil {
			return config.Config{}, err
		}
		path = gFlags.configPath
	} else {
		cfg, path, err = config.LoadAuto()
		if err != nil {
			return config.Config{}, err
		}
	}
	cfg = config.ApplyEnv(cfg)

	// Flags override env + YAML.
	if len(roots) > 0 {
		cfg.Roots = roots
	}
	cfg.Mode = mode
	if gFlags.workers != 0 {
		cfg.Concurrency.Workers = gFlags.workers
	}
	if gFlags.actionWorkers != 0 {
		cfg.Concurrency.ActionWorkers = gFlags.actionWorkers
	}
	if len(gFlags.include) > 0 {
		cfg.Filters.Include = gFlags.include
	}
	if len(gFlags.exclude) > 0 {
		cfg.Filters.Exclude = append(cfg.Filters.Exclude, gFlags.exclude...)
	}
	if len(gFlags.exts) > 0 {
		cfg.Filters.Ext = gFlags.exts
	}
	if gFlags.followSymlinks {
		cfg.FollowSymlinks = true
	}
	if gFlags.strict {
		cfg.Strict = true
	}
	if gFlags.logLevel != "" {
		cfg.Logging.Level = gFlags.logLevel
	}
	if gFlags.logFormat != "" {
		cfg.Logging.Format = gFlags.logFormat
	}
	if gFlags.auditDir != "" {
		cfg.Audit.Dir = gFlags.auditDir
	}

	_ = path
	return cfg, nil
}

// validateRoots verifies every root exists on disk.
func validateRoots(roots []string) error {
	for _, r := range roots {
		if _, err := os.Stat(r); err != nil {
			if os.IsNotExist(err) {
				return newCLIError(ExitRootNotExist, fmt.Errorf("root does not exist: %s", r))
			}
			return newCLIError(ExitGenericError, err)
		}
	}
	return nil
}

func signalContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ch
		cancel()
	}()
	return ctx, func() {
		signal.Stop(ch)
		cancel()
	}
}
