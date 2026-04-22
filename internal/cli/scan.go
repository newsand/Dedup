package cli

import (
	"github.com/spf13/cobra"

	"deduplicator/internal/model"
)

func newScanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scan <roots...>",
		Short: "Alias of 'report' (dry-run, human-readable output)",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPipeline(cmd, args, model.ModeReport, true, nil)
		},
	}
	return cmd
}
