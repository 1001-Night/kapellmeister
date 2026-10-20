package helm

import (
	"fmt"
	"io"
	"os"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/registry"
	"k8s.io/client-go/kubernetes"

	"gitlab.com/1001n/kapellmeister/config"
	"gitlab.com/1001n/kapellmeister/kube"
)

type Logger interface {
	io.Writer
	Printf(format string, args ...any)
}

type Client struct {
	ActionConfig *action.Configuration
	Settings     *cli.EnvSettings
	KubeClient   kubernetes.Interface
	Options      config.Options
}

func New(opts config.Options, logger Logger) (*Client, error) {
	settings := cli.New()
	settings.SetNamespace(opts.Namespace)
	settings.KubeConfig = opts.Kubeconfig
	settings.KubeContext = opts.Context

	user := os.Getenv("GITLAB_REGISTRY_USER")
	token := os.Getenv("GITLAB_REGISTRY_TOKEN")

	var regClient *registry.Client
	var err error

	if user != "" && token != "" {
		regClient, err = registry.NewClient(
			registry.ClientOptBasicAuth(user, token),
			registry.ClientOptWriter(logger),
		)
	} else {
		regClient, err = registry.NewClient(
			registry.ClientOptWriter(logger),
		)
	}
	if err != nil {
		return nil, fmt.Errorf("create registry client: %w", err)
	}

	actionConfig := &action.Configuration{RegistryClient: regClient}
	getter, err := kube.RESTClientGetter(opts)
	if err != nil {
		return nil, err
	}
	err = actionConfig.Init(
		getter,
		opts.Namespace,
		os.Getenv("HELM_DRIVER"),
		logger.Printf,
	)
	if err != nil {
		return nil, fmt.Errorf("init action config: %w", err)
	}

	kubeClient, err := actionConfig.KubernetesClientSet()
	if err != nil {
		return nil, fmt.Errorf("create Kubernetes client: %w", err)
	}

	return &Client{
		ActionConfig: actionConfig,
		Settings:     settings,
		KubeClient:   kubeClient,
		Options:      opts,
	}, nil
}
