package cli

import (
	"github.com/spf13/cobra"

	"deduplicator/internal/model"
)

func newReportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "report <roots...>",
		Short: "Report duplicates without touching the filesystem",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPipeline(cmd, args, model.ModeReport, true, nil)
		},
		Example: `  dedup report ./photos ./backup
  dedup report ./photos --format=json --out-report=./report.json`,
	}
	return cmd
}
