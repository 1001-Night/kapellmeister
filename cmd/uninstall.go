package cmd

import (
	"github.com/spf13/cobra"
	"gitlab.com/1001n/kapellmeister/config"
	"gitlab.com/1001n/kapellmeister/helm"
)

func newUninstallCmd(opts *config.Options) *cobra.Command {

	uninstallCmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Удалить приложение",
		Args:  cobra.NoArgs,

		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := helm.New(*opts)
			if err != nil {
				return err
			}

			err = client.Uninstall(cmd.Context())
			if err != nil {
				return err
			}
			return nil
		},
	}
	return uninstallCmd
}
