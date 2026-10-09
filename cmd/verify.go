package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/release"

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

// verifyAfterOperation показывает результат проверки после изменения релиза.
func verifyAfterOperation(
	ctx context.Context,
	opts *config.Options,
	client *helm.Client,
	rel *release.Release,
	reporter *ui.Reporter,
) error {
	reporter.Start("Verifying application")
	if rel.Info == nil || rel.Info.Status != release.StatusDeployed {
		return fmt.Errorf("release %q is not deployed after operation", rel.Name)
	}

	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()
	checks, err := client.Verify(ctx, rel)
	if err != nil {
		return fmt.Errorf("verify release after operation: %w", err)
	}

	failed, warnings := 0, 0
	showingIssues := false
	for _, check := range checks {
		if check.Status != helm.CheckPassed && !showingIssues {
			reporter.Section("Verification")
			showingIssues = true
		}
		switch check.Status {
		case helm.CheckFailed:
			reporter.Failed(check.Name, check.Detail)
			failed++
		case helm.CheckWarning:
			reporter.Warning(check.Name, check.Detail)
			warnings++
		}
	}
	if failed > 0 {
		return fmt.Errorf("verification failed: %d checks failed, %d warnings", failed, warnings)
	}

	reporter.Success(fmt.Sprintf("Verified %d checks, %d warnings", len(checks), warnings))
	return nil
}
