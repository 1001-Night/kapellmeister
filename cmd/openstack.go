package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"gitlab.com/1001n/kapellmeister/config"
	"gitlab.com/1001n/kapellmeister/openstack"
)

// Получает kubeconfig один раз перед выполнением команды.
func prepareOpenStack(ctx context.Context, opts *config.Options) error {
	if opts.OpenStackCluster == "" {
		if opts.Cloud != "" || opts.CloudsFile != "" {
			return fmt.Errorf("--cloud and --clouds-yaml require --openstack")
		}
		return nil
	}
	if strings.TrimSpace(opts.OpenStackCluster) == "" {
		return fmt.Errorf("--openstack requires a cluster name or UUID")
	}
	if opts.Kubeconfig != "" {
		return fmt.Errorf("--openstack and --kubeconfig cannot be used together")
	}
	if opts.CloudsFile != "" {
		info, err := os.Stat(opts.CloudsFile)
		if err != nil {
			return fmt.Errorf("read clouds.yaml: %w", err)
		}
		if info.IsDir() {
			return fmt.Errorf("clouds.yaml path is a directory")
		}

		// Флаг переопределяет путь только на время обращения к OpenStack.
		previous, existed := os.LookupEnv("OS_CLIENT_CONFIG_FILE")
		if err := os.Setenv("OS_CLIENT_CONFIG_FILE", opts.CloudsFile); err != nil {
			return err
		}
		defer func() {
			if existed {
				_ = os.Setenv("OS_CLIENT_CONFIG_FILE", previous)
			} else {
				_ = os.Unsetenv("OS_CLIENT_CONFIG_FILE")
			}
		}()
	}

	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()
	client, err := openstack.NewMagnumClient(ctx, opts.Cloud, "", "")
	if err != nil {
		return fmt.Errorf("connect to OpenStack: %w", err)
	}
	data, err := openstack.Kubeconfig(ctx, client, opts.OpenStackCluster)
	if err != nil {
		return fmt.Errorf("get Magnum kubeconfig: %w", err)
	}
	opts.KubeconfigData = data
	return nil
}
