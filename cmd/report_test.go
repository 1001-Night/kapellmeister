package cmd

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"

	"gitlab.com/1001n/kapellmeister/config"
	"gitlab.com/1001n/kapellmeister/helm"
)

func TestWriteVerificationReport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "report.json")
	rel := &release.Release{
		Name:      "kapellmeister",
		Namespace: "kapellmeister",
		Version:   2,
		Info:      &release.Info{Status: release.StatusDeployed},
		Chart:     &chart.Chart{Metadata: &chart.Metadata{Name: "kapellmeister", Version: "0.6.0"}},
		Manifest:  "secret-not-for-report",
		Config:    map[string]interface{}{"password": "secret-not-for-report"},
	}
	checks := []helm.CheckResult{
		{Section: "Cluster", Name: "Kubernetes", Detail: "available", Status: helm.CheckPassed},
		{Section: "Events", Name: "Warning", Detail: "restart", Status: helm.CheckWarning},
		{Section: "Resources", Name: "Service", Detail: "missing", Status: helm.CheckFailed},
	}
	if err := writeVerificationReport(path, config.DefaultOptions(), rel, checks, errors.New("verification failed")); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "secret-not-for-report") {
		t.Fatal("report contains release manifest or values")
	}
	var report verificationReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if report.Release != rel.Name || report.Revision != 2 || report.Chart != "kapellmeister 0.6.0" {
		t.Fatalf("release details = %#v", report)
	}
	if report.Result != "failed" || report.Passed != 2 || report.Warnings != 1 || report.Failed != 1 || report.Error != "verification failed" {
		t.Fatalf("report summary = %#v", report)
	}
	if len(report.Checks) != 4 || report.Checks[0].Name != "Status" || report.Checks[3].Name != "Service" {
		t.Fatalf("report checks = %#v", report.Checks)
	}

	if err := writeVerificationReport(path, config.DefaultOptions(), rel, checks[:1], nil); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	report = verificationReport{}
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if report.Result != "passed" || report.Error != "" || report.Passed != 2 || len(report.Checks) != 2 {
		t.Fatalf("updated report = %#v", report)
	}
}

func TestWriteVerificationReportWithoutRelease(t *testing.T) {
	path := filepath.Join(t.TempDir(), "report.json")
	opts := config.DefaultOptions()
	if err := writeVerificationReport(path, opts, nil, nil, errors.New("release not found")); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var report verificationReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if report.Release != opts.Release || report.Status != "unknown" || report.Result != "failed" || report.Error != "release not found" {
		t.Fatalf("report = %#v", report)
	}
	if report.Checks == nil {
		t.Fatal("checks must be an empty JSON array")
	}
}
