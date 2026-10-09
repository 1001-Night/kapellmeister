package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"gitlab.com/1001n/kapellmeister/config"
	"gitlab.com/1001n/kapellmeister/helm"
	"gitlab.com/1001n/kapellmeister/ui"
)

func newUpdateCmd(opts *config.Options, reporter *ui.Reporter) *cobra.Command {
	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Обновить приложение",
		Args:  cobra.NoArgs,

		RunE: func(cmd *cobra.Command, args []string) error {
			reporter.Start("Updating application")

			values, err := loadValues(opts.ValuesFile)
			if err != nil {
				return err
			}

			client, err := helm.New(*opts, reporter)
			if err != nil {
				return err
			}

			rel, err := client.Update(cmd.Context(), values)
			if err != nil {
				return err
			}

			reporter.Success(fmt.Sprintf(
				"Updated release=%s revision=%d",
				rel.Name,
				rel.Version,
			))
			return verifyAfterOperation(cmd.Context(), opts, client, rel, reporter)
		},
	}

	updateCmd.Flags().StringVarP(
		&opts.ValuesFile,
		"values",
		"f",
		"",
		"путь к файлу параметров values.yaml",
	)
	updateCmd.Flags().StringVar(
		&opts.Version,
		"version",
		opts.Version,
		"версия Helm chart",
	)

	return updateCmd
}
