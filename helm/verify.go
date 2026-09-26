package helm

import (
	"fmt"
	"strings"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/release"
)

func (c *Client) VerifyRelease() (*release.Release, error) {
	status := action.NewStatus(c.ActionConfig)

	rel, err := status.Run(c.Options.Release)
	if err != nil {
		return nil, fmt.Errorf(
			"get release %q status: %w",
			c.Options.Release,
			err,
		)
	}

	if rel.Info == nil {
		return nil, fmt.Errorf(
			"release %q has no status information",
			c.Options.Release,
		)
	}

	if rel.Info.Status != release.StatusDeployed {
		return nil, fmt.Errorf(
			"release %q is not deployed: status=%s",
			rel.Name,
			rel.Info.Status,
		)
	}

	return rel, nil
}

func (c *Client) VerifyResources(
	rel *release.Release,
) (int, error) {
	if strings.TrimSpace(rel.Manifest) == "" {
		return 0, fmt.Errorf(
			"release %q has an empty manifest",
			rel.Name,
		)
	}

	resources, err := c.ActionConfig.KubeClient.Build(
		strings.NewReader(rel.Manifest),
		false,
	)
	if err != nil {
		return 0, fmt.Errorf(
			"parse release manifest: %w",
			err,
		)
	}

	var failed []string

	for _, resource := range resources {
		if err := resource.Get(); err != nil {
			name := resource.ObjectName()

			if resource.Namespace != "" {
				name += fmt.Sprintf(
					" namespace=%q",
					resource.Namespace,
				)
			}

			failed = append(
				failed,
				fmt.Sprintf("%s: %v", name, err),
			)
		}
	}

	if len(failed) > 0 {
		return len(resources), fmt.Errorf(
			"resource verification failed:\n- %s",
			strings.Join(failed, "\n- "),
		)
	}

	return len(resources), nil
}