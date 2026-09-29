package helm

import (
	"errors"
	"fmt"
	"sort"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"
)

func (c *Client) History() ([]*release.Release, error) {
	history := action.NewHistory(c.ActionConfig)

	releases, err := history.Run(c.Options.Release)
	if err != nil {
		if errors.Is(err, driver.ErrReleaseNotFound) {
			return nil, errors.New("Releases no found")
		}
		return nil, fmt.Errorf("get release history: %w", err)
	}

	sort.Slice(releases, func(i, j int) bool {
		return releases[i].Version < releases[j].Version
	})

	return releases, nil
}
