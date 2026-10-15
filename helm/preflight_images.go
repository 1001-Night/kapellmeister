package helm

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// preflightImageRegistries проверяет registry образов до изменения релиза.
func preflightImageRegistries(ctx context.Context, manifest string) error {
	registries, err := imageRegistries(manifest)
	if err != nil {
		return fmt.Errorf("find image registries: %w", err)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	for _, registry := range registries {
		endpoint := registryEndpoint(registry)
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return fmt.Errorf("check image registry %q: %w", registry, err)
		}

		response, err := client.Do(request)
		if err != nil {
			return fmt.Errorf("check image registry %q: %w", registry, err)
		}
		response.Body.Close()

		// 401 означает, что registry доступен, но требует авторизацию.
		if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusUnauthorized {
			return fmt.Errorf("check image registry %q: %s", registry, response.Status)
		}
	}

	return nil
}

func imageRegistries(manifest string) ([]string, error) {
	objects, err := manifestObjects(manifest)
	if err != nil {
		return nil, err
	}

	unique := make(map[string]struct{})
	for _, object := range objects {
		path := podSpecPath(object.GetKind())
		if path == nil {
			continue
		}
		for _, field := range []string{"containers", "initContainers", "ephemeralContainers"} {
			containers, _, err := unstructured.NestedSlice(object.Object, append(path, field)...)
			if err != nil {
				return nil, fmt.Errorf("read %s %s: %w", object.GetKind(), field, err)
			}
			for _, item := range containers {
				container, ok := item.(map[string]interface{})
				if !ok {
					return nil, fmt.Errorf("invalid container in %s", object.GetKind())
				}
				image, _ := container["image"].(string)
				if strings.TrimSpace(image) == "" {
					return nil, fmt.Errorf("empty image in %s", object.GetKind())
				}
				unique[registryHost(image)] = struct{}{}
			}
		}
	}

	registries := make([]string, 0, len(unique))
	for registry := range unique {
		registries = append(registries, registry)
	}
	sort.Strings(registries)
	return registries, nil
}

func podSpecPath(kind string) []string {
	switch kind {
	case "Pod":
		return []string{"spec"}
	case "Deployment", "StatefulSet", "DaemonSet", "ReplicaSet", "ReplicationController", "Job":
		return []string{"spec", "template", "spec"}
	case "CronJob":
		return []string{"spec", "jobTemplate", "spec", "template", "spec"}
	default:
		return nil
	}
}

func registryHost(image string) string {
	parts := strings.Split(image, "/")
	if len(parts) > 1 && (strings.ContainsAny(parts[0], ".:") || parts[0] == "localhost") {
		if parts[0] == "docker.io" || parts[0] == "index.docker.io" {
			return "registry-1.docker.io"
		}
		return parts[0]
	}
	return "registry-1.docker.io"
}

func registryEndpoint(host string) string {
	scheme := "https"
	hostname := host
	if name, _, err := net.SplitHostPort(host); err == nil {
		hostname = name
	}
	ip := net.ParseIP(hostname)
	if hostname == "localhost" || ip != nil && ip.IsLoopback() {
		scheme = "http"
	}
	return (&url.URL{Scheme: scheme, Host: host, Path: "/v2/"}).String()
}
