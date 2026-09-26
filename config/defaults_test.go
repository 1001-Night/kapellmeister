package config

import (
	"testing"
	"time"
)

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if opts.Namespace != "test-ngix" {
		t.Errorf("Namespace = %q, want %q", opts.Namespace, "test-ngix")
	}
	if opts.Release != "test-ngix" {
		t.Errorf("Release = %q, want %q", opts.Release, "test-ngix")
	}
	if opts.Version != "0.0.3" {
		t.Errorf("Version = %q, want %q", opts.Version, "0.0.3")
	}
	if opts.Timeout != 5*time.Minute {
		t.Errorf("Timeout = %v, want %v", opts.Timeout, 5*time.Minute)
	}
	if opts.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q", opts.LogLevel, "info")
	}
	if opts.Output != "text" {
		t.Errorf("Output = %q, want %q", opts.Output, "text")
	}
}
