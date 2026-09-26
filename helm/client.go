package helm

import (
	"fmt"
	"log"
	"os"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/registry"
	"k8s.io/client-go/kubernetes"

	"gitlab.com/1001n/kapellmeister/config"
)

type Client struct {
	ActionConfig *action.Configuration
	Settings     *cli.EnvSettings
	Registry     *registry.Client
	KubeClient   kubernetes.Interface
	Options      config.Options
}

func New(opts config.Options) (*Client, error) {
	settings := cli.New()
	settings.SetNamespace(opts.Namespace)

	user := os.Getenv("GITLAB_REGISTRY_USER")
	token := os.Getenv("GITLAB_REGISTRY_TOKEN")

	var regClient *registry.Client
	var err error

	if user != "" && token != "" {
		regClient, err = registry.NewClient(
			registry.ClientOptBasicAuth(user, token),
			registry.ClientOptWriter(os.Stderr),
		)
	} else {
		regClient, err = registry.NewClient(
			registry.ClientOptWriter(os.Stderr),
		)
	}
	if err != nil {
		return nil, fmt.Errorf("create registry client: %w", err)
	}

	actionConfig := new(action.Configuration)
	err = actionConfig.Init(
		settings.RESTClientGetter(),
		opts.Namespace,
		os.Getenv("HELM_DRIVER"),
		log.Printf,
	)
	if err != nil {
		return nil, fmt.Errorf("init action config: %w", err)
	}

	return &Client{
		ActionConfig: actionConfig,
		Settings:     settings,
		Registry:     regClient,
		KubeClient:   kubeClient,
		Options:      opts,
	}, nil
}
