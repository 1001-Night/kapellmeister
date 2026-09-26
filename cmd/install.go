package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"gitlab.com/1001n/kapellmeister/config"
	"gitlab.com/1001n/kapellmeister/helm"
	"gitlab.com/1001n/kapellmeister/ui"
)

func newInstallCmd(opts *config.Options, reporter *ui.Reporter) *cobra.Command {

	return &cobra.Command{
		Use:   "install",
		Short: "Установить приложение",
		Args:  cobra.NoArgs,

		RunE: func(cmd *cobra.Command, args []string) error {
			reporter.Start("Installing application")

			client, err := helm.New(*opts, reporter)
			if err != nil {
				return err
			}

			values := map[string]interface{}{"replicaCount": 2}

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
}
