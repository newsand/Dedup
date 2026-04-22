package cli

import (
	"github.com/spf13/cobra"

	"deduplicator/internal/config"
	"deduplicator/internal/model"
)

func newMoveDuplicatesCmd() *cobra.Command {
	var (
		dest   string
		dryRun bool
	)
	cmd := &cobra.Command{
		Use:   "move-duplicates <roots...>",
		Short: "Move duplicates to a quarantine directory (canonical stays in place)",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPipeline(cmd, args, model.ModeMoveDuplicates, dryRun, func(cfg *config.Config) error {
				if dest != "" {
					cfg.Destination.Dir = dest
				}
				return nil
			})
		},
		Example: `  dedup move-duplicates ./photos --dest ./quarantine --dry-run=false`,
	}
	cmd.Flags().StringVar(&dest, "dest", "", "destination directory for duplicates (default: <root>/_duplicates)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", true, "when true, only plan; do not move files")
	return cmd
}
