package helm

import (
	"fmt"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"
)

// preflightKubernetesVersion проверяет кластер до изменения релиза.
func (c *Client) preflightKubernetesVersion(ch *chart.Chart) error {
	check := c.verifyKubernetesVersion(&release.Release{Chart: ch})
	if check.Status != CheckPassed {
		return fmt.Errorf("check Kubernetes version: %s", check.Detail)
	}
	return nil
}
