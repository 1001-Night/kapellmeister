package helm

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"helm.sh/helm/v3/pkg/release"
)

func (c *Client) verifyPods(
	ctx context.Context,
	rel *release.Release,
) ([]CheckResult, error) {
	objects, err := manifestObjects(rel.Manifest)
	if err != nil {
		return nil, fmt.Errorf("inspect workloads for pods: %w", err)
	}

	var checks []CheckResult
	seen := make(map[string]struct{})

	for _, object := range objects {
		gvk := object.GroupVersionKind()
		if gvk.Group != "apps" || gvk.Kind != "Deployment" {
			continue
		}

		namespace := object.GetNamespace()
		if namespace == "" {
			namespace = c.Options.Namespace
		}

		deployment, err := c.KubeClient.
			AppsV1().
			Deployments(namespace).
			Get(ctx, object.GetName(), metav1.GetOptions{})
		if err != nil {
			checks = append(checks, CheckResult{
				Section: "Pods",
				Name:    fmt.Sprintf("deployment/%s", object.GetName()),
				Detail:  fmt.Sprintf("get Deployment: %v", err),
				Status:  CheckFailed,
			})
			continue
		}

		if deployment.Spec.Selector == nil {
			checks = append(checks, CheckResult{
				Section: "Pods",
				Name:    fmt.Sprintf("deployment/%s", deployment.Name),
				Detail:  "Deployment has no Pod selector",
				Status:  CheckFailed,
			})
			continue
		}

		selector, err := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
		if err != nil {
			checks = append(checks, CheckResult{
				Section: "Pods",
				Name:    fmt.Sprintf("deployment/%s", deployment.Name),
				Detail:  fmt.Sprintf("invalid Pod selector: %v", err),
				Status:  CheckFailed,
			})
			continue
		}

		pods, err := c.KubeClient.
			CoreV1().
			Pods(namespace).
			List(ctx, metav1.ListOptions{LabelSelector: selector.String()})
		if err != nil {
			checks = append(checks, CheckResult{
				Section: "Pods",
				Name:    fmt.Sprintf("deployment/%s", deployment.Name),
				Detail:  fmt.Sprintf("list Pods: %v", err),
				Status:  CheckFailed,
			})
			continue
		}

		desired := int32(1)
		if deployment.Spec.Replicas != nil {
			desired = *deployment.Spec.Replicas
		}

		active := 0
		for i := range pods.Items {
			pod := &pods.Items[i]
			if pod.DeletionTimestamp != nil {
				continue
			}

			key := pod.Namespace + "/" + pod.Name
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			active++
			checks = append(checks, checkPod(pod))
		}

		if desired == 0 && active == 0 {
			checks = append(checks, CheckResult{
				Section: "Pods",
				Name:    fmt.Sprintf("deployment/%s", deployment.Name),
				Detail:  "scaled to 0 replicas",
				Status:  CheckPassed,
			})
			continue
		}

		if int32(active) < desired {
			checks = append(checks, CheckResult{
				Section: "Pods",
				Name:    fmt.Sprintf("deployment/%s", deployment.Name),
				Detail: fmt.Sprintf(
					"Pods found=%d expected=%d",
					active,
					desired,
				),
				Status: CheckFailed,
			})
		}
	}

	return checks, nil
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
