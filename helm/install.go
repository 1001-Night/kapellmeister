package helm

import (
	"context"
	"errors"
	"fmt"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/release"

	"gitlab.com/1001n/kapellmeister/config"
)

func (c *Client) Install(
	ctx context.Context,
	values map[string]interface{},
) (*release.Release, error) {
	install := c.newInstallAction()

	chartPath, err := install.LocateChart(config.ChartRef, c.Settings)
	if err != nil {
		return nil, fmt.Errorf("locate chart: %w", err)
	}

	ch, err := loader.Load(chartPath)
	if err != nil {
		return nil, fmt.Errorf("load chart: %w", err)
	}
	if err := c.preflightKubernetesVersion(ch); err != nil {
		return nil, err
	}

	manifest, err := c.RenderInstall(ctx, ch, values)
	if err != nil {
		return nil, err
	}
	if err := preflightImageRegistries(ctx, manifest); err != nil {
		return nil, err
	}

	if err := c.CheckPermissions(
		ctx,
		OperationInstall,
		manifest,
		"",
	); err != nil {
		return nil, err
	}

	namespaceCreated, err := c.ensureNamespace(ctx)
	if err != nil {
		return nil, err
	}

	rel, err := install.RunWithContext(ctx, ch, values)
	if err != nil {
		if namespaceCreated {
			if cleanupErr := c.deleteOwnedNamespace(ctx); cleanupErr != nil {
				return nil, errors.Join(
					fmt.Errorf("install release: %w", err),
					fmt.Errorf("cleanup namespace: %w", cleanupErr),
				)
			}
		}
		return nil, fmt.Errorf("install release: %w", err)
	}

	return rel, nil
}

func (c *Client) newInstallAction() *action.Install {
	install := action.NewInstall(c.ActionConfig)
	install.SetRegistryClient(c.Registry)

	install.ReleaseName = c.Options.Release
	install.Namespace = c.Options.Namespace
	install.Version = c.Options.Version
	install.CreateNamespace = false

	install.Wait = true
	install.WaitForJobs = true
	install.Atomic = true
	install.Timeout = c.Options.Timeout

	return install
}
