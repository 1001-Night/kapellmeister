package kube

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"gitlab.com/1001n/kapellmeister/config"
)

func RESTClientGetter(opts config.Options) (genericclioptions.RESTClientGetter, error) {
	if len(opts.KubeconfigData) > 0 {
		raw, err := clientcmd.Load(opts.KubeconfigData)
		if err != nil {
			return nil, fmt.Errorf("parse Magnum kubeconfig: %w", err)
		}
		overrides := &clientcmd.ConfigOverrides{
			Context: clientcmdapi.Context{Namespace: opts.Namespace},
		}
		if opts.Timeout > 0 {
			overrides.Timeout = opts.Timeout.String()
		}
		return &memoryClientGetter{
			config: clientcmd.NewNonInteractiveClientConfig(*raw, opts.Context, overrides, nil),
		}, nil
	}

	flags := genericclioptions.NewConfigFlags(true)
	flags.Namespace = &opts.Namespace
	flags.KubeConfig = &opts.Kubeconfig
	flags.Context = &opts.Context
	if opts.Timeout > 0 {
		timeout := opts.Timeout.String()
		flags.Timeout = &timeout
	}
	return flags, nil
}

func NewClient(opts config.Options) (kubernetes.Interface, error) {
	getter, err := RESTClientGetter(opts)
	if err != nil {
		return nil, err
	}
	restConfig, err := getter.ToRESTConfig()
	if err != nil {
		return nil, fmt.Errorf("create Kubernetes REST config: %w", err)
	}

	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("create Kubernetes client: %w", err)
	}

	return client, nil
}

// Даёт Helm и client-go один kubeconfig, не сохраняя ключ на диск.
type memoryClientGetter struct {
	config clientcmd.ClientConfig
}

func (g *memoryClientGetter) ToRESTConfig() (*rest.Config, error) {
	return g.config.ClientConfig()
}

func (g *memoryClientGetter) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return g.config
}

func (g *memoryClientGetter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	config, err := g.ToRESTConfig()
	if err != nil {
		return nil, err
	}
	client, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, err
	}
	return memory.NewMemCacheClient(client), nil
}

func (g *memoryClientGetter) ToRESTMapper() (meta.RESTMapper, error) {
	client, err := g.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}
	return restmapper.NewDeferredDiscoveryRESTMapper(client), nil
}
