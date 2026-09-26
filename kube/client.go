package kube

import (
	"fmt"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"

	"gitlab.com/1001n/kapellmeister/config"
)

func RESTClientGetter(opts config.Options) *genericclioptions.ConfigFlags {
	flags := genericclioptions.NewConfigFlags(true)
	flags.Namespace = &opts.Namespace
	flags.KubeConfig = &opts.Kubeconfig
	flags.Context = &opts.Context
	return flags
}

func NewClient(opts config.Options) (kubernetes.Interface, error) {
	restConfig, err := RESTClientGetter(opts).ToRESTConfig()
	if err != nil {
		return nil, fmt.Errorf("create Kubernetes REST config: %w", err)
	}

	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("create Kubernetes client: %w", err)
	}

	return client, nil
}
