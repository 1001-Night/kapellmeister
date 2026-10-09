package helm

import (
	"errors"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/version"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"

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

func TestPreflightKubernetesVersion(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	discovery := clientset.Discovery().(*fakediscovery.FakeDiscovery)
	discovery.FakedServerVersion = &version.Info{GitVersion: "v1.30.2"}
	client := Client{KubeClient: clientset}

	for _, tt := range []struct {
		name       string
		required   string
		wantErr    bool
		wantDetail string
	}{
		{name: "supported", required: ">= 1.29.0"},
		{name: "unsupported", required: ">= 1.31.0", wantErr: true, wantDetail: "does not satisfy"},
		{name: "invalid constraint", required: "not-a-version", wantErr: true, wantDetail: "invalid chart kubeVersion"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ch := &chart.Chart{Metadata: &chart.Metadata{KubeVersion: tt.required}}
			err := client.preflightKubernetesVersion(ch)
			if tt.wantErr {
				if err == nil || !strings.Contains(err.Error(), tt.wantDetail) {
					t.Fatalf("preflightKubernetesVersion() error = %v, want %q", err, tt.wantDetail)
				}
			} else if err != nil {
				t.Fatalf("preflightKubernetesVersion() error = %v", err)
			}
		})
	}
}

func TestPreflightKubernetesVersionClusterUnavailable(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	clientset.PrependReactor("get", "version", func(clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("connection refused")
	})

	client := Client{KubeClient: clientset}
	err := client.preflightKubernetesVersion(nil)
	if err == nil || !strings.Contains(err.Error(), "connection refused") {
		t.Fatalf("preflightKubernetesVersion() error = %v, want connection refused", err)
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

	objects, err := manifestObjects(rel.Manifest)
	if err != nil {
		t.Fatalf("manifestObjects() error = %v", err)
	}
	checks := client.verifyAPIs(objects)
	if len(checks) != 2 {
		t.Fatalf("len(checks) = %d, want 2", len(checks))
	}
	if hasFailedChecks(checks) {
		t.Fatalf("checks contain a failure: %#v", checks)
	}
}
