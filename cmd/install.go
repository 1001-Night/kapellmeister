package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"gitlab.com/1001n/kapellmeister/config"
	"gitlab.com/1001n/kapellmeister/helm"
	"gitlab.com/1001n/kapellmeister/ui"
)

func newInstallCmd(opts *config.Options, reporter *ui.Reporter) *cobra.Command {
	installCmd := &cobra.Command{
		Use:   "install",
		Short: "Установить приложение",
		Args:  cobra.NoArgs,

		RunE: func(cmd *cobra.Command, args []string) error {
			reporter.Start("Installing application")

			values, err := loadValues(opts.ValuesFile)
			if err != nil {
				return err
			}

			client, err := helm.New(*opts, reporter)
			if err != nil {
				return err
			}

			rel, err := client.Install(cmd.Context(), values)
			if err != nil {
				return err
			}

			reporter.Success(fmt.Sprintf(
				"Installed release=%s revision=%d",
				rel.Name,
				rel.Version,
			))
			return nil
		},
	}

	installCmd.Flags().StringVarP(
		&opts.ValuesFile,
		"values",
		"f",
		"",
		"путь к файлу параметров values.yaml",
	)
	installCmd.Flags().StringVar(
		&opts.Version,
		"version",
		opts.Version,
		"версия Helm chart",
	)

	return installCmd
}
