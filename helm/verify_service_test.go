package helm

import (
	"context"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/kubernetes/fake"
)

func TestVerifyServiceEndpoints(t *testing.T) {
	ready := true
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "test"},
	}
	clientset := fake.NewSimpleClientset(
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
	detail, err := client.verifyServiceEndpoints(context.Background(), &resource.Info{Object: service})
	if err != nil {
		t.Fatalf("verifyServiceEndpoints() error = %v", err)
	}
	if !strings.Contains(detail, "1 ready endpoints") {
		t.Fatalf("detail = %q", detail)
	}
}

func TestVerifyServiceEndpointsExternalName(t *testing.T) {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "external", Namespace: "test"},
		Spec: corev1.ServiceSpec{
			Type:         corev1.ServiceTypeExternalName,
			ExternalName: "example.com",
		},
	}
	client := Client{KubeClient: fake.NewSimpleClientset()}
	detail, err := client.verifyServiceEndpoints(context.Background(), &resource.Info{Object: service})
	if err != nil || detail != "ExternalName example.com" {
		t.Fatalf("verifyServiceEndpoints() = (%q, %v)", detail, err)
	}
}
