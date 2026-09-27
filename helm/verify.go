package helm

import (
	"context"
	"fmt"
	"strings"

	"helm.sh/helm/v3/pkg/action"
	helmkube "helm.sh/helm/v3/pkg/kube"
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

type CheckResult struct {
	Section string
	Name    string
	Detail  string
	Status  CheckStatus
}

type CheckStatus string

const (
	CheckPassed  CheckStatus = "passed"
	CheckWarning CheckStatus = "warning"
	CheckFailed  CheckStatus = "failed"
)

func (c *Client) Verify(
	ctx context.Context,
	rel *release.Release,
) ([]CheckResult, error) {
	if strings.TrimSpace(rel.Manifest) == "" {
		return nil, fmt.Errorf("release %q has an empty manifest", rel.Name)
	}

	objects, err := manifestObjects(rel.Manifest)
	if err != nil {
		return nil, fmt.Errorf("inspect release manifest: %w", err)
	}

	checks := []CheckResult{c.verifyKubernetesVersion(rel)}
	apiChecks := c.verifyAPIs(objects)
	checks = append(checks, apiChecks...)
	if hasFailedChecks(apiChecks) {
		return checks, nil
	}

	resources, err := c.ActionConfig.KubeClient.Build(
		strings.NewReader(rel.Manifest),
		false,
	)
	if err != nil {
		return nil, fmt.Errorf("build release resources: %w", err)
	}

	checks = append(checks, c.verifyResources(ctx, resources)...)

	podChecks, pods := c.verifyPods(ctx, resources)
	checks = append(checks, podChecks...)
	checks = append(checks, c.verifyEvents(ctx, resources, pods)...)

	return checks, nil
}

func hasFailedChecks(checks []CheckResult) bool {
	for _, check := range checks {
		if check.Status == CheckFailed {
			return true
		}
	}
	return false
}

func (c *Client) verifyResources(
	ctx context.Context,
	resources helmkube.ResourceList,
) []CheckResult {
	checks := make([]CheckResult, 0, len(resources))

	for _, resource := range resources {
		check := CheckResult{
			Section: "Resources",
			Name:    resource.ObjectName(),
			Status:  CheckFailed,
		}

		if err := resource.Get(); err != nil {
			check.Detail = resourceError(resource.Namespace, err)
			checks = append(checks, check)
			continue
		}

		check.Status = CheckPassed
		check.Detail = "exists"

		if helmChecksReadiness(resource.Mapping.GroupVersionKind.Kind) {
			var reasons []string
			readyChecker := helmkube.NewReadyChecker(
				c.KubeClient,
				func(format string, args ...interface{}) {
					reasons = append(reasons, fmt.Sprintf(format, args...))
				},
				helmkube.CheckJobs(true),
			)

			ready, err := readyChecker.IsReady(ctx, resource)
			switch {
			case err != nil:
				check.Status = CheckFailed
				check.Detail = err.Error()
			case !ready:
				check.Status = CheckFailed
				check.Detail = "not ready"
				if len(reasons) > 0 {
					check.Detail += ": " + reasons[len(reasons)-1]
				}
			default:
				check.Detail = "ready"
			}
		}

		if resource.Mapping.GroupVersionKind.Kind == "Service" {
			detail, err := c.verifyServiceEndpoints(
				ctx,
				resource.Namespace,
				resource.Name,
			)
			if err != nil {
				check.Status = CheckFailed
				check.Detail = err.Error()
			} else {
				check.Detail = detail
			}
		}

		checks = append(checks, check)
	}

	return checks
}

func resourceError(namespace string, err error) string {
	if namespace == "" {
		return err.Error()
	}
	return fmt.Sprintf("namespace=%s: %v", namespace, err)
}

func helmChecksReadiness(kind string) bool {
	switch kind {
	case "Pod", "Job", "Deployment", "PersistentVolumeClaim", "Service",
		"DaemonSet", "CustomResourceDefinition", "StatefulSet",
		"ReplicationController", "ReplicaSet":
		return true
	default:
		return false
	}
}
