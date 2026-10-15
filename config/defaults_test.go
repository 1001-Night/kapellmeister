package config

import (
	"testing"
	"time"

	"github.com/Masterminds/semver/v3"
)

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if opts.Namespace != "kapellmeister" {
		t.Errorf("Namespace = %q, want %q", opts.Namespace, "kapellmeister")
	}
	if opts.Release != "kapellmeister" {
		t.Errorf("Release = %q, want %q", opts.Release, "kapellmeister")
	}
	if opts.Version == "" {
		t.Error("Version must not be empty")
	} else if _, err := semver.NewVersion(opts.Version); err != nil {
		t.Errorf(
			"Version = %q, want valid semantic version: %v",
			opts.Version,
			err,
		)
	}
	if opts.Timeout != 5*time.Minute {
		t.Errorf("Timeout = %v, want %v", opts.Timeout, 5*time.Minute)
	}
}
