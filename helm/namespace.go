package helm

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const namespaceOwnerLabel = "kapellmeister.io/owned"

// ensureNamespace создаёт namespace и отмечает его как принадлежащий утилите.
func (c *Client) ensureNamespace(ctx context.Context) (bool, error) {
	_, err := c.KubeClient.CoreV1().Namespaces().Get(
		ctx,
		c.Options.Namespace,
		metav1.GetOptions{},
	)
	if err == nil {
		return false, nil
	}
	if !apierrors.IsNotFound(err) {
		return false, fmt.Errorf("get namespace %q: %w", c.Options.Namespace, err)
	}

	_, err = c.KubeClient.CoreV1().Namespaces().Create(
		ctx,
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: c.Options.Namespace,
				Labels: map[string]string{
					namespaceOwnerLabel: "true",
				},
			},
		},
		metav1.CreateOptions{},
	)
	if apierrors.IsAlreadyExists(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("create namespace %q: %w", c.Options.Namespace, err)
	}

	return true, nil
}

// deleteOwnedNamespace не удаляет namespace, созданный не нашей утилитой.
func (c *Client) deleteOwnedNamespace(ctx context.Context) error {
	namespace, err := c.KubeClient.CoreV1().Namespaces().Get(
		ctx,
		c.Options.Namespace,
		metav1.GetOptions{},
	)
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("get namespace %q: %w", c.Options.Namespace, err)
	}

	if namespace.Labels[namespaceOwnerLabel] != "true" {
		return nil
	}

	err = c.KubeClient.CoreV1().Namespaces().Delete(
		ctx,
		c.Options.Namespace,
		metav1.DeleteOptions{},
	)
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("delete namespace %q: %w", c.Options.Namespace, err)
	}

	return nil
}
