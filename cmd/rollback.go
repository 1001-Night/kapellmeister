package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"gitlab.com/1001n/kapellmeister/config"
	"gitlab.com/1001n/kapellmeister/helm"
	"gitlab.com/1001n/kapellmeister/ui"
)

func newRollbackCmd(
	opts *config.Options,
	reporter *ui.Reporter,
) *cobra.Command {
	var revision int

	rollbackCmd := &cobra.Command{
		Use:   "rollback",
		Short: "Откатить приложение",
		Args:  cobra.NoArgs,

		RunE: func(cmd *cobra.Command, args []string) error {
			reporter.Start("Rolling back application")

			client, err := helm.New(*opts, reporter)
			if err != nil {
				return err
			}

			rel, targetRevision, err := client.Rollback(
				cmd.Context(),
				revision,
			)
			if err != nil {
				return err
			}

			reporter.Success(fmt.Sprintf(
				"Rolled back release=%s target=%d revision=%d",
				rel.Name,
				targetRevision,
				rel.Version,
			))
			return verifyAfterOperation(cmd.Context(), opts, client, rel, reporter)
		},
	}

	rollbackCmd.Flags().IntVar(
		&revision,
		"revision",
		0,
		"ревизия Helm (0 — предыдущая)",
	)
	addReportFileFlag(rollbackCmd, opts)

	return rollbackCmd
}
