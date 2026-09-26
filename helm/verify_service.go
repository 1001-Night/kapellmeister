package helm

import (
	"context"
	"fmt"

	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func (c *Client) verifyServiceEndpoints(
	ctx context.Context,
	namespace string,
	name string,
) (string, error) {
	service, err := c.KubeClient.
		CoreV1().
		Services(namespace).
		Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get current state: %w", err)
	}

	if service.Spec.Type == "ExternalName" {
		return fmt.Sprintf("ExternalName %s", service.Spec.ExternalName), nil
	}

	selector := labels.Set{
		discoveryv1.LabelServiceName: name,
	}.AsSelector().String()

	slices, err := c.KubeClient.
		DiscoveryV1().
		EndpointSlices(namespace).
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
