package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/release"

	"gitlab.com/1001n/kapellmeister/config"
	"gitlab.com/1001n/kapellmeister/helm"
)

type verificationReport struct {
	Time      time.Time          `json:"time"`
	Release   string             `json:"release"`
	Namespace string             `json:"namespace"`
	Revision  int                `json:"revision"`
	Chart     string             `json:"chart,omitempty"`
	Status    string             `json:"status"`
	Result    string             `json:"result"`
	Passed    int                `json:"passed"`
	Warnings  int                `json:"warnings"`
	Failed    int                `json:"failed"`
	Checks    []helm.CheckResult `json:"checks"`
	Error     string             `json:"error,omitempty"`
}

func addReportFileFlag(cmd *cobra.Command, opts *config.Options) {
	cmd.Flags().StringVar(
		&opts.ReportFile,
		"report-file",
		"",
		"сохранить результат проверки в JSON (заменяет файл)",
	)
}

// finishReport возвращает итог проверки и сохраняет отчёт, если указан файл.
func finishReport(
	opts *config.Options,
	report verificationReport,
	resultErr error,
) error {
	if resultErr == nil && report.Failed > 0 {
		resultErr = fmt.Errorf("verification failed: %d checks failed, %d warnings", report.Failed, report.Warnings)
	}
	if resultErr != nil {
		report.Result = "failed"
		report.Error = resultErr.Error()
	}
	if opts.ReportFile == "" {
		return resultErr
	}
	if err := writeVerificationReport(opts.ReportFile, report); err != nil {
		return errors.Join(resultErr, fmt.Errorf("save report: %w", err))
	}
	return resultErr
}

// newVerificationReport собирает общий результат для консоли и JSON.
func newVerificationReport(
	opts config.Options,
	rel *release.Release,
	checks []helm.CheckResult,
) verificationReport {
	report := verificationReport{
		Time:      time.Now().UTC(),
		Release:   opts.Release,
		Namespace: opts.Namespace,
		Status:    "unknown",
		Result:    "passed",
		Checks:    []helm.CheckResult{},
	}
	if rel != nil {
		report.Release = rel.Name
		report.Namespace = rel.Namespace
		report.Revision = rel.Version
		if rel.Info != nil {
			report.Status = string(rel.Info.Status)
			status := helm.CheckPassed
			if rel.Info.Status != release.StatusDeployed {
				status = helm.CheckFailed
			}
			report.Checks = append(report.Checks, helm.CheckResult{
				Section: "Release",
				Name:    "Status",
				Detail:  report.Status,
				Status:  status,
			})
		}
		if rel.Chart != nil && rel.Chart.Metadata != nil {
			report.Chart = fmt.Sprintf("%s %s", rel.Chart.Metadata.Name, rel.Chart.Metadata.Version)
		}
	}
	report.Checks = append(report.Checks, checks...)
	for _, check := range report.Checks {
		switch check.Status {
		case helm.CheckPassed:
			report.Passed++
		case helm.CheckWarning:
			report.Warnings++
		default:
			report.Failed++
		}
	}
	if report.Failed > 0 {
		report.Result = "failed"
	}
	return report
}

func writeVerificationReport(path string, report verificationReport) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("encode report: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write %q: %w", path, err)
	}
	return nil
}
