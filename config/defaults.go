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
	ReportFile string
}

func DefaultOptions() Options {
	return Options{
		Namespace: "kapellmeister",
		Release:   "kapellmeister",
		Version:   "0.6.0",
		Timeout:   5 * time.Minute,
	}
}
