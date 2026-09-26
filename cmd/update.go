package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"gitlab.com/1001n/kapellmeister/config"
	"gitlab.com/1001n/kapellmeister/helm"
)

func newUpdateCmd(opts *config.Options) *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Обновить приложение",
		Args:  cobra.NoArgs,

		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := helm.New(*opts)
			if err != nil {
				return err
			}

			values := map[string]interface{}{"replicaCount": 2}

			rel, err := client.Update(cmd.Context(), values)
			if err != nil {
				return err
			}

			fmt.Printf("updated release=%s revision=%d\n", rel.Name, rel.Version)
			return nil
		},
	}
}
