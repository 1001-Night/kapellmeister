package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"gitlab.com/1001n/kapellmeister/config"
	"gitlab.com/1001n/kapellmeister/helm"
	"gitlab.com/1001n/kapellmeister/ui"
)

func newVerifyCmd(
	opts *config.Options,
	reporter *ui.Reporter,
) *cobra.Command {
	return &cobra.Command{
		Use:   "verify",
		Short: "Проверить состояние приложения",
		Args:  cobra.NoArgs,

		RunE: func(cmd *cobra.Command, args []string) error {
			reporter.Start("Verifying application")

			client, err := helm.New(*opts, reporter)
			if err != nil {
				return err
			}

			rel, err := client.VerifyRelease()
			if err != nil {
				return err
			}

			resourceCount, err := client.VerifyResources(rel)
			if err != nil {
				return err
			}

			reporter.Success(fmt.Sprintf(
				"Release=%s status=%s revision=%d resources=%d",
				rel.Name,
				rel.Info.Status,
				rel.Version,
				resourceCount,
			))

			return nil
		},
	}
}