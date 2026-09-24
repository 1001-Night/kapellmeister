package fs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func GetPods() error {
	ctx := context.Background()
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home directory: %w", err)
	}

	kubeconfig := filepath.Join(home, ".kube", "config")

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return fmt.Errorf("build Kubernetes config: %w", err)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("create Kubernetes client: %w", err)
	}

	namespaces, err := client.CoreV1().
		Namespaces().
		List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("list namespaces: %w", err)
	}

	fmt.Println("Namespaces:")
	for _, namespace := range namespaces.Items {
		fmt.Println("-", namespace.Name)
	}

	pods, err := client.CoreV1().
		Pods("local-path-storage").
		List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("list pods in all namespaces: %w", err)
	}

	fmt.Println("\nPods:")
	for _, pod := range pods.Items {
		fmt.Printf(
			"%-25s %-40s %s\n",
			pod.Namespace,
			pod.Name,
			pod.Status.Phase,
		)
	}

	return nil
}