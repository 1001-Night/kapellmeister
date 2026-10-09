package cmd

import (
	"testing"

	"helm.sh/helm/v3/pkg/release"
)

func TestReleaseStatus(t *testing.T) {
	tests := []struct {
		name string
		info *release.Info
		want string
	}{
		{name: "missing info", want: "unknown"},
		{name: "deployed", info: &release.Info{Status: release.StatusDeployed}, want: "deployed"},
		{name: "failed", info: &release.Info{Status: release.StatusFailed}, want: "failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := releaseStatus(&release.Release{Info: tt.info})
			if got != tt.want {
				t.Errorf("releaseStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}
