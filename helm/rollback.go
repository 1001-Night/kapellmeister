package helm

import (
	"context"
	"fmt"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/release"
)

func (c *Client) Rollback(
	ctx context.Context,
	revision int,
) (*release.Release, int, error) {
	get := action.NewGet(c.ActionConfig)
	current, err := get.Run(c.Options.Release)
	if err != nil {
		return nil, 0, fmt.Errorf("get current release: %w", err)
	}

	targetRevision := revision
	if targetRevision == 0 {
		targetRevision = current.Version - 1
	}
	if targetRevision < 1 {
		return nil, 0, fmt.Errorf(
			"release %q has no previous revision",
			c.Options.Release,
		)
	}

	targetGet := action.NewGet(c.ActionConfig)
	targetGet.Version = targetRevision
	target, err := targetGet.Run(c.Options.Release)
	if err != nil {
		return nil, 0, fmt.Errorf(
			"get target revision %d: %w",
			targetRevision,
			err,
		)
	}
	if err := c.preflightKubernetesVersion(target.Chart); err != nil {
		return nil, 0, err
	}
	if err := preflightImageRegistries(ctx, releaseManifest(target)); err != nil {
		return nil, 0, err
	}

	if err := c.CheckPermissions(
		ctx,
		OperationRollback,
		releaseManifest(target),
		releaseManifest(current),
	); err != nil {
		return nil, 0, err
	}

	rollback := action.NewRollback(c.ActionConfig)
	rollback.Version = targetRevision
	rollback.Wait = true
	rollback.WaitForJobs = true
	rollback.CleanupOnFail = true
	rollback.Timeout = c.Options.Timeout

	if err := rollback.Run(c.Options.Release); err != nil {
		return nil, 0, fmt.Errorf("rollback release: %w", err)
	}

	rel, err := c.VerifyRelease()
	if err != nil {
		return nil, 0, err
	}

	return rel, targetRevision, nil
}
