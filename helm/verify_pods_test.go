package helm

import (
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
