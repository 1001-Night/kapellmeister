package helm

import (
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

type ResourceCheck struct {
	Name   string
	Detail string
	Passed bool
}

func (c *Client) VerifyResources(
	ctx context.Context,
	rel *release.Release,
) ([]ResourceCheck, error) {
	if strings.TrimSpace(rel.Manifest) == "" {
		return nil, fmt.Errorf(
			"release %q has an empty manifest",
			rel.Name,
		)
	}

	resources, err := c.ActionConfig.KubeClient.Build(
		strings.NewReader(rel.Manifest),
		false,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"parse release manifest: %w",
			err,
		)
	}

	checks := make([]ResourceCheck, 0, len(resources))

	for _, resource := range resources {
		check := ResourceCheck{
			Name: resource.ObjectName(),
		}

		if err := resource.Get(); err != nil {
			check.Detail = err.Error()

			if resource.Namespace != "" {
				check.Detail = fmt.Sprintf(
					"namespace=%s: %v",
					resource.Namespace,
					err,
				)
			}

			checks = append(checks, check)
			continue
		}

		check.Passed = true
		check.Detail = "exists"

		gvk := resource.Mapping.GroupVersionKind

		if gvk.Group == "apps" && gvk.Kind == "Deployment" {
			detail, err := c.verifyDeployment(
				ctx,
				resource.Namespace,
				resource.Name,
			)

			if err != nil {
				check.Passed = false
				check.Detail = err.Error()
			} else {
				check.Detail = detail
			}
		}

		checks = append(checks, check)
	}

	return checks, nil
}

func (c *Client) verifyDeployment(
	ctx context.Context,
	namespace string,
	name string,
) (string, error) {
	deployment, err := c.KubeClient.
		AppsV1().
		Deployments(namespace).
		Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf(
			"get current state: %w",
			err,
		)
	}

	var desired int32 = 1
	if deployment.Spec.Replicas != nil {
		desired = *deployment.Spec.Replicas
	}

	var problems []string

	if deployment.Status.ObservedGeneration < deployment.Generation {
		problems = append(
			problems,
			fmt.Sprintf(
				"generation is not observed: observed=%d expected=%d",
				deployment.Status.ObservedGeneration,
				deployment.Generation,
			),
		)
	}

	if deployment.Status.UpdatedReplicas != desired {
		problems = append(
			problems,
			fmt.Sprintf(
				"updated=%d expected=%d",
				deployment.Status.UpdatedReplicas,
				desired,
			),
		)
	}

	if deployment.Status.ReadyReplicas != desired {
		problems = append(
			problems,
			fmt.Sprintf(
				"ready=%d expected=%d",
				deployment.Status.ReadyReplicas,
				desired,
			),
		)
	}

	if deployment.Status.AvailableReplicas < desired {
		problems = append(
			problems,
			fmt.Sprintf(
				"available=%d expected=%d",
				deployment.Status.AvailableReplicas,
				desired,
			),
		)
	}

	for _, condition := range deployment.Status.Conditions {
		if condition.Type == appsv1.DeploymentProgressing &&
			condition.Status == "False" {
			problems = append(
				problems,
				fmt.Sprintf(
					"progressing=false: %s",
					condition.Message,
				),
			)
		}
	}

	if len(problems) > 0 {
		return "", fmt.Errorf(
			"not ready: %s",
			strings.Join(problems, "; "),
		)
	}

	return fmt.Sprintf(
		"%d/%d ready",
		deployment.Status.ReadyReplicas,
		desired,
	), nil
}
