package fs

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/registry"
)

const (
	chartRef     = "oci://registry.gitlab.com/1001n/test-chart/charts/test-ngix"
	chartVersion = "0.0.3"

	releaseName = "test-ngix"
	namespace   = "test-ngix"
)

func InstallChart() error {
	ctx := context.Background()
	settings := cli.New()

	registryClient, err := registry.NewClient(
		registry.ClientOptBasicAuth(
			os.Getenv("GITLAB_REGISTRY_USER"),
			os.Getenv("GITLAB_REGISTRY_TOKEN"),
		),
		registry.ClientOptWriter(os.Stderr),
	)
	settings.SetNamespace(namespace)
	// Подключение к Kubernetes через текущий kubeconfig.
	actionConfig := new(action.Configuration)

	err = actionConfig.Init(
		settings.RESTClientGetter(),
		namespace,
		os.Getenv("HELM_DRIVER"),
		log.Printf,
	)
	if err != nil {
		log.Fatal(err)
	}

	install := action.NewInstall(actionConfig)
	install.SetRegistryClient(registryClient)

	install.ReleaseName = releaseName
	install.Namespace = namespace
	install.CreateNamespace = true

	install.Version = chartVersion
	install.Wait = true
	install.WaitForJobs = true
	install.Atomic = true
	install.Timeout = 5 * time.Minute

	// Helm скачает OCI chart во внутренний cache/temp.
	chartPath, err := install.LocateChart(chartRef, settings)
	if err != nil {
		log.Fatalf("pull chart: %v", err)
	}

	ch, err := loader.Load(chartPath)
	if err != nil {
		log.Fatalf("load chart: %v", err)
	}

	values := map[string]interface{}{
		"replicaCount": 2,
	}

	release, err := install.RunWithContext(ctx, ch, values)
	if err != nil {
		log.Fatalf("install release: %v", err)
	}

	fmt.Printf(
		"installed release=%s namespace=%s revision=%d\n",
		release.Name,
		release.Namespace,
		release.Version,
	)
	return nil
}