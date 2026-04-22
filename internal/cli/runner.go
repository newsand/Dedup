package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"deduplicator/internal/config"
	"deduplicator/internal/logging"
	"deduplicator/internal/model"
	"deduplicator/internal/pipeline"
	"deduplicator/internal/report"
)

// runPipeline is the shared backbone of every action subcommand. It builds the
// config, validates roots, invokes pipeline.Run, prints the report in the
// configured format and returns a cli.Error with the appropriate exit code.
func runPipeline(cmd *cobra.Command, roots []string, mode model.Mode, dryRun bool, overrides func(*config.Config) error) error {
	if len(roots) == 0 {
		return newCLIError(ExitNoRoots, fmt.Errorf("at least one root is required"))
	}

	cfg, err := loadConfig(roots, mode)
	if err != nil {
		return newCLIError(ExitUsageError, err)
	}
	cfg.DryRun = dryRun
	if overrides != nil {
		if err := overrides(&cfg); err != nil {
			return err
		}
	}
	if err := cfg.Validate(); err != nil {
		return newCLIError(ExitUsageError, err)
	}
	if err := validateRoots(cfg.Roots); err != nil {
		return err
	}

	logger := logging.New(logging.Options{
		Level:  cfg.Logging.Level,
		Format: cfg.Logging.Format,
		Out:    cmd.ErrOrStderr(),
	})

	out, err := pipeline.Run(cmd.Context(), pipeline.Input{
		Cfg: cfg, Version: Version, Logger: logger,
	})
	if err != nil {
		return newCLIError(ExitGenericError, err)
	}
	if cfg.Strict && len(out.Errors) > 0 {
		return newCLIError(ExitStrictAborted, fmt.Errorf("aborted with %d errors under --strict", len(out.Errors)))
	}

	return writeReport(cmd.OutOrStdout(), out)
}

func writeReport(stdout io.Writer, out pipeline.Output) error {
	data := report.BuildData(out.Job, out.Groups, out.Uniques, out.Selections,
		out.Mappings, out.Plan, out.Logs, out.AuditPath, out.Errors, out.Duration.Milliseconds())

	if gFlags.outReport != "" {
		f, err := os.Create(gFlags.outReport)
		if err != nil {
			return newCLIError(ExitGenericError, err)
		}
		defer f.Close()
		if err := writeFormat(f, data, gFlags.format); err != nil {
			return newCLIError(ExitGenericError, err)
		}
	}

	return writeFormat(stdout, data, gFlags.format)
}

func writeFormat(w io.Writer, data report.Data, format string) error {
	if format == "json" {
		return report.WriteJSON(w, data)
	}
	return report.WriteText(w, data)
}
