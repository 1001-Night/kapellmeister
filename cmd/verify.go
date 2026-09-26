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

			checks, err := client.VerifyResources(
				cmd.Context(),
				rel,
			)
			if err != nil {
				return err
			}

			reporter.Section("Release")
			reporter.Field("Name", rel.Name)
			reporter.Passed("Status", string(rel.Info.Status))
			reporter.Field("Revision", rel.Version)

			if rel.Chart != nil && rel.Chart.Metadata != nil {
				reporter.Field(
					"Chart",
					fmt.Sprintf(
						"%s %s",
						rel.Chart.Metadata.Name,
						rel.Chart.Metadata.Version,
					),
				)
			}

			reporter.Section("Resources")

			failed := 0

			for _, check := range checks {
				if check.Passed {
					reporter.Passed(check.Name, check.Detail)
					continue
				}

				reporter.Failed(check.Name, check.Detail)
				failed++
			}

			if failed > 0 {
				return fmt.Errorf(
					"verification failed: %d of %d resource checks failed",
					failed,
					len(checks),
				)
			}

			reporter.Complete(fmt.Sprintf(
				"Verification passed: %d resources checked",
				len(checks),
			))

			return nil
		},
	}
}
