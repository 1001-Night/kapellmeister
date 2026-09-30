package config

import "time"

const ChartRef = "oci://registry.gitlab.com/1001n/kapellmeister-test-chart/charts/kapellmeister"

type Options struct {
	Kubeconfig string
	Context    string
	Namespace  string
	Release    string
	Version    string
	ValuesFile string
	Timeout    time.Duration
	Verbose    bool
	LogLevel   string
	Output     string
	ReportFile string
}

func DefaultOptions() Options {
	return Options{
		Namespace: "kapellmeister",
		Release:   "kapellmeister",
		Version:   "0.1.0",
		Timeout:   5 * time.Minute,
		LogLevel:  "info",
		Output:    "text",
	}
}
