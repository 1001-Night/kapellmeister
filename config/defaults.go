package config

import "time"

type Options struct {
	Kubeconfig string
	Context    string
	Namespace  string
	Release    string
	Timeout    time.Duration
	LogLevel   string
	Output     string
	ReportFile string
}

func DefaultOptions() Options {
	return Options{
		Namespace: "kapellmeister-system",
		Release:   "kapellmeister",
		Timeout:   5 * time.Minute,
		LogLevel:  "info",
		Output:    "text",
	}
}