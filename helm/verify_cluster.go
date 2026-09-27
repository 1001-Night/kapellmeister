package helm

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"helm.sh/helm/v3/pkg/release"
)

const defaultKubeVersionConstraint = ">= 1.28.0"

func (c *Client) verifyKubernetesVersion(
	rel *release.Release,
) CheckResult {
	check := CheckResult{
		Section: "Cluster",
		Name:    "Kubernetes version",
		Status:  CheckFailed,
	}

	serverVersion, err := c.KubeClient.Discovery().ServerVersion()
	if err != nil {
		check.Detail = fmt.Sprintf("request server version: %v", err)
		return check
	}

	required := defaultKubeVersionConstraint
	if rel.Chart != nil &&
		rel.Chart.Metadata != nil &&
		strings.TrimSpace(rel.Chart.Metadata.KubeVersion) != "" {
		required = rel.Chart.Metadata.KubeVersion
	}

	constraint, err := semver.NewConstraint(required)
	if err != nil {
		check.Detail = fmt.Sprintf("invalid chart kubeVersion %q: %v", required, err)
		return check
	}

	version, err := semver.NewVersion(serverVersion.GitVersion)
	if err != nil {
		check.Detail = fmt.Sprintf(
			"parse server version %q: %v",
			serverVersion.GitVersion,
			err,
		)
		return check
	}

	if !constraint.Check(version) {
		check.Detail = fmt.Sprintf(
			"%s does not satisfy %s",
			serverVersion.GitVersion,
			required,
		)
		return check
	}

	check.Status = CheckPassed
	check.Detail = fmt.Sprintf(
		"%s satisfies %s",
		serverVersion.GitVersion,
		required,
	)
	return check
}

func (c *Client) verifyAPIs(
	objects []unstructured.Unstructured,
) []CheckResult {
	unique := make(map[schema.GroupVersionKind]struct{})
	for _, object := range objects {
		unique[object.GroupVersionKind()] = struct{}{}
	}

	gvks := make([]schema.GroupVersionKind, 0, len(unique))
	for gvk := range unique {
		gvks = append(gvks, gvk)
	}
	sort.Slice(gvks, func(i, j int) bool {
		return gvks[i].String() < gvks[j].String()
	})

	type discoveryResult struct {
		resources *metav1.APIResourceList
		err       error
	}
	discovered := make(map[string]discoveryResult)
	checks := make([]CheckResult, 0, len(gvks))

	for _, gvk := range gvks {
		apiVersion := gvk.GroupVersion().String()
		check := CheckResult{
			Section: "Cluster APIs",
			Name:    fmt.Sprintf("%s %s", apiVersion, gvk.Kind),
			Status:  CheckFailed,
		}

		result, ok := discovered[apiVersion]
		if !ok {
			result.resources, result.err = c.KubeClient.
				Discovery().
				ServerResourcesForGroupVersion(apiVersion)
			discovered[apiVersion] = result
		}

		if result.err != nil {
			check.Detail = fmt.Sprintf("API version unavailable: %v", result.err)
			checks = append(checks, check)
			continue
		}

		for _, resource := range result.resources.APIResources {
			if resource.Kind == gvk.Kind && !strings.Contains(resource.Name, "/") {
				check.Status = CheckPassed
				check.Detail = "available"
				break
			}
		}

		if check.Status == CheckFailed && check.Detail == "" {
			check.Detail = "kind is not served by the cluster"
		}

		checks = append(checks, check)
	}

	return checks
}
