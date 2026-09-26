package helm

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"helm.sh/helm/v3/pkg/release"
)

const warningEventWindow = 15 * time.Minute

type eventTarget struct {
	Namespace string
	Kind      string
	Name      string
}

func (c *Client) verifyEvents(
	ctx context.Context,
	rel *release.Release,
) ([]CheckResult, error) {
	objects, err := manifestObjects(rel.Manifest)
	if err != nil {
		return nil, fmt.Errorf("inspect manifest for events: %w", err)
	}

	targets := make(map[eventTarget]struct{})
	namespaces := make(map[string]struct{})
	for _, object := range objects {
		namespace := object.GetNamespace()
		if namespace == "" && !isClusterScopedKind(object.GetKind()) {
			namespace = c.Options.Namespace
		}
		targets[eventTarget{
			Namespace: namespace,
			Kind:      object.GetKind(),
			Name:      object.GetName(),
		}] = struct{}{}
		if namespace != "" {
			namespaces[namespace] = struct{}{}
		}

		gvk := object.GroupVersionKind()
		if gvk.Group == "apps" && gvk.Kind == "Deployment" {
			if err := c.addDeploymentEventTargets(
				ctx,
				namespace,
				object.GetName(),
				targets,
			); err != nil {
				return []CheckResult{{
					Section: "Events",
					Name:    "Workload events",
					Detail:  err.Error(),
					Status:  CheckFailed,
				}}, nil
			}
		}
	}

	if len(namespaces) == 0 && c.Options.Namespace != "" {
		namespaces[c.Options.Namespace] = struct{}{}
	}

	var warnings []corev1.Event
	for namespace := range namespaces {
		events, err := c.KubeClient.
			CoreV1().
			Events(namespace).
			List(ctx, metav1.ListOptions{})
		if err != nil {
			return []CheckResult{{
				Section: "Events",
				Name:    namespace,
				Detail:  fmt.Sprintf("list events: %v", err),
				Status:  CheckFailed,
			}}, nil
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
		}}, nil
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

	return checks, nil
}

func (c *Client) addDeploymentEventTargets(
	ctx context.Context,
	namespace string,
	name string,
	targets map[eventTarget]struct{},
) error {
	deployment, err := c.KubeClient.
		AppsV1().
		Deployments(namespace).
		Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get Deployment %s/%s: %w", namespace, name, err)
	}
	if deployment.Spec.Selector == nil {
		return fmt.Errorf("Deployment %s/%s has no Pod selector", namespace, name)
	}

	selector, err := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
	if err != nil {
		return fmt.Errorf("parse Deployment %s/%s selector: %w", namespace, name, err)
	}

	replicaSets, err := c.KubeClient.
		AppsV1().
		ReplicaSets(namespace).
		List(ctx, metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return fmt.Errorf("list ReplicaSets for %s/%s: %w", namespace, name, err)
	}

	replicaSetUIDs := make(map[types.UID]struct{})
	for _, replicaSet := range replicaSets.Items {
		if !hasOwnerUID(replicaSet.OwnerReferences, deployment.UID) {
			continue
		}
		replicaSetUIDs[replicaSet.UID] = struct{}{}
		targets[eventTarget{
			Namespace: namespace,
			Kind:      "ReplicaSet",
			Name:      replicaSet.Name,
		}] = struct{}{}
	}

	pods, err := c.KubeClient.
		CoreV1().
		Pods(namespace).
		List(ctx, metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return fmt.Errorf("list Pods for %s/%s: %w", namespace, name, err)
	}

	for _, pod := range pods.Items {
		owned := false
		for replicaSetUID := range replicaSetUIDs {
			if hasOwnerUID(pod.OwnerReferences, replicaSetUID) {
				owned = true
				break
			}
		}
		if !owned {
			continue
		}
		targets[eventTarget{
			Namespace: namespace,
			Kind:      "Pod",
			Name:      pod.Name,
		}] = struct{}{}
	}

	return nil
}

func hasOwnerUID(references []metav1.OwnerReference, uid types.UID) bool {
	for _, reference := range references {
		if reference.UID == uid {
			return true
		}
	}
	return false
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

func isClusterScopedKind(kind string) bool {
	switch kind {
	case "Namespace",
		"Node",
		"PersistentVolume",
		"ClusterRole",
		"ClusterRoleBinding",
		"CustomResourceDefinition",
		"StorageClass",
		"PriorityClass":
		return true
	default:
		return false
	}
}
