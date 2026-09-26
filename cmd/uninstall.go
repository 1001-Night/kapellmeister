package cmd

import (
	"github.com/spf13/cobra"
	"gitlab.com/1001n/kapellmeister/config"
	"gitlab.com/1001n/kapellmeister/helm"
	"gitlab.com/1001n/kapellmeister/ui"
)

func newUninstallCmd(opts *config.Options, reporter *ui.Reporter) *cobra.Command {

	uninstallCmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Удалить приложение",
		Args:  cobra.NoArgs,

		RunE: func(cmd *cobra.Command, args []string) error {
			reporter.Start("Uninstalling application")

			client, err := helm.New(*opts, reporter)
			if err != nil {
				return err
			}

			err = client.Uninstall(cmd.Context())
			if err != nil {
				return err
			}
			reporter.Success("Application uninstalled")
			return nil
		},
	}
	return uninstallCmd
}
