package helm

import (
	"context"
	"fmt"
	"time"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/release"

	"gitlab.com/1001n/kapellmeister/config"
)

func (c *Client) Install(
	ctx context.Context,
	values map[string]interface{},
) (*release.Release, error) {

	err := c.CanI(
		ctx,
		"create",
		"apps",
		"deployments",
		c.Options.Namespace,
	)
	if err != nil {
		return nil, err
	}

	install := action.NewInstall(c.ActionConfig)
	install.SetRegistryClient(c.Registry)

	install.ReleaseName = c.Options.Release
	install.Namespace = c.Options.Namespace
	install.CreateNamespace = true
	install.Version = c.Options.Version

	install.Wait = true
	install.WaitForJobs = true
	install.Atomic = true
	install.Timeout = 5 * time.Minute

	chartPath, err := install.LocateChart(config.ChartRef, c.Settings)
	if err != nil {
		return nil, fmt.Errorf("locate chart: %w", err)
	}

	ch, err := loader.Load(chartPath)
	if err != nil {
		return nil, fmt.Errorf("load chart: %w", err)
	}

	rel, err := install.RunWithContext(ctx, ch, values)
	if err != nil {
		return nil, fmt.Errorf("install release: %w", err)
	}

	return rel, nil
}
