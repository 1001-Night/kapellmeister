package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitlab.com/1001n/kapellmeister/config"
)

func TestPrepareOpenStackValidation(t *testing.T) {
	tests := []struct {
		name string
		opts config.Options
		want string
	}{
		{name: "normal kubeconfig", opts: config.Options{Kubeconfig: "existing"}},
		{name: "cloud without OpenStack", opts: config.Options{Cloud: "lab"}, want: "require --openstack"},
		{name: "file without OpenStack", opts: config.Options{CloudsFile: "clouds.yaml"}, want: "require --openstack"},
		{name: "conflicting sources", opts: config.Options{OpenStackCluster: "lab", Kubeconfig: "existing"}, want: "cannot be used together"},
		{name: "blank cluster", opts: config.Options{OpenStackCluster: "  "}, want: "cluster name or UUID"},
		{name: "missing file", opts: config.Options{OpenStackCluster: "lab", CloudsFile: filepath.Join(t.TempDir(), "missing.yaml")}, want: "read clouds.yaml"},
		{name: "directory instead of file", opts: config.Options{OpenStackCluster: "lab", CloudsFile: t.TempDir()}, want: "is a directory"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := prepareOpenStack(context.Background(), &tt.opts)
			if tt.want == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestPrepareOpenStackRestoresConfigEnvironment(t *testing.T) {
	t.Setenv("OS_CLOUD", "")
	t.Setenv("OS_AUTH_URL", "")
	t.Setenv("OS_CLIENT_CONFIG_FILE", "original-clouds.yaml")
	path := filepath.Join(t.TempDir(), "clouds.yaml")
	if err := os.WriteFile(path, []byte("clouds:\n  first: {}\n  second: {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	opts := config.DefaultOptions()
	opts.OpenStackCluster = "lab"
	opts.CloudsFile = path
	err := prepareOpenStack(context.Background(), &opts)
	if err == nil || !strings.Contains(err.Error(), "select a cloud") {
		t.Fatalf("error = %v, want cloud selection error", err)
	}
	if got := os.Getenv("OS_CLIENT_CONFIG_FILE"); got != "original-clouds.yaml" {
		t.Fatalf("OS_CLIENT_CONFIG_FILE = %q, want original value", got)
	}
	if len(opts.KubeconfigData) != 0 {
		t.Fatal("failed authentication must not replace kubeconfig")
	}
}
