package helm

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/resource"

	helmkube "helm.sh/helm/v3/pkg/kube"
)

const warningEventWindow = 15 * time.Minute

type eventTarget struct {
	Namespace string
	Kind      string
	Name      string
}

func (c *Client) verifyEvents(
	ctx context.Context,
	resources helmkube.ResourceList,
	pods []corev1.Pod,
) []CheckResult {
	targets := make(map[eventTarget]struct{})
	namespaces := make(map[string]struct{})

	for _, resource := range resources {
		addEventTarget(targets, namespaces, eventTarget{
			Namespace: resource.Namespace,
			Kind:      resource.Mapping.GroupVersionKind.Kind,
			Name:      resource.Name,
		})

		if resource.Mapping.GroupVersionKind.Kind == "Deployment" {
			if err := c.addReplicaSetEventTargets(
				ctx,
				resource,
				targets,
				namespaces,
			); err != nil {
				return []CheckResult{{
					Section: "Events",
					Name:    resource.ObjectName(),
					Detail:  err.Error(),
					Status:  CheckFailed,
				}}
			}
		}
	}

	for _, pod := range pods {
		addEventTarget(targets, namespaces, eventTarget{
			Namespace: pod.Namespace,
			Kind:      "Pod",
			Name:      pod.Name,
		})

		for _, owner := range pod.OwnerReferences {
			addEventTarget(targets, namespaces, eventTarget{
				Namespace: pod.Namespace,
				Kind:      owner.Kind,
				Name:      owner.Name,
			})
		}
	}

	var warnings []corev1.Event
	for namespace := range namespaces {
		events, err := c.KubeClient.CoreV1().Events(namespace).List(
			ctx,
			metav1.ListOptions{},
		)
		if err != nil {
			return []CheckResult{{
				Section: "Events",
				Name:    namespace,
				Detail:  fmt.Sprintf("list events: %v", err),
				Status:  CheckFailed,
			}}
		}

		for _, event := range events.Items {
			if event.Type != corev1.EventTypeWarning {
				continue
			}
			if eventTimestamp(event).Before(time.Now().Add(-warningEventWindow)) {
				continue
			}

			target := eventTarget{
				Namespace: event.InvolvedObject.Namespace,
				Kind:      event.InvolvedObject.Kind,
				Name:      event.InvolvedObject.Name,
			}
			if target.Namespace == "" {
				target.Namespace = event.Namespace
			}
			if _, ok := targets[target]; ok {
				warnings = append(warnings, event)
			}
		}
	}

	if len(warnings) == 0 {
		return []CheckResult{{
			Section: "Events",
			Name:    "Warnings",
			Detail:  fmt.Sprintf("none in the last %s", warningEventWindow),
			Status:  CheckPassed,
		}}
	}

	sort.Slice(warnings, func(i, j int) bool {
		return eventTimestamp(warnings[i]).Before(eventTimestamp(warnings[j]))
	})

	checks := make([]CheckResult, 0, len(warnings))
	for _, event := range warnings {
		message := strings.Join(strings.Fields(event.Message), " ")
		if len(message) > 160 {
			message = message[:157] + "..."
		}

		detail := message
		if event.Count > 1 {
			detail = fmt.Sprintf("%s (count=%d)", detail, event.Count)
		}

		checks = append(checks, CheckResult{
			Section: "Events",
			Name: fmt.Sprintf(
				"%s/%s %s",
				event.InvolvedObject.Kind,
				event.InvolvedObject.Name,
				event.Reason,
			),
			Detail: detail,
			Status: CheckWarning,
		})
	}

	return checks
}

func (c *Client) addReplicaSetEventTargets(
	ctx context.Context,
	resource *resource.Info,
	targets map[eventTarget]struct{},
	namespaces map[string]struct{},
) error {
	deployment, ok := helmkube.AsVersioned(resource).(*appsv1.Deployment)
	if !ok {
		return fmt.Errorf("convert Deployment to apps/v1")
	}
	selector, err := helmkube.SelectorsForObject(deployment)
	if err != nil {
		return fmt.Errorf("get Deployment selector: %w", err)
	}

	replicaSets, err := c.KubeClient.AppsV1().ReplicaSets(resource.Namespace).List(
		ctx,
		metav1.ListOptions{LabelSelector: selector.String()},
	)
	if err != nil {
		return fmt.Errorf("list ReplicaSets: %w", err)
	}

	for i := range replicaSets.Items {
		replicaSet := &replicaSets.Items[i]
		if !metav1.IsControlledBy(replicaSet, deployment) {
			continue
		}
		addEventTarget(targets, namespaces, eventTarget{
			Namespace: replicaSet.Namespace,
			Kind:      "ReplicaSet",
			Name:      replicaSet.Name,
		})
	}

	return nil
}

func addEventTarget(
	targets map[eventTarget]struct{},
	namespaces map[string]struct{},
	target eventTarget,
) {
	if target.Namespace == "" {
		return
	}
	targets[target] = struct{}{}
	namespaces[target.Namespace] = struct{}{}
}

func eventTimestamp(event corev1.Event) time.Time {
	if event.Series != nil && !event.Series.LastObservedTime.IsZero() {
		return event.Series.LastObservedTime.Time
	}
	if !event.EventTime.IsZero() {
		return event.EventTime.Time
	}
	if !event.LastTimestamp.IsZero() {
		return event.LastTimestamp.Time
	}
	if !event.FirstTimestamp.IsZero() {
		return event.FirstTimestamp.Time
	}
	return event.CreationTimestamp.Time
}
