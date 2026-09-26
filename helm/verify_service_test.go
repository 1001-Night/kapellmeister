package helm

import (
	"context"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestVerifyServiceEndpoints(t *testing.T) {
	ready := true
	clientset := fake.NewSimpleClientset(
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "app",
				Namespace: "test",
			},
		},
		&discoveryv1.EndpointSlice{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "app-123",
				Namespace: "test",
				Labels: map[string]string{
					discoveryv1.LabelServiceName: "app",
				},
			},
			Endpoints: []discoveryv1.Endpoint{{
				Addresses:  []string{"10.0.0.2"},
				Conditions: discoveryv1.EndpointConditions{Ready: &ready},
			}},
		},
	)

	client := Client{KubeClient: clientset}
	detail, err := client.verifyServiceEndpoints(context.Background(), "test", "app")
	if err != nil {
		t.Fatalf("verifyServiceEndpoints() error = %v", err)
	}
	if !strings.Contains(detail, "1 ready endpoints") {
		t.Fatalf("detail = %q", detail)
	}
}
