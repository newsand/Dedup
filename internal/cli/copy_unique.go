package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"deduplicator/internal/config"
	"deduplicator/internal/model"
)

func newCopyUniqueCmd() *cobra.Command {
	var (
		outDir             string
		suppressName       bool
		preserveTimestamps bool
		dryRun             bool
	)
	cmd := &cobra.Command{
		Use:   "copy-unique <roots...>",
		Short: "Copy unique + canonical files to a flattened output directory",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if outDir == "" {
				return newCLIError(ExitUsageError, fmt.Errorf("--out is required for copy-unique"))
			}
			return runPipeline(cmd, args, model.ModeCopyUnique, dryRun, func(cfg *config.Config) error {
				cfg.Output.Dir = outDir
				cfg.SuppressName = suppressName
				cfg.Output.PreserveTimestamps = preserveTimestamps
				return nil
			})
		},
		Example: `  dedup copy-unique ./photos --out ./deduped --dry-run=false`,
	}
	cmd.Flags().StringVar(&outDir, "out", "", "flattened output directory (required)")
	cmd.Flags().BoolVar(&suppressName, "suppressname", false, "drop the original base name (see Docs/08)")
	cmd.Flags().BoolVar(&preserveTimestamps, "preserve-timestamps", true, "preserve mtime on copy")
	cmd.Flags().BoolVar(&dryRun, "dry-run", true, "when true, only plan; do not write files")
	return cmd
}
