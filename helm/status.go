package helm

import (
	"fmt"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/release"
)

// Status возвращает последнее состояние релиза, записанное Helm.
func (c *Client) Status() (*release.Release, error) {
	rel, err := action.NewStatus(c.ActionConfig).Run(c.Options.Release)
	if err != nil {
		return nil, fmt.Errorf("get release %q status: %w", c.Options.Release, err)
	}
	return rel, nil
}
