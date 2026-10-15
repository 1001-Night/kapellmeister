package helm

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/cli-runtime/pkg/resource"

	helmkube "helm.sh/helm/v3/pkg/kube"
)

func (c *Client) verifyServiceEndpoints(
	ctx context.Context,
	info *resource.Info,
) (string, error) {
	service, ok := helmkube.AsVersioned(info).(*corev1.Service)
	if !ok {
		return "", fmt.Errorf("convert Service to core/v1")
	}

	if service.Spec.Type == corev1.ServiceTypeExternalName {
		return fmt.Sprintf("ExternalName %s", service.Spec.ExternalName), nil
	}

	selector := labels.Set{
		discoveryv1.LabelServiceName: service.Name,
	}.AsSelector().String()

	slices, err := c.KubeClient.
		DiscoveryV1().
		EndpointSlices(service.Namespace).
		List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return "", fmt.Errorf("list EndpointSlices: %w", err)
	}
	if len(slices.Items) == 0 {
		return "", fmt.Errorf("no EndpointSlice found")
	}

	ready := 0
	for _, slice := range slices.Items {
		for _, endpoint := range slice.Endpoints {
			if endpoint.Conditions.Ready == nil || *endpoint.Conditions.Ready {
				ready += len(endpoint.Addresses)
			}
		}
	}

	if ready == 0 {
		return "", fmt.Errorf("no ready endpoints")
	}

	return fmt.Sprintf(
		"%d ready endpoints in %d EndpointSlices",
		ready,
		len(slices.Items),
	), nil
}
