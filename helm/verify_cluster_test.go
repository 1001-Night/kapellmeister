package helm

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/version"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/fake"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"
)

func TestVerifyKubernetesVersion(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	discovery := clientset.Discovery().(*fakediscovery.FakeDiscovery)
	discovery.FakedServerVersion = &version.Info{GitVersion: "v1.30.2"}

	client := Client{KubeClient: clientset}
	rel := &release.Release{
		Chart: &chart.Chart{Metadata: &chart.Metadata{
			KubeVersion: ">= 1.29.0",
		}},
	}

	check := client.verifyKubernetesVersion(rel)
	if check.Status != CheckPassed {
		t.Fatalf("status = %q, detail = %q", check.Status, check.Detail)
	}

	rel.Chart.Metadata.KubeVersion = ">= 1.31.0"
	check = client.verifyKubernetesVersion(rel)
	if check.Status != CheckFailed {
		t.Fatalf("status = %q, want %q", check.Status, CheckFailed)
	}
}

func TestVerifyAPIs(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	discovery := clientset.Discovery().(*fakediscovery.FakeDiscovery)
	discovery.Resources = []*metav1.APIResourceList{
		{
			GroupVersion: "v1",
			APIResources: []metav1.APIResource{{Name: "services", Kind: "Service"}},
		},
		{
			GroupVersion: "apps/v1",
			APIResources: []metav1.APIResource{{Name: "deployments", Kind: "Deployment"}},
		},
	}

	client := Client{KubeClient: clientset}
	rel := &release.Release{Manifest: `
apiVersion: v1
kind: Service
metadata:
  name: app
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
`}

	checks, err := client.verifyAPIs(rel)
	if err != nil {
		t.Fatalf("verifyAPIs() error = %v", err)
	}
	if len(checks) != 2 {
		t.Fatalf("len(checks) = %d, want 2", len(checks))
	}
	if hasFailedChecks(checks) {
		t.Fatalf("checks contain a failure: %#v", checks)
	}
}
