package cmd

import (
	"context"
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

			ctx, cancel := context.WithTimeout(
				cmd.Context(),
				opts.Timeout,
			)
			defer cancel()

			client, err := helm.New(*opts, reporter)
			if err != nil {
				return err
			}

			rel, err := client.VerifyRelease()
			if err != nil {
				return err
			}

			checks, err := client.Verify(
				ctx,
				rel,
			)
			if err != nil {
				return err
			}

			passed := 0
			warnings := 0
			failed := 0

			reporter.Section("Release")
			reporter.Field("Name", rel.Name)
			reporter.Passed("Status", string(rel.Info.Status))
			passed++
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

			currentSection := ""

			for _, check := range checks {
				if check.Section != currentSection {
					reporter.Section(check.Section)
					currentSection = check.Section
				}

				switch check.Status {
				case helm.CheckPassed:
					reporter.Passed(check.Name, check.Detail)
					passed++
				case helm.CheckWarning:
					reporter.Warning(check.Name, check.Detail)
					warnings++
				default:
					reporter.Failed(check.Name, check.Detail)
					failed++
				}
			}

			if failed > 0 {
				return fmt.Errorf(
					"verification failed: %d checks failed, %d warnings",
					failed,
					warnings,
				)
			}

			reporter.Complete(fmt.Sprintf(
				"Verification passed: %d checks, %d warnings",
				passed,
				warnings,
			))

			return nil
		},
	}
}
