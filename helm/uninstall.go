package helm

import (
	"context"
	"fmt"
	"time"
 
	"helm.sh/helm/v3/pkg/action"
)

func (c *Client) Uninstall(
	ctx context.Context,
	releaseName string,
) error {
	
	uninstall := action.NewUninstall(c.ActionConfig)
 
	uninstall.Wait = true
	uninstall.Timeout = 5 * time.Minute
	uninstall.IgnoreNotFound = false
	uninstall.KeepHistory = false

	_, err := uninstall.Run(releaseName)
	if err != nil {
		return fmt.Errorf("uninstall release: %w", err)
	}

	return nil
}