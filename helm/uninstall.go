package helm

import (
	"context"
	"fmt"
	"time"

	"helm.sh/helm/v3/pkg/action"
)

func (c *Client) Uninstall(ctx context.Context) error {

	uninstall := action.NewUninstall(c.ActionConfig)

	uninstall.Wait = true
	uninstall.Timeout = c.Options.Timeout
	uninstall.IgnoreNotFound = false
	uninstall.KeepHistory = false

	get := action.NewGet(c.ActionConfig)

	rel, err := get.Run(c.Options.Release)
	if err != nil {
		return err
	}

	manifest := releaseManifest(rel)

	if err := c.CheckPermissions(
		ctx,
		OperationUninstall,
		"",
		manifest,
	); err != nil {
		return err
	}

	_, err = uninstall.Run(c.Options.Release)
	if err != nil {
		return fmt.Errorf("uninstall release: %w", err)
	}

	return nil
}
