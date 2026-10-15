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
	verifyCmd := &cobra.Command{
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
				return finishReport(opts, newVerificationReport(*opts, nil, nil), err)
			}

			rel, err := client.VerifyRelease()
			if err != nil {
				return finishReport(opts, newVerificationReport(*opts, nil, nil), err)
			}

			checks, err := client.Verify(
				ctx,
				rel,
			)
			if err != nil {
				return finishReport(opts, newVerificationReport(*opts, rel, nil), err)
			}

			report := newVerificationReport(*opts, rel, checks)
			showVerification(reporter, report, true)
			if err := finishReport(opts, report, nil); err != nil {
				return err
			}

			reporter.Complete(fmt.Sprintf(
				"Verification passed: %d checks, %d warnings",
				report.Passed,
				report.Warnings,
			))

			return nil
		},
	}
	addReportFileFlag(verifyCmd, opts)
	return verifyCmd
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
		return finishReport(opts, newVerificationReport(*opts, rel, nil), fmt.Errorf("release %q is not deployed after operation", rel.Name))
	}

	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()
	checks, err := client.Verify(ctx, rel)
	if err != nil {
		return finishReport(opts, newVerificationReport(*opts, rel, nil), fmt.Errorf("verify release after operation: %w", err))
	}

	report := newVerificationReport(*opts, rel, checks)
	showVerification(reporter, report, false)
	if err := finishReport(opts, report, nil); err != nil {
		return err
	}

	reporter.Success(fmt.Sprintf("Verified %d checks, %d warnings", len(report.Checks), report.Warnings))
	return nil
}

// showVerification выводит все проверки или только замечания после операции.
func showVerification(reporter *ui.Reporter, report verificationReport, detailed bool) {
	currentSection := ""
	for _, check := range report.Checks {
		if !detailed && check.Status == helm.CheckPassed {
			continue
		}
		section := check.Section
		if !detailed {
			section = "Verification"
		}
		if section != currentSection {
			reporter.Section(section)
			currentSection = section
			if detailed && section == "Release" {
				reporter.Field("Name", report.Release)
			}
		}
		switch check.Status {
		case helm.CheckPassed:
			reporter.Passed(check.Name, check.Detail)
		case helm.CheckWarning:
			reporter.Warning(check.Name, check.Detail)
		default:
			reporter.Failed(check.Name, check.Detail)
		}
		if detailed && check.Section == "Release" {
			reporter.Field("Revision", report.Revision)
			if report.Chart != "" {
				reporter.Field("Chart", report.Chart)
			}
		}
	}
}
