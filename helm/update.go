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
	upgrade.Timeout = 5 * time.Minute

	chartPath, err := upgrade.LocateChart(config.ChartRef, c.Settings)
	if err != nil {
		return nil, fmt.Errorf("locate chart: %w", err)
	}

	ch, err := loader.Load(chartPath)
	if err != nil {
		return nil, fmt.Errorf("load chart: %w", err)
	}

	rel, err := upgrade.RunWithContext(ctx, c.Options.Release, ch, values)
	if err != nil {
		return nil, fmt.Errorf("upgrade release: %w", err)
	}

	return rel, nil
}
