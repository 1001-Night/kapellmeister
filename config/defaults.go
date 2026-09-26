package config

import "time"

const ChartRef = "oci://registry.gitlab.com/1001n/test-chart/charts/test-ngix"

type Options struct {
	Kubeconfig string
	Context    string
	Namespace  string
	Release    string
	Version    string
	Timeout    time.Duration
	Verbose    bool
	LogLevel   string
	Output     string
	ReportFile string
}

func DefaultOptions() Options {
	return Options{
		Namespace: "test-ngix",
		Release:   "test-ngix",
		Version:   "0.0.3",
		Timeout:   5 * time.Minute,
		LogLevel:  "info",
		Output:    "text",
	}
}
