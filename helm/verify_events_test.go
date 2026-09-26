package helm

import (
	"context"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"

	"helm.sh/helm/v3/pkg/release"

	"gitlab.com/1001n/kapellmeister/config"
)

func TestVerifyEventsReturnsRecentWarning(t *testing.T) {
	deployment, replicaSet, pod := eventTestWorkload()
	clientset := fake.NewSimpleClientset(deployment, replicaSet, pod, &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "warning",
			Namespace: "test",
		},
		InvolvedObject: corev1.ObjectReference{
			Kind:      "Pod",
			Name:      "app-abc",
			Namespace: "test",
		},
		Type:          corev1.EventTypeWarning,
		Reason:        "FailedCreate",
		Message:       "could not create Pod",
		LastTimestamp: metav1.NewTime(time.Now()),
	})

	client := Client{
		KubeClient: clientset,
		Options:    config.Options{Namespace: "helm-storage"},
	}
	rel := &release.Release{Manifest: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
  namespace: test
`}

	checks, err := client.verifyEvents(context.Background(), rel)
	if err != nil {
		t.Fatalf("verifyEvents() error = %v", err)
	}
	if len(checks) != 1 || checks[0].Status != CheckWarning {
		t.Fatalf("checks = %#v, want one warning", checks)
	}
}

func TestVerifyEventsIgnoresOldWarning(t *testing.T) {
	deployment, replicaSet, pod := eventTestWorkload()
	clientset := fake.NewSimpleClientset(deployment, replicaSet, pod, &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "old-warning",
			Namespace: "test",
		},
		InvolvedObject: corev1.ObjectReference{
			Kind:      "Deployment",
			Name:      "app",
			Namespace: "test",
		},
		Type:          corev1.EventTypeWarning,
		Reason:        "FailedCreate",
		LastTimestamp: metav1.NewTime(time.Now().Add(-time.Hour)),
	})

	client := Client{
		KubeClient: clientset,
		Options:    config.Options{Namespace: "test"},
	}
	rel := &release.Release{Manifest: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
  namespace: test
`}

	checks, err := client.verifyEvents(context.Background(), rel)
	if err != nil {
		t.Fatalf("verifyEvents() error = %v", err)
	}
	if len(checks) != 1 || checks[0].Status != CheckPassed {
		t.Fatalf("checks = %#v, want one passed check", checks)
	}
}

func eventTestWorkload() (*appsv1.Deployment, *appsv1.ReplicaSet, *corev1.Pod) {
	controller := true
	selector := &metav1.LabelSelector{MatchLabels: map[string]string{"app": "test"}}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app",
			Namespace: "test",
			UID:       types.UID("deployment-uid"),
		},
		Spec: appsv1.DeploymentSpec{Selector: selector},
	}
	replicaSet := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app-rs",
			Namespace: "test",
			UID:       types.UID("replicaset-uid"),
			Labels:    map[string]string{"app": "test"},
			OwnerReferences: []metav1.OwnerReference{{
				UID:        deployment.UID,
				Controller: &controller,
			}},
		},
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app-abc",
			Namespace: "test",
			Labels:    map[string]string{"app": "test"},
			OwnerReferences: []metav1.OwnerReference{{
				UID:        replicaSet.UID,
				Controller: &controller,
			}},
		},
	}

	return deployment, replicaSet, pod
}
