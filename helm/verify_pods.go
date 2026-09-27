package helm

import (
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	helmkube "helm.sh/helm/v3/pkg/kube"
)

func (c *Client) verifyPods(
	ctx context.Context,
	resources helmkube.ResourceList,
) ([]CheckResult, []corev1.Pod) {
	var checks []CheckResult
	var relatedPods []corev1.Pod
	seen := make(map[string]struct{})

	for _, resource := range resources {
		kind := resource.Mapping.GroupVersionKind.Kind
		if kind == "Pod" {
			pod, ok := helmkube.AsVersioned(resource).(*corev1.Pod)
			if !ok {
				checks = append(checks, failedWorkloadCheck(
					resource.ObjectName(),
					fmt.Errorf("convert Pod to core/v1"),
				))
				continue
			}
			key := pod.Namespace + "/" + pod.Name
			if _, ok := seen[key]; !ok {
				seen[key] = struct{}{}
				relatedPods = append(relatedPods, *pod)
				if pod.DeletionTimestamp == nil {
					checks = append(checks, checkPod(pod))
				}
			}
			continue
		}
		if !controlsPods(kind) {
			continue
		}

		object := helmkube.AsVersioned(resource)
		selector, err := helmkube.SelectorsForObject(object)
		if err != nil {
			checks = append(checks, failedWorkloadCheck(resource.ObjectName(), err))
			continue
		}

		pods, err := c.KubeClient.CoreV1().Pods(resource.Namespace).List(
			ctx,
			metav1.ListOptions{LabelSelector: selector.String()},
		)
		if err != nil {
			checks = append(checks, failedWorkloadCheck(
				resource.ObjectName(),
				fmt.Errorf("list Pods: %w", err),
			))
			continue
		}

		active := 0
		for i := range pods.Items {
			pod := &pods.Items[i]
			key := pod.Namespace + "/" + pod.Name
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			relatedPods = append(relatedPods, *pod)

			if pod.DeletionTimestamp != nil {
				continue
			}
			active++
			checks = append(checks, checkPod(pod))
		}

		desired, known := desiredPodCount(object)
		if known && desired == 0 && active == 0 {
			checks = append(checks, CheckResult{
				Section: "Pods",
				Name:    resource.ObjectName(),
				Detail:  "scaled to 0 replicas",
				Status:  CheckPassed,
			})
			continue
		}

		if active == 0 || known && int32(active) < desired {
			expected := "at least 1"
			if known {
				expected = fmt.Sprint(desired)
			}
			checks = append(checks, CheckResult{
				Section: "Pods",
				Name:    resource.ObjectName(),
				Detail:  fmt.Sprintf("Pods found=%d expected=%s", active, expected),
				Status:  CheckFailed,
			})
		}
	}

	return checks, relatedPods
}

func controlsPods(kind string) bool {
	switch kind {
	case "Deployment", "StatefulSet", "DaemonSet", "ReplicaSet",
		"ReplicationController":
		return true
	default:
		return false
	}
}

func desiredPodCount(object runtime.Object) (int32, bool) {
	switch workload := object.(type) {
	case *appsv1.Deployment:
		return replicasOrOne(workload.Spec.Replicas), true
	case *appsv1.StatefulSet:
		return replicasOrOne(workload.Spec.Replicas), true
	case *appsv1.ReplicaSet:
		return replicasOrOne(workload.Spec.Replicas), true
	case *corev1.ReplicationController:
		return replicasOrOne(workload.Spec.Replicas), true
	case *appsv1.DaemonSet:
		return workload.Status.DesiredNumberScheduled, true
	default:
		return 0, false
	}
}

func replicasOrOne(replicas *int32) int32 {
	if replicas == nil {
		return 1
	}
	return *replicas
}

func failedWorkloadCheck(name string, err error) CheckResult {
	return CheckResult{
		Section: "Pods",
		Name:    name,
		Detail:  err.Error(),
		Status:  CheckFailed,
	}
}

func checkPod(pod *corev1.Pod) CheckResult {
	check := CheckResult{
		Section: "Pods",
		Name:    "pod/" + pod.Name,
		Status:  CheckPassed,
	}

	var problems []string
	if pod.Status.Phase != corev1.PodRunning {
		problems = append(problems, "phase="+string(pod.Status.Phase))
	}

	ready := false
	for _, condition := range pod.Status.Conditions {
		if condition.Type != corev1.PodReady {
			continue
		}
		ready = condition.Status == corev1.ConditionTrue
		if !ready && condition.Message != "" {
			problems = append(problems, "not Ready: "+condition.Message)
		}
		break
	}
	if !ready && !containsProblemPrefix(problems, "not Ready") {
		problems = append(problems, "not Ready")
	}
	statuses := make([]corev1.ContainerStatus, 0,
		len(pod.Status.InitContainerStatuses)+
			len(pod.Status.ContainerStatuses)+
			len(pod.Status.EphemeralContainerStatuses),
	)
	statuses = append(statuses, pod.Status.InitContainerStatuses...)
	statuses = append(statuses, pod.Status.ContainerStatuses...)
	statuses = append(statuses, pod.Status.EphemeralContainerStatuses...)

	var restarts int32
	for _, status := range statuses {
		restarts += status.RestartCount
		if status.State.Waiting == nil {
			continue
		}

		reason := status.State.Waiting.Reason
		if reason == "CrashLoopBackOff" ||
			reason == "ImagePullBackOff" ||
			reason == "ErrImagePull" {
			problems = append(
				problems,
				fmt.Sprintf("container %s: %s", status.Name, reason),
			)
		}
	}

	if len(problems) > 0 {
		check.Status = CheckFailed
		check.Detail = fmt.Sprintf(
			"%s; restarts=%d",
			strings.Join(problems, "; "),
			restarts,
		)
		return check
	}

	check.Detail = fmt.Sprintf("Running, Ready, restarts=%d", restarts)
	if restarts > 0 {
		check.Status = CheckWarning
	}
	return check
}

func containsProblemPrefix(problems []string, prefix string) bool {
	for _, problem := range problems {
		if strings.HasPrefix(problem, prefix) {
			return true
		}
	}
	return false
}
