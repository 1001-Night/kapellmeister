package helm

import (
	"context"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/kubernetes/fake"

	helmkube "helm.sh/helm/v3/pkg/kube"
)

func TestCheckPod(t *testing.T) {
	ready := corev1.PodCondition{
		Type:   corev1.PodReady,
		Status: corev1.ConditionTrue,
	}

	tests := []struct {
		name       string
		pod        corev1.Pod
		wantStatus CheckStatus
		wantDetail string
	}{
		{
			name: "running and ready",
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "healthy"},
				Status: corev1.PodStatus{
					Phase:      corev1.PodRunning,
					Conditions: []corev1.PodCondition{ready},
					ContainerStatuses: []corev1.ContainerStatus{{
						Name: "app",
					}},
				},
			},
			wantStatus: CheckPassed,
			wantDetail: "restarts=0",
		},
		{
			name: "healthy with restarts",
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "restarted"},
				Status: corev1.PodStatus{
					Phase:      corev1.PodRunning,
					Conditions: []corev1.PodCondition{ready},
					ContainerStatuses: []corev1.ContainerStatus{{
						Name:         "app",
						RestartCount: 2,
					}},
				},
			},
			wantStatus: CheckWarning,
			wantDetail: "restarts=2",
		},
		{
			name: "crash loop",
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "crashing"},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					ContainerStatuses: []corev1.ContainerStatus{{
						Name:         "app",
						RestartCount: 5,
						State: corev1.ContainerState{
							Waiting: &corev1.ContainerStateWaiting{
								Reason: "CrashLoopBackOff",
							},
						},
					}},
				},
			},
			wantStatus: CheckFailed,
			wantDetail: "CrashLoopBackOff",
		},
		{
			name: "image pull failure",
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "no-image"},
				Status: corev1.PodStatus{
					Phase: corev1.PodPending,
					ContainerStatuses: []corev1.ContainerStatus{{
						Name: "app",
						State: corev1.ContainerState{
							Waiting: &corev1.ContainerStateWaiting{
								Reason: "ImagePullBackOff",
							},
						},
					}},
				},
			},
			wantStatus: CheckFailed,
			wantDetail: "ImagePullBackOff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check := checkPod(&tt.pod)
			if check.Status != tt.wantStatus {
				t.Errorf("status = %q, want %q", check.Status, tt.wantStatus)
			}
			if !strings.Contains(check.Detail, tt.wantDetail) {
				t.Errorf("detail = %q, want it to contain %q", check.Detail, tt.wantDetail)
			}
		})
	}
}

func TestDesiredPodCountSupportsStatefulSet(t *testing.T) {
	replicas := int32(3)
	desired, known := desiredPodCount(&appsv1.StatefulSet{
		Spec: appsv1.StatefulSetSpec{Replicas: &replicas},
	})
	if !known || desired != replicas {
		t.Fatalf("desiredPodCount() = (%d, %t), want (%d, true)", desired, known, replicas)
	}
}

func TestVerifyPodsSupportsStatefulSet(t *testing.T) {
	replicas := int32(1)
	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "database", Namespace: "test"},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "database"},
			},
		},
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "database-0",
			Namespace: "test",
			Labels:    map[string]string{"app": "database"},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{{
				Type:   corev1.PodReady,
				Status: corev1.ConditionTrue,
			}},
		},
	}

	client := Client{KubeClient: fake.NewSimpleClientset(pod)}
	resources := helmkube.ResourceList{&resource.Info{
		Name:      statefulSet.Name,
		Namespace: statefulSet.Namespace,
		Object:    statefulSet,
		Mapping: &meta.RESTMapping{
			GroupVersionKind: schema.GroupVersionKind{
				Group:   "apps",
				Version: "v1",
				Kind:    "StatefulSet",
			},
		},
	}}

	checks, pods := client.verifyPods(context.Background(), resources)
	if len(checks) != 1 || checks[0].Status != CheckPassed {
		t.Fatalf("checks = %#v, want one passed Pod check", checks)
	}
	if len(pods) != 1 || pods[0].Name != pod.Name {
		t.Fatalf("pods = %#v, want %q", pods, pod.Name)
	}
}
