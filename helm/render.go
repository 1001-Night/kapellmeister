package helm

import (
	"context"
	"fmt"
	"strings"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"
)

func (c *Client) RenderInstall(
	ctx context.Context,
	ch *chart.Chart,
	values map[string]interface{},
) (string, error) {
	render := c.newInstallAction()

	// Рендер манифеста для проверки прав
	render.DryRun = true
	render.DryRunOption = "server"

	render.Wait = false
	render.WaitForJobs = false
	render.Atomic = false

	rel, err := render.RunWithContext(ctx, ch, values)
	if err != nil {
		return "", fmt.Errorf("render install: %w", err)
	}

	return releaseManifest(rel), nil
}

func (c *Client) RenderUpdate(
	ctx context.Context,
	ch *chart.Chart,
	values map[string]interface{},
) (string, error) {
	render := action.NewUpgrade(c.ActionConfig)
	render.SetRegistryClient(c.Registry)

	render.Namespace = c.Options.Namespace
	render.Version = c.Options.Version
	render.DryRun = true

	rel, err := render.RunWithContext(
		ctx,
		c.Options.Release,
		ch,
		values,
	)
	if err != nil {
		return "", fmt.Errorf("render update: %w", err)
	}

	return releaseManifest(rel), nil
}

func releaseManifest(rel *release.Release) string {
	var manifests strings.Builder

	manifests.WriteString(rel.Manifest)

	for _, hook := range rel.Hooks {
		manifests.WriteString("\n---\n")
		manifests.WriteString(hook.Manifest)
	}

	return manifests.String()
}
