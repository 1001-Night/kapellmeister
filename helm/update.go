package helm

import (
	"context"
	"fmt"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/release"

	"gitlab.com/1001n/kapellmeister/config"
)

func (c *Client) Update(
	ctx context.Context,
	values map[string]interface{},
) (*release.Release, error) {

	upgrade := action.NewUpgrade(c.ActionConfig)
	upgrade.SetRegistryClient(c.Registry)

	upgrade.Namespace = c.Options.Namespace
	upgrade.Version = c.Options.Version

	upgrade.Wait = true
	upgrade.WaitForJobs = true
	upgrade.Atomic = true
	upgrade.Timeout = c.Options.Timeout

	chartPath, err := upgrade.LocateChart(config.ChartRef, c.Settings)
	if err != nil {
		return nil, fmt.Errorf("locate chart: %w", err)
	}

	ch, err := loader.Load(chartPath)
	if err != nil {
		return nil, fmt.Errorf("load chart: %w", err)
	}

	manifest, err := c.RenderUpdate(ctx, ch, values)
	if err != nil {
		return nil, err
	}

	get := action.NewGet(c.ActionConfig)

	currentRelease, err := get.Run(c.Options.Release)
	if err != nil {
		return nil, fmt.Errorf("get current release: %w", err)
	}

	if err := c.CheckPermissions(
		ctx,
		OperationUpdate,
		manifest,
		releaseManifest(currentRelease),
	); err != nil {
		return nil, err
	}

	rel, err := upgrade.RunWithContext(ctx, c.Options.Release, ch, values)
	if err != nil {
		return nil, fmt.Errorf("upgrade release: %w", err)
	}

	return rel, nil
}
