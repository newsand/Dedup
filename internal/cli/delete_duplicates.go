package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"deduplicator/internal/config"
	"deduplicator/internal/model"
)

func newDeleteDuplicatesCmd() *cobra.Command {
	var (
		yes    bool
		dryRun bool
	)
	cmd := &cobra.Command{
		Use:   "delete-duplicates <roots...>",
		Short: "Delete duplicates (canonical is kept). Requires --yes.",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !dryRun && !yes {
				return newCLIError(ExitDeleteNeedsYes, fmt.Errorf("refusing to run without --yes (delete is irreversible)"))
			}
			return runPipeline(cmd, args, model.ModeDeleteDuplicates, dryRun, func(cfg *config.Config) error { return nil })
		},
		Example: `  dedup delete-duplicates ./photos --yes --dry-run=false`,
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "explicit confirmation — required for real execution")
	cmd.Flags().BoolVar(&dryRun, "dry-run", true, "when true, only plan; do not delete files")
	return cmd
}
