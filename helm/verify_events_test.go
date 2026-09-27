package helm

import (
	"context"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/kubernetes/fake"

	helmkube "helm.sh/helm/v3/pkg/kube"
)

func TestVerifyEventsReturnsRecentWarning(t *testing.T) {
	deployment, replicaSet, pod := eventTestWorkload()
	clientset := fake.NewSimpleClientset(deployment, replicaSet, pod, &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "warning",
			Namespace: "test",
		},
		InvolvedObject: corev1.ObjectReference{
			Kind:      "ReplicaSet",
			Name:      "app-rs",
			Namespace: "test",
		},
		Type:          corev1.EventTypeWarning,
		Reason:        "FailedCreate",
		Message:       "could not create Pod",
		LastTimestamp: metav1.NewTime(time.Now()),
	})

	client := Client{KubeClient: clientset}
	checks := client.verifyEvents(
		context.Background(),
		eventTestResources(deployment),
		nil,
	)
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

	client := Client{KubeClient: clientset}
	checks := client.verifyEvents(
		context.Background(),
		eventTestResources(deployment),
		[]corev1.Pod{*pod},
	)
	if len(checks) != 1 || checks[0].Status != CheckPassed {
		t.Fatalf("checks = %#v, want one passed check", checks)
	}
}

func eventTestResources(deployment *appsv1.Deployment) helmkube.ResourceList {
	return helmkube.ResourceList{&resource.Info{
		Name:      "app",
		Namespace: "test",
		Object:    deployment,
		Mapping: &meta.RESTMapping{
			GroupVersionKind: schema.GroupVersionKind{
				Group: "apps",
				Kind:  "Deployment",
			},
		},
	}}
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
				Kind:       "Deployment",
				Name:       deployment.Name,
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
				Kind:       "ReplicaSet",
				Name:       replicaSet.Name,
				Controller: &controller,
			}},
		},
	}

	return deployment, replicaSet, pod
}
